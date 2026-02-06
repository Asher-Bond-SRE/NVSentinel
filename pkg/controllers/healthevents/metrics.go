// Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

package healthevents

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	registerOnce sync.Once

	// quarantineActionsTotal tracks the number of quarantine actions taken.
	quarantineActionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "nvsentinel",
			Subsystem: "quarantine_controller",
			Name:      "actions_total",
			Help:      "Total number of quarantine actions taken by outcome",
		},
		[]string{"node", "outcome"}, // outcome: success, failed, skipped
	)
)

// registerMetrics registers all metrics with the controller-runtime metrics registry.
func registerMetrics() {
	registerOnce.Do(func() {
		metrics.Registry.MustRegister(
			quarantineActionsTotal,
		)
	})
}

// =============================================================================
// Drain Controller Metrics
// =============================================================================

var (
	registerDrainOnce sync.Once

	// drainActionsTotal tracks the number of drain actions taken.
	drainActionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "nvsentinel",
			Subsystem: "drain_controller",
			Name:      "actions_total",
			Help:      "Total number of drain actions taken by outcome",
		},
		[]string{"node", "outcome"}, // outcome: evicted, failed, skipped, completed
	)
)

// registerDrainMetrics registers drain controller metrics.
func registerDrainMetrics() {
	registerDrainOnce.Do(func() {
		metrics.Registry.MustRegister(
			drainActionsTotal,
		)
	})
}

// =============================================================================
// TTL Controller Metrics
// =============================================================================

var (
	registerTTLOnce sync.Once

	// ttlDeletionsTotal tracks the number of events deleted by TTL.
	ttlDeletionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "nvsentinel",
			Subsystem: "ttl_controller",
			Name:      "deletions_total",
			Help:      "Total number of HealthEvents deleted by TTL controller",
		},
		[]string{"node", "phase"},
	)
)

// registerTTLMetrics registers TTL controller metrics.
func registerTTLMetrics() {
	registerTTLOnce.Do(func() {
		metrics.Registry.MustRegister(
			ttlDeletionsTotal,
		)
	})
}

// =============================================================================
// Remediation Controller Metrics
// =============================================================================

var (
	registerRemediationOnce sync.Once

	// remediationActionsTotal tracks successful remediation actions.
	remediationActionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "nvsentinel",
			Subsystem: "remediation_controller",
			Name:      "actions_total",
			Help:      "Total number of remediation actions executed",
		},
		[]string{"node", "strategy"},
	)

	// remediationFailuresTotal tracks failed remediation attempts.
	remediationFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "nvsentinel",
			Subsystem: "remediation_controller",
			Name:      "failures_total",
			Help:      "Total number of failed remediation attempts",
		},
		[]string{"node", "strategy"},
	)
)

// registerRemediationMetrics registers remediation controller metrics.
func registerRemediationMetrics() {
	registerRemediationOnce.Do(func() {
		metrics.Registry.MustRegister(
			remediationActionsTotal,
			remediationFailuresTotal,
		)
	})
}
