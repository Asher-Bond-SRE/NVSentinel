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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultGangIDLabel is the default label key for gang identification.
	DefaultGangIDLabel = "app.kubernetes.io/gang-id"

	// DefaultGangSizeLabel is the default label key for expected gang size.
	DefaultGangSizeLabel = "app.kubernetes.io/gang-size"
)

// LabelDiscovererConfig contains configuration for label-based gang discovery.
type LabelDiscovererConfig struct {
	// GangIDLabel is the label key that identifies the gang.
	GangIDLabel string

	// GangSizeLabel is the label key that specifies the expected gang size.
	GangSizeLabel string
}

// DefaultLabelDiscovererConfig returns the default label discoverer configuration.
func DefaultLabelDiscovererConfig() LabelDiscovererConfig {
	return LabelDiscovererConfig{
		GangIDLabel:   DefaultGangIDLabel,
		GangSizeLabel: DefaultGangSizeLabel,
	}
}

// LabelDiscoverer discovers gang members using configurable labels.
// This is useful for custom schedulers or standard Kubernetes deployments
// that use labels for gang identification.
type LabelDiscoverer struct {
	kubeClient kubernetes.Interface
	config     LabelDiscovererConfig
}

// NewLabelDiscoverer creates a new label-based gang discoverer.
func NewLabelDiscoverer(kubeClient kubernetes.Interface, config LabelDiscovererConfig) *LabelDiscoverer {
	if config.GangIDLabel == "" {
		config.GangIDLabel = DefaultGangIDLabel
	}

	if config.GangSizeLabel == "" {
		config.GangSizeLabel = DefaultGangSizeLabel
	}

	return &LabelDiscoverer{
		kubeClient: kubeClient,
		config:     config,
	}
}

// Name returns the discoverer name.
func (l *LabelDiscoverer) Name() string {
	return "labels"
}

// CanHandle returns true if the pod has the configured gang ID label.
func (l *LabelDiscoverer) CanHandle(pod *corev1.Pod) bool {
	if pod.Labels == nil {
		return false
	}

	_, ok := pod.Labels[l.config.GangIDLabel]

	return ok
}

// ExtractGangID extracts the gang identifier from a pod's labels.
func (l *LabelDiscoverer) ExtractGangID(pod *corev1.Pod) string {
	if pod.Labels == nil {
		return ""
	}

	gangID := pod.Labels[l.config.GangIDLabel]
	if gangID == "" {
		return ""
	}

	return fmt.Sprintf("labels-%s-%s", pod.Namespace, gangID)
}

// DiscoverPeers finds all pods with the same gang ID label.
func (l *LabelDiscoverer) DiscoverPeers(ctx context.Context, pod *corev1.Pod) (*GangInfo, error) {
	if !l.CanHandle(pod) {
		return nil, nil
	}

	gangLabelValue := pod.Labels[l.config.GangIDLabel]
	gangID := l.ExtractGangID(pod)

	slog.Debug("Discovering gang by labels",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"gangIdLabel", l.config.GangIDLabel,
		"gangLabelValue", gangLabelValue,
		"gangID", gangID)

	// Get expected size from pod label if present
	expectedCount := 0
	if sizeStr := pod.Labels[l.config.GangSizeLabel]; sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
			expectedCount = size
		}
	}

	// List all pods with the same gang ID label in the namespace
	labelSelector := fmt.Sprintf("%s=%s", l.config.GangIDLabel, gangLabelValue)

	pods, err := l.kubeClient.CoreV1().Pods(pod.Namespace).List(ctx, metav1.ListOptions{
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

	// Use discovered count if size label was not present
	if expectedCount == 0 {
		expectedCount = len(peers)
	}

	slog.Info("Discovered gang by labels",
		"gangID", gangID,
		"gangLabelValue", gangLabelValue,
		"expectedCount", expectedCount,
		"discoveredPeers", len(peers))

	return &GangInfo{
		GangID:           gangID,
		ExpectedMinCount: expectedCount,
		Peers:            peers,
	}, nil
}
