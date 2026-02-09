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
	"fmt"

	"github.com/nvidia/nvsentinel/preflight/pkg/gang/coordinator"
	"github.com/nvidia/nvsentinel/preflight/pkg/gang/discoverer"
	"github.com/nvidia/nvsentinel/preflight/pkg/gang/types"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Scheduler identifies which gang scheduler to use.
type Scheduler string

const (
	SchedulerKubernetes Scheduler = "kubernetes" // K8s 1.35+ native (Workload API)
	SchedulerVolcano    Scheduler = "volcano"    // Volcano PodGroup
)

// Re-export types for convenience.
type (
	PeerInfo          = types.PeerInfo
	GangInfo          = types.GangInfo
	GangDiscoverer    = types.GangDiscoverer
	Coordinator       = coordinator.Coordinator
	CoordinatorConfig = coordinator.CoordinatorConfig
)

// Re-export coordinator functions.
var (
	ConfigMapName            = coordinator.ConfigMapName
	NewCoordinator           = coordinator.NewCoordinator
	DefaultCoordinatorConfig = coordinator.DefaultCoordinatorConfig
	ParsePeers               = coordinator.ParsePeers
	GetRank                  = coordinator.GetRank
)

// NewDiscoverer creates a gang discoverer for the specified scheduler.
func NewDiscoverer(
	scheduler Scheduler,
	kubeClient kubernetes.Interface,
	dynamicClient dynamic.Interface,
) (GangDiscoverer, error) {
	switch scheduler {
	case SchedulerKubernetes:
		return discoverer.NewWorkloadRefDiscoverer(kubeClient), nil
	case SchedulerVolcano:
		return discoverer.NewVolcanoDiscoverer(kubeClient, dynamicClient), nil
	default:
		return nil, fmt.Errorf("unknown scheduler: %q (valid: kubernetes, volcano)", scheduler)
	}
}
