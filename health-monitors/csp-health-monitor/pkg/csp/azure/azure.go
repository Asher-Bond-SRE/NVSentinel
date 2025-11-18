package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	pb "github.com/nvidia/nvsentinel/data-models/pkg/protos"
	"github.com/nvidia/nvsentinel/health-monitors/csp-health-monitor/pkg/config"
	"github.com/nvidia/nvsentinel/health-monitors/csp-health-monitor/pkg/datastore"
	eventpkg "github.com/nvidia/nvsentinel/health-monitors/csp-health-monitor/pkg/event"
	"github.com/nvidia/nvsentinel/health-monitors/csp-health-monitor/pkg/metrics"
	"github.com/nvidia/nvsentinel/health-monitors/csp-health-monitor/pkg/model"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client encapsulates all state required to poll Azure for
// maintenance events and forward them to the main pipeline.
type Client struct {
	config                config.AzureConfig
	VirtualMachinesClient *armcompute.VirtualMachinesClient
	k8sClient             kubernetes.Interface
	normalizer            eventpkg.Normalizer
	clusterName           string
	kubeconfigPath        string
	store                 datastore.Store
}

// NewClient builds and initialises a new Azure monitoring Client.
func NewClient(
	ctx context.Context,
	cfg config.AzureConfig,
	clusterName string,
	kubeconfigPath string,
	store datastore.Store,
) (*Client, error) {
	// Get the Azure subscription ID from config or IMDS
	subscriptionID, err := getSubscriptionID(cfg)
	if err != nil {
		return nil, err
	}

	// Create an Azure client
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Azure credential: %w", err)
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Azure client: %w", err)
	}

	slog.Info("Successfully initialized Azure VM client", "subscriptionID", subscriptionID)

	// Initialize Kubernetes client
	var k8sClient kubernetes.Interface
	var k8sRestConfig *rest.Config

	if kubeconfigPath != "" {
		slog.Info("Azure Client: Using kubeconfig from path", "path", kubeconfigPath)
		k8sRestConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		slog.Info("Azure Client: KubeconfigPath not specified, attempting in-cluster config")
		k8sRestConfig, err = rest.InClusterConfig()
	}

	if err != nil {
		metrics.CSPMonitorErrors.WithLabelValues(string(model.CSPAzure), "k8s_config_error").Inc()
		return nil, fmt.Errorf("failed to load Kubernetes config: %w", err)
	}

	k8sClient, err = kubernetes.NewForConfig(k8sRestConfig)
	if err != nil {
		metrics.CSPMonitorErrors.WithLabelValues(string(model.CSPAzure), "k8s_client_error").Inc()
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	slog.Info("Azure Client: Successfully initialized Kubernetes client")

	// Create normalizer
	normalizer, err := eventpkg.GetNormalizer(model.CSPAzure)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure normalizer: %w", err)
	}

	return &Client{
		config:                cfg,
		VirtualMachinesClient: vmClient,
		k8sClient:             k8sClient,
		normalizer:            normalizer,
		clusterName:           clusterName,
		kubeconfigPath:        kubeconfigPath,
		store:                 store,
	}, nil
}

func (c *Client) GetName() model.CSP {
	return model.CSPAzure
}

func (c *Client) StartMonitoring(ctx context.Context, eventChan chan<- model.MaintenanceEvent) error {
	slog.Info("Starting Azure VM maintenance monitoring",
		"intervalSeconds", c.config.PollingIntervalSeconds)

	// Perform initial poll immediately
	if ctx.Err() == nil {
		c.pollForMaintenanceEvents(ctx, eventChan)
	} else {
		slog.Info("Azure monitoring not starting initial poll due to context cancellation.")
		return ctx.Err()
	}

	ticker := time.NewTicker(time.Duration(c.config.PollingIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Azure monitoring stopping due to context cancellation.")
			return ctx.Err()
		case <-ticker.C:
			c.pollForMaintenanceEvents(ctx, eventChan)
		}
	}
}

// pollForMaintenanceEvents checks all cluster nodes for Azure maintenance events in parallel.
func (c *Client) pollForMaintenanceEvents(ctx context.Context, eventChan chan<- model.MaintenanceEvent) {
	pollStart := time.Now()

	defer func() {
		metrics.CSPPollingDuration.WithLabelValues(string(model.CSPAzure)).Observe(time.Since(pollStart).Seconds())
	}()

	slog.Debug("Polling Azure for VM maintenance events")

	// List all nodes in the cluster
	nodeList, err := c.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		metrics.CSPAPIErrors.WithLabelValues(string(model.CSPAzure), "list_nodes_error").Inc()
		slog.Error("Failed to list Kubernetes nodes", "error", err)
		return
	}

	slog.Debug("Found nodes to check for maintenance events", "count", len(nodeList.Items))

	// Check each node for maintenance events in parallel
	var wg sync.WaitGroup
	for _, node := range nodeList.Items {
		// Skip nodes without a provider ID
		if node.Spec.ProviderID == "" {
			slog.Debug("Skipping node without provider ID", "node", node.Name)
			continue
		}

		wg.Add(1)
		go func(node v1.Node) {
			defer wg.Done()

			// Parse the Azure provider ID
			resourceGroup, vmName, err := parseAzureProviderID(node.Spec.ProviderID)
			if err != nil {
				slog.Warn("Failed to parse Azure provider ID",
					"node", node.Name,
					"providerID", node.Spec.ProviderID,
					"error", err)
				return
			}

			// Get the VM instance view from Azure
			instanceViewResp, err := c.VirtualMachinesClient.InstanceView(ctx, resourceGroup, vmName, nil)
			if err != nil {
				metrics.CSPAPIErrors.WithLabelValues(string(model.CSPAzure), "instance_view_error").Inc()
				slog.Error("Failed to get Azure VM instance view",
					"node", node.Name,
					"resourceGroup", resourceGroup,
					"vmName", vmName,
					"error", err)
				return
			}

			// Check if there's a maintenance event
			if instanceViewResp.MaintenanceRedeployStatus != nil {
				metrics.CSPEventsReceived.WithLabelValues(string(model.CSPAzure)).Inc()

				slog.Info("Detected Azure maintenance event",
					"node", node.Name,
					"resourceGroup", resourceGroup,
					"vmName", vmName)

				// Create and send the maintenance event
				event := c.createMaintenanceEvent(
					&node,
					resourceGroup,
					vmName,
					instanceViewResp.MaintenanceRedeployStatus,
				)

				// Send the event to the channel
				select {
				case eventChan <- event:
					slog.Debug("Sent maintenance event to channel",
						"eventID", event.EventID,
						"node", event.NodeName)
				case <-ctx.Done():
					slog.Info("Context cancelled while sending event")
					return
				}
			}
		}(node)
	}

	// Wait for all node checks to complete
	wg.Wait()

	slog.Debug("Completed Azure maintenance event poll")
}

// createMaintenanceEvent creates a normalized maintenance event from Azure data.
func (c *Client) createMaintenanceEvent(
	node *v1.Node,
	resourceGroup string,
	vmName string,
	maintenanceStatus *armcompute.MaintenanceRedeployStatus,
) model.MaintenanceEvent {
	now := time.Now().UTC()

	// Generate a unique event ID based on the VM and timestamp
	eventID := fmt.Sprintf("azure-%s-%s-%d", resourceGroup, vmName, now.Unix())

	// Create metadata map
	metadata := map[string]string{
		"resourceGroup": resourceGroup,
		"vmName":        vmName,
		"providerID":    node.Spec.ProviderID,
	}

	// Add maintenance window information if available
	if maintenanceStatus.MaintenanceWindowStartTime != nil {
		metadata["maintenanceWindowStartTime"] = maintenanceStatus.MaintenanceWindowStartTime.Format(time.RFC3339)
	}
	if maintenanceStatus.MaintenanceWindowEndTime != nil {
		metadata["maintenanceWindowEndTime"] = maintenanceStatus.MaintenanceWindowEndTime.Format(time.RFC3339)
	}
	if maintenanceStatus.LastOperationResultCode != nil {
		metadata["lastOperationResultCode"] = string(*maintenanceStatus.LastOperationResultCode)
	}
	if maintenanceStatus.LastOperationMessage != nil {
		metadata["lastOperationMessage"] = *maintenanceStatus.LastOperationMessage
	}

	// Determine status based on maintenance information
	status := model.StatusDetected
	cspStatus := model.CSPStatusUnknown
	if maintenanceStatus.IsCustomerInitiatedMaintenanceAllowed != nil && *maintenanceStatus.IsCustomerInitiatedMaintenanceAllowed {
		cspStatus = model.CSPStatusPending
	}

	// Extract scheduled times
	var scheduledStartTime, scheduledEndTime *time.Time
	if maintenanceStatus.MaintenanceWindowStartTime != nil {
		scheduledStartTime = maintenanceStatus.MaintenanceWindowStartTime
	}
	if maintenanceStatus.MaintenanceWindowEndTime != nil {
		scheduledEndTime = maintenanceStatus.MaintenanceWindowEndTime
	}

	event := model.MaintenanceEvent{
		EventID:                eventID,
		CSP:                    model.CSPAzure,
		ClusterName:            c.clusterName,
		ResourceType:           "VirtualMachine",
		ResourceID:             fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", "unknown", resourceGroup, vmName),
		MaintenanceType:        model.TypeScheduled,
		Status:                 status,
		CSPStatus:              cspStatus,
		ScheduledStartTime:     scheduledStartTime,
		ScheduledEndTime:       scheduledEndTime,
		EventReceivedTimestamp: now,
		LastUpdatedTimestamp:   now,
		RecommendedAction:      pb.RecommendedAction_RESTART_VM.String(),
		Metadata:               metadata,
		NodeName:               node.Name,
	}

	return event
}

func getSubscriptionID(cfg config.AzureConfig) (string, error) {
	if cfg.SubscriptionID != "" {
		return cfg.SubscriptionID, nil
	}

	// pulled from https://github.com/Microsoft/azureimds/blob/master/imdssample.go
	var PTransport = &http.Transport{Proxy: nil}

	client := http.Client{Transport: PTransport}

	req, _ := http.NewRequest("GET", "http://169.254.169.254/metadata/instance", nil)
	req.Header.Add("Metadata", "True")

	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", "2021-02-01")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// now that we have the response get the subscription ID from it
	var result struct {
		Compute struct {
			SubscriptionID string `json:"subscriptionId"`
		} `json:"compute"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode IMDS response: %v", err)
	}
	return result.Compute.SubscriptionID, nil
}

// parseAzureProviderID parses the provider ID to extract the resource group and VM name.
// Example provider ID: azure:///subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.Compute/virtualMachines/<vm-name>
func parseAzureProviderID(providerID string) (string, string, error) {
	parts := strings.Split(providerID, "/")
	if len(parts) < 9 {
		return "", "", fmt.Errorf("invalid provider ID format: %s", providerID)
	}

	// Extract resource group and VM name from the provider ID
	// Format: azure:///subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.Compute/virtualMachines/<vm-name>
	var resourceGroup, vmName string
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			resourceGroup = parts[i+1]
		}
		if part == "virtualMachines" && i+1 < len(parts) {
			vmName = parts[i+1]
		}
	}

	if resourceGroup == "" || vmName == "" {
		return "", "", fmt.Errorf("could not extract resource group or VM name from provider ID: %s", providerID)
	}

	return resourceGroup, vmName, nil
}
