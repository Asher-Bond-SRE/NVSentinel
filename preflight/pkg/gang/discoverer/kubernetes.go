// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discoverer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nvidia/nvsentinel/preflight/pkg/gang/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// WorkloadRefDiscoverer discovers gang members using K8s 1.35+ native workloadRef.
// Pods are linked to Workloads via spec.workloadRef:
//
//	spec:
//	  workloadRef:
//	    name: training-job-workload
//	    podGroup: workers
type WorkloadRefDiscoverer struct {
	kubeClient kubernetes.Interface
}

// NewWorkloadRefDiscoverer creates a new workloadRef gang discoverer.
func NewWorkloadRefDiscoverer(kubeClient kubernetes.Interface) *WorkloadRefDiscoverer {
	return &WorkloadRefDiscoverer{
		kubeClient: kubeClient,
	}
}

func (w *WorkloadRefDiscoverer) Name() string {
	return "kubernetes"
}

// CanHandle returns true if the pod has a workloadRef.
func (w *WorkloadRefDiscoverer) CanHandle(pod *corev1.Pod) bool {
	// Check if pod has workloadRef in spec
	// Note: As of K8s 1.35, workloadRef is a new field in PodSpec
	return getWorkloadRefName(pod) != ""
}

// ExtractGangID extracts the gang identifier from a pod's workloadRef.
func (w *WorkloadRefDiscoverer) ExtractGangID(pod *corev1.Pod) string {
	workloadName := getWorkloadRefName(pod)
	podGroup := getWorkloadRefPodGroup(pod)

	if workloadName == "" {
		return ""
	}

	if podGroup != "" {
		return fmt.Sprintf("kubernetes-%s-%s-%s", pod.Namespace, workloadName, podGroup)
	}

	return fmt.Sprintf("kubernetes-%s-%s", pod.Namespace, workloadName)
}

// DiscoverPeers finds all pods with the same workloadRef.
func (w *WorkloadRefDiscoverer) DiscoverPeers(ctx context.Context, pod *corev1.Pod) (*types.GangInfo, error) {
	if !w.CanHandle(pod) {
		return nil, nil
	}

	workloadName := getWorkloadRefName(pod)
	podGroup := getWorkloadRefPodGroup(pod)
	gangID := w.ExtractGangID(pod)

	slog.Debug("Discovering workloadRef gang",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"workload", workloadName,
		"podGroup", podGroup,
		"gangID", gangID)

	// Get expected minCount from Workload CRD
	expectedMinCount, err := w.getWorkloadMinCount(ctx, pod.Namespace, workloadName, podGroup)
	if err != nil {
		slog.Warn("Failed to get Workload minCount, will use discovered pod count",
			"workload", workloadName,
			"error", err)
	}

	// List all pods in the namespace and filter by workloadRef
	pods, err := w.kubeClient.CoreV1().Pods(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", pod.Namespace, err)
	}

	var peers []types.PeerInfo

	for i := range pods.Items {
		p := &pods.Items[i]

		// Check if pod has same workloadRef
		pWorkloadName := getWorkloadRefName(p)
		pPodGroup := getWorkloadRefPodGroup(p)

		if pWorkloadName != workloadName {
			continue
		}

		// If we're filtering by podGroup, check it matches
		if podGroup != "" && pPodGroup != podGroup {
			continue
		}

		// Skip pods that are not running or pending
		if p.Status.Phase != corev1.PodRunning && p.Status.Phase != corev1.PodPending {
			continue
		}

		peers = append(peers, types.PeerInfo{
			PodName:   p.Name,
			PodIP:     p.Status.PodIP,
			NodeName:  p.Spec.NodeName,
			Namespace: p.Namespace,
		})
	}

	if len(peers) == 0 {
		return nil, nil
	}

	// Use discovered count if Workload lookup failed
	if expectedMinCount == 0 {
		expectedMinCount = len(peers)
	}

	slog.Info("Discovered workloadRef gang",
		"gangID", gangID,
		"workload", workloadName,
		"podGroup", podGroup,
		"expectedMinCount", expectedMinCount,
		"discoveredPeers", len(peers))

	return &types.GangInfo{
		GangID:           gangID,
		ExpectedMinCount: expectedMinCount,
		Peers:            peers,
	}, nil
}

// getWorkloadMinCount retrieves the minCount from a Workload's podGroup gang policy.
func (w *WorkloadRefDiscoverer) getWorkloadMinCount(ctx context.Context, namespace, name, podGroup string) (int, error) {
	workload, err := w.kubeClient.SchedulingV1alpha1().Workloads(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get Workload %s/%s: %w", namespace, name, err)
	}

	for _, pg := range workload.Spec.PodGroups {
		// If podGroup specified, match it; otherwise take first one
		if podGroup != "" && pg.Name != podGroup {
			continue
		}

		if pg.Policy.Gang != nil {
			return int(pg.Policy.Gang.MinCount), nil
		}
	}

	return 0, nil
}

// getWorkloadRefName extracts workloadRef.name from a pod.
// Returns empty string if not present.
func getWorkloadRefName(pod *corev1.Pod) string {
	if pod.Spec.WorkloadRef != nil {
		return pod.Spec.WorkloadRef.Name
	}

	return ""
}

// getWorkloadRefPodGroup extracts workloadRef.podGroup from a pod.
// Returns empty string if not present.
func getWorkloadRefPodGroup(pod *corev1.Pod) string {
	if pod.Spec.WorkloadRef != nil {
		return pod.Spec.WorkloadRef.PodGroup
	}

	return ""
}
