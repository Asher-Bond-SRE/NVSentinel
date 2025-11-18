package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/nvidia/nvsentinel/health-monitors/csp-health-monitor/pkg/config"
	"github.com/nvidia/nvsentinel/health-monitors/csp-health-monitor/pkg/model"
)

// Client encapsulates all state required to poll Azure for
// maintenance events and forward them to the main pipeline.
type Client struct {
	VirtualMachinesClient *armcompute.VirtualMachinesClient
}

// NewClient builds and initialises a new Azure monitoring Client.
func NewClient(cfg config.AzureConfig) (*Client, error) {
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

	return &Client{
		VirtualMachinesClient: vmClient,
	}, nil
}

func (c *Client) GetName() model.CSP {
	return model.CSPAzure
}

func (c *Client) StartMonitoring(ctx context.Context, eventChan chan<- model.MaintenanceEvent) error {
	return fmt.Errorf("StartMonitoring is not implemented for Azure")
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
