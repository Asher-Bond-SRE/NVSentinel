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

package gang

import (
	"context"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	// VolcanoPodGroupAnnotation is the annotation used by Volcano to identify pod groups.
	VolcanoPodGroupAnnotation = "volcano.sh/pod-group"

	// VolcanoQueueNameAnnotation is the annotation for the Volcano queue name.
	VolcanoQueueNameAnnotation = "volcano.sh/queue-name"
)

// VolcanoPodGroupGVR is the GroupVersionResource for Volcano PodGroups.
var VolcanoPodGroupGVR = schema.GroupVersionResource{
	Group:    "scheduling.volcano.sh",
	Version:  "v1beta1",
	Resource: "podgroups",
}

// VolcanoDiscoverer discovers gang members using Volcano scheduler's PodGroup.
type VolcanoDiscoverer struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewVolcanoDiscoverer creates a new Volcano gang discoverer.
func NewVolcanoDiscoverer(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) *VolcanoDiscoverer {
	return &VolcanoDiscoverer{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
	}
}

// Name returns the discoverer name.
func (v *VolcanoDiscoverer) Name() string {
	return "volcano"
}

// CanHandle returns true if the pod has a Volcano pod-group annotation.
func (v *VolcanoDiscoverer) CanHandle(pod *corev1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}

	_, ok := pod.Annotations[VolcanoPodGroupAnnotation]

	return ok
}

// ExtractGangID extracts the gang identifier from a pod's Volcano annotation.
func (v *VolcanoDiscoverer) ExtractGangID(pod *corev1.Pod) string {
	if pod.Annotations == nil {
		return ""
	}

	podGroupName := pod.Annotations[VolcanoPodGroupAnnotation]
	if podGroupName == "" {
		return ""
	}

	return fmt.Sprintf("volcano-%s-%s", pod.Namespace, podGroupName)
}

// DiscoverPeers finds all pods in the same Volcano PodGroup.
func (v *VolcanoDiscoverer) DiscoverPeers(ctx context.Context, pod *corev1.Pod) (*GangInfo, error) {
	if !v.CanHandle(pod) {
		return nil, nil
	}

	podGroupName := pod.Annotations[VolcanoPodGroupAnnotation]
	gangID := v.ExtractGangID(pod)

	slog.Debug("Discovering Volcano gang",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"podGroup", podGroupName,
		"gangID", gangID)

	// Get expected size from PodGroup resource
	expectedCount, err := v.getPodGroupMinMember(ctx, pod.Namespace, podGroupName)
	if err != nil {
		slog.Warn("Failed to get PodGroup minMember, will use discovered pod count",
			"podGroup", podGroupName,
			"error", err)
	}

	// List all pods with the same pod-group annotation in the namespace
	pods, err := v.kubeClient.CoreV1().Pods(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", pod.Namespace, err)
	}

	var peers []PeerInfo

	for i := range pods.Items {
		p := &pods.Items[i]

		if p.Annotations == nil {
			continue
		}

		if p.Annotations[VolcanoPodGroupAnnotation] != podGroupName {
			continue
		}

		// Skip pods that are not running or pending
		if p.Status.Phase != corev1.PodRunning && p.Status.Phase != corev1.PodPending {
			continue
		}

		peers = append(peers, PeerInfo{
			PodName:   p.Name,
			PodIP:     p.Status.PodIP,
			NodeName:  p.Spec.NodeName,
			Namespace: p.Namespace,
		})
	}

	if len(peers) == 0 {
		return nil, nil
	}

	// Use discovered count if PodGroup lookup failed
	if expectedCount == 0 {
		expectedCount = len(peers)
	}

	slog.Info("Discovered Volcano gang",
		"gangID", gangID,
		"podGroup", podGroupName,
		"expectedCount", expectedCount,
		"discoveredPeers", len(peers))

	return &GangInfo{
		GangID:           gangID,
		ExpectedMinCount: expectedCount,
		Peers:            peers,
	}, nil
}

// getPodGroupMinMember retrieves the minMember field from a Volcano PodGroup.
func (v *VolcanoDiscoverer) getPodGroupMinMember(ctx context.Context, namespace, name string) (int, error) {
	if v.dynamicClient == nil {
		return 0, fmt.Errorf("dynamic client not configured")
	}

	podGroup, err := v.dynamicClient.Resource(VolcanoPodGroupGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get PodGroup %s/%s: %w", namespace, name, err)
	}

	minMember, found, err := unstructured.NestedInt64(podGroup.Object, "spec", "minMember")
	if err != nil {
		return 0, fmt.Errorf("failed to extract minMember from PodGroup: %w", err)
	}

	if !found {
		return 0, nil
	}

	return int(minMember), nil
}
