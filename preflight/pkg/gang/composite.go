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

	corev1 "k8s.io/api/core/v1"
)

// CompositeGangDiscoverer tries multiple discoverers in order until one can handle the pod.
// First discoverer that returns true from CanHandle() wins.
type CompositeGangDiscoverer struct {
	discoverers []GangDiscoverer
}

// NewCompositeGangDiscoverer creates a composite discoverer that tries each provided
// discoverer in order until one returns a gang ID.
func NewCompositeGangDiscoverer(discoverers ...GangDiscoverer) *CompositeGangDiscoverer {
	return &CompositeGangDiscoverer{discoverers: discoverers}
}

// Name returns "composite".
func (c *CompositeGangDiscoverer) Name() string {
	return "composite"
}

// CanHandle returns true if any of the underlying discoverers can handle the pod.
func (c *CompositeGangDiscoverer) CanHandle(pod *corev1.Pod) bool {
	for _, d := range c.discoverers {
		if d.CanHandle(pod) {
			return true
		}
	}

	return false
}

// ExtractGangID tries each discoverer in order and returns the first non-empty gang ID.
func (c *CompositeGangDiscoverer) ExtractGangID(pod *corev1.Pod) string {
	for _, d := range c.discoverers {
		if d.CanHandle(pod) {
			if gangID := d.ExtractGangID(pod); gangID != "" {
				return gangID
			}
		}
	}

	return ""
}

// DiscoverPeers uses the first discoverer that can handle the pod to find peers.
func (c *CompositeGangDiscoverer) DiscoverPeers(ctx context.Context, pod *corev1.Pod) (*GangInfo, error) {
	for _, d := range c.discoverers {
		if d.CanHandle(pod) {
			return d.DiscoverPeers(ctx, pod)
		}
	}

	return nil, nil // No discoverer can handle this pod, it's a singleton
}

// ActiveDiscoverer returns the discoverer that would handle the given pod, or nil.
func (c *CompositeGangDiscoverer) ActiveDiscoverer(pod *corev1.Pod) GangDiscoverer {
	for _, d := range c.discoverers {
		if d.CanHandle(pod) {
			return d
		}
	}

	return nil
}
