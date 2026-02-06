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
	"k8s.io/client-go/kubernetes"
)

const (
	// KueueWorkloadNameLabel is the label used by Kueue to identify workloads.
	KueueWorkloadNameLabel = "kueue.x-k8s.io/workload-name"
)

// KueueDiscoverer discovers gang members using Kueue's workload-name label.
type KueueDiscoverer struct {
	kubeClient kubernetes.Interface
}

// NewKueueDiscoverer creates a new Kueue gang discoverer.
func NewKueueDiscoverer(kubeClient kubernetes.Interface) *KueueDiscoverer {
	return &KueueDiscoverer{
		kubeClient: kubeClient,
	}
}

// Name returns the discoverer name.
func (k *KueueDiscoverer) Name() string {
	return "kueue"
}

// CanHandle returns true if the pod has a Kueue workload-name label.
func (k *KueueDiscoverer) CanHandle(pod *corev1.Pod) bool {
	if pod.Labels == nil {
		return false
	}

	_, ok := pod.Labels[KueueWorkloadNameLabel]

	return ok
}

// ExtractGangID extracts the gang identifier from a pod's Kueue label.
func (k *KueueDiscoverer) ExtractGangID(pod *corev1.Pod) string {
	if pod.Labels == nil {
		return ""
	}

	workloadName := pod.Labels[KueueWorkloadNameLabel]
	if workloadName == "" {
		return ""
	}

	return fmt.Sprintf("kueue-%s-%s", pod.Namespace, workloadName)
}

// DiscoverPeers finds all pods with the same Kueue workload name.
func (k *KueueDiscoverer) DiscoverPeers(ctx context.Context, pod *corev1.Pod) (*GangInfo, error) {
	if !k.CanHandle(pod) {
		return nil, nil
	}

	workloadName := pod.Labels[KueueWorkloadNameLabel]
	gangID := k.ExtractGangID(pod)

	slog.Debug("Discovering Kueue gang",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"workloadName", workloadName,
		"gangID", gangID)

	// List all pods with the same workload-name label in the namespace
	labelSelector := fmt.Sprintf("%s=%s", KueueWorkloadNameLabel, workloadName)

	pods, err := k.kubeClient.CoreV1().Pods(pod.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods with selector %s: %w", labelSelector, err)
	}

	var peers []PeerInfo

	for i := range pods.Items {
		p := &pods.Items[i]

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

	slog.Info("Discovered Kueue gang",
		"gangID", gangID,
		"workloadName", workloadName,
		"discoveredPeers", len(peers))

	return &GangInfo{
		GangID:           gangID,
		ExpectedMinCount: len(peers), // Kueue doesn't directly expose expected size on pods
		Peers:            peers,
	}, nil
}
