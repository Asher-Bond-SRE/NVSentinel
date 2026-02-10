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

package coordinator

import (
	"strings"
	"testing"

	"github.com/nvidia/nvsentinel/preflight/pkg/gang/types"
)

func TestConfigMapName(t *testing.T) {
	tests := []struct {
		name      string
		gangID    string
		wantLen   int // -1 means check exact match
		wantExact string
	}{
		{
			name:      "short gang ID",
			gangID:    "volcano-ns-pg",
			wantExact: "preflight-volcano-ns-pg",
		},
		{
			name:    "long gang ID gets truncated with hash",
			gangID:  "volcano-very-long-namespace-name-that-exceeds-limits-podgroup-name-also-long",
			wantLen: MaxLength,
		},
		{
			name:      "special chars replaced",
			gangID:    "kai-ns/pod_group",
			wantExact: "preflight-kai-ns-pod-group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConfigMapName(tt.gangID)

			if tt.wantExact != "" && got != tt.wantExact {
				t.Errorf("ConfigMapName() = %q, want %q", got, tt.wantExact)
			}

			if tt.wantLen > 0 && len(got) > tt.wantLen {
				t.Errorf("ConfigMapName() len = %d, want <= %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestSanitizeLabelValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantLen int
	}{
		{
			name:    "short value unchanged length",
			value:   "volcano-ns-pg",
			wantLen: 13,
		},
		{
			name:    "long value truncated to 63",
			value:   strings.Repeat("a", 100),
			wantLen: MaxLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLabelValue(tt.value)

			if len(got) != tt.wantLen {
				t.Errorf("SanitizeLabelValue() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestParsePeers(t *testing.T) {
	tests := []struct {
		name      string
		peersData string
		wantCount int
		wantFirst types.PeerInfo
	}{
		{
			name:      "empty string",
			peersData: "",
			wantCount: 0,
		},
		{
			name:      "single peer",
			peersData: "pod-0:10.0.0.1:0",
			wantCount: 1,
			wantFirst: types.PeerInfo{PodName: "pod-0", PodIP: "10.0.0.1"},
		},
		{
			name:      "multiple peers",
			peersData: "pod-0:10.0.0.1:0\npod-1:10.0.0.2:1\npod-2:10.0.0.3:2",
			wantCount: 3,
			wantFirst: types.PeerInfo{PodName: "pod-0", PodIP: "10.0.0.1"},
		},
		{
			name:      "handles whitespace",
			peersData: "  pod-0:10.0.0.1:0  \n\n  pod-1:10.0.0.2:1  ",
			wantCount: 2,
			wantFirst: types.PeerInfo{PodName: "pod-0", PodIP: "10.0.0.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePeers(tt.peersData)

			if len(got) != tt.wantCount {
				t.Errorf("ParsePeers() count = %d, want %d", len(got), tt.wantCount)
			}

			if tt.wantCount > 0 && (got[0].PodName != tt.wantFirst.PodName || got[0].PodIP != tt.wantFirst.PodIP) {
				t.Errorf("ParsePeers()[0] = %+v, want %+v", got[0], tt.wantFirst)
			}
		})
	}
}

func TestGetRank(t *testing.T) {
	peers := []types.PeerInfo{
		{PodName: "worker-2", PodIP: "10.0.0.3"},
		{PodName: "worker-0", PodIP: "10.0.0.1"},
		{PodName: "worker-1", PodIP: "10.0.0.2"},
	}

	tests := []struct {
		podName  string
		wantRank int
	}{
		{"worker-0", 0}, // alphabetically first
		{"worker-1", 1},
		{"worker-2", 2},
		{"worker-9", -1}, // not found
	}

	for _, tt := range tests {
		t.Run(tt.podName, func(t *testing.T) {
			if got := GetRank(tt.podName, peers); got != tt.wantRank {
				t.Errorf("GetRank(%q) = %d, want %d", tt.podName, got, tt.wantRank)
			}
		})
	}
}
