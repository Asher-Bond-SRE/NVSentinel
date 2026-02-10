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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodGroupDiscoverer_CanHandle(t *testing.T) {
	tests := []struct {
		name   string
		config PodGroupConfig
		pod    *corev1.Pod
		want   bool
	}{
		{
			name:   "KAI - matches annotation",
			config: KAIConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"pod-group-name": "my-pg"},
				},
			},
			want: true,
		},
		{
			name:   "KAI - no annotation",
			config: KAIConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"some-label": "value"},
				},
			},
			want: false,
		},
		{
			name:   "Volcano - matches annotation",
			config: VolcanoConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"volcano.sh/pod-group": "my-pg"},
				},
			},
			want: true,
		},
		{
			name:   "Volcano - matches job label",
			config: VolcanoConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"volcano.sh/job-name": "my-job"},
				},
			},
			want: true,
		},
		{
			name:   "Volcano - matches group-name annotation",
			config: VolcanoConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"scheduling.k8s.io/group-name": "my-pg"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPodGroupDiscoverer(nil, nil, tt.config)

			if got := d.CanHandle(tt.pod); got != tt.want {
				t.Errorf("CanHandle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodGroupDiscoverer_ExtractGangID(t *testing.T) {
	tests := []struct {
		name   string
		config PodGroupConfig
		pod    *corev1.Pod
		want   string
	}{
		{
			name:   "KAI gang ID format",
			config: KAIConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   "ml-team",
					Annotations: map[string]string{"pod-group-name": "training-job"},
				},
			},
			want: "kai-ml-team-training-job",
		},
		{
			name:   "Volcano gang ID format",
			config: VolcanoConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   "default",
					Annotations: map[string]string{"volcano.sh/pod-group": "pg-123"},
				},
			},
			want: "volcano-default-pg-123",
		},
		{
			name:   "no matching annotation returns empty",
			config: KAIConfig(),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPodGroupDiscoverer(nil, nil, tt.config)

			if got := d.ExtractGangID(tt.pod); got != tt.want {
				t.Errorf("ExtractGangID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPresets(t *testing.T) {
	// Verify all presets are properly configured
	expected := []string{"kai", "volcano"}

	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			fn, ok := Presets[name]
			if !ok {
				t.Fatalf("Preset %q not found", name)
			}

			cfg := fn()
			if cfg.Name != name {
				t.Errorf("Preset name = %q, want %q", cfg.Name, name)
			}

			if cfg.PodGroupGVR.Resource == "" {
				t.Error("Preset PodGroupGVR.Resource is empty")
			}
		})
	}
}
