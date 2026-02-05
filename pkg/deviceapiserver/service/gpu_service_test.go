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

package service

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/klog/v2"

	v1alpha1 "github.com/nvidia/nvsentinel/internal/generated/device/v1alpha1"
	"github.com/nvidia/nvsentinel/pkg/deviceapiserver/cache"
)

func newTestService(t *testing.T) *GpuService {
	t.Helper()
	logger := klog.Background()
	c := cache.New(logger, nil)
	return NewGpuService(c)
}

func TestCreateGpu_Idempotent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	req := &v1alpha1.CreateGpuRequest{
		Gpu: &v1alpha1.Gpu{
			Metadata: &v1alpha1.ObjectMeta{Name: "gpu-0"},
			Spec:     &v1alpha1.GpuSpec{Uuid: "GPU-1234"},
			Status:   &v1alpha1.GpuStatus{},
		},
	}

	// First create should succeed
	gpu1, err := svc.CreateGpu(ctx, req)
	if err != nil {
		t.Fatalf("First CreateGpu failed: %v", err)
	}
	if gpu1 == nil {
		t.Fatal("First CreateGpu returned nil GPU")
	}

	// Second create (idempotent) should succeed and return non-nil GPU
	gpu2, err := svc.CreateGpu(ctx, req)
	if err != nil {
		t.Fatalf("Second CreateGpu failed: %v", err)
	}
	if gpu2 == nil {
		t.Fatal("Second CreateGpu returned nil GPU — error from Get() was swallowed")
	}
	if gpu2.GetMetadata().GetName() != "gpu-0" {
		t.Errorf("Expected name=gpu-0, got %s", gpu2.GetMetadata().GetName())
	}
}

func TestCreateGpu_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	tests := []struct {
		name string
		req  *v1alpha1.CreateGpuRequest
	}{
		{
			name: "nil gpu",
			req:  &v1alpha1.CreateGpuRequest{},
		},
		{
			name: "nil metadata",
			req: &v1alpha1.CreateGpuRequest{
				Gpu: &v1alpha1.Gpu{
					Spec: &v1alpha1.GpuSpec{Uuid: "GPU-1234"},
				},
			},
		},
		{
			name: "empty name",
			req: &v1alpha1.CreateGpuRequest{
				Gpu: &v1alpha1.Gpu{
					Metadata: &v1alpha1.ObjectMeta{Name: ""},
					Spec:     &v1alpha1.GpuSpec{Uuid: "GPU-1234"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateGpu(ctx, tt.req)
			if err == nil {
				t.Error("Expected error for invalid request")
			}
		})
	}
}

func TestGetGpu(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Create a GPU first
	createReq := &v1alpha1.CreateGpuRequest{
		Gpu: &v1alpha1.Gpu{
			Metadata: &v1alpha1.ObjectMeta{Name: "gpu-get"},
			Spec:     &v1alpha1.GpuSpec{Uuid: "GPU-GET-1234"},
			Status:   &v1alpha1.GpuStatus{},
		},
	}
	_, err := svc.CreateGpu(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateGpu failed: %v", err)
	}

	// Get existing GPU
	resp, err := svc.GetGpu(ctx, &v1alpha1.GetGpuRequest{Name: "gpu-get"})
	if err != nil {
		t.Fatalf("GetGpu failed: %v", err)
	}
	if resp.GetGpu().GetMetadata().GetName() != "gpu-get" {
		t.Errorf("Expected name=gpu-get, got %s", resp.GetGpu().GetMetadata().GetName())
	}

	// Get non-existent GPU
	_, err = svc.GetGpu(ctx, &v1alpha1.GetGpuRequest{Name: "gpu-nonexistent"})
	if err == nil {
		t.Error("Expected error for non-existent GPU")
	}

	// Get with empty name
	_, err = svc.GetGpu(ctx, &v1alpha1.GetGpuRequest{Name: ""})
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestListGpus(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// List empty cache
	resp, err := svc.ListGpus(ctx, &v1alpha1.ListGpusRequest{})
	if err != nil {
		t.Fatalf("ListGpus failed: %v", err)
	}
	if len(resp.GetGpuList().GetItems()) != 0 {
		t.Errorf("Expected 0 GPUs, got %d", len(resp.GetGpuList().GetItems()))
	}

	// Create GPUs
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("gpu-%d", i)
		_, err := svc.CreateGpu(ctx, &v1alpha1.CreateGpuRequest{
			Gpu: &v1alpha1.Gpu{
				Metadata: &v1alpha1.ObjectMeta{Name: name},
				Spec:     &v1alpha1.GpuSpec{Uuid: fmt.Sprintf("GPU-%d", i)},
				Status:   &v1alpha1.GpuStatus{},
			},
		})
		if err != nil {
			t.Fatalf("CreateGpu failed: %v", err)
		}
	}

	// List should return 3
	resp, err = svc.ListGpus(ctx, &v1alpha1.ListGpusRequest{})
	if err != nil {
		t.Fatalf("ListGpus failed: %v", err)
	}
	if len(resp.GetGpuList().GetItems()) != 3 {
		t.Errorf("Expected 3 GPUs, got %d", len(resp.GetGpuList().GetItems()))
	}
}

func TestDeleteGpu(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Create then delete
	_, err := svc.CreateGpu(ctx, &v1alpha1.CreateGpuRequest{
		Gpu: &v1alpha1.Gpu{
			Metadata: &v1alpha1.ObjectMeta{Name: "gpu-del"},
			Spec:     &v1alpha1.GpuSpec{Uuid: "GPU-DEL"},
			Status:   &v1alpha1.GpuStatus{},
		},
	})
	if err != nil {
		t.Fatalf("CreateGpu failed: %v", err)
	}

	_, err = svc.DeleteGpu(ctx, &v1alpha1.DeleteGpuRequest{Name: "gpu-del"})
	if err != nil {
		t.Fatalf("DeleteGpu failed: %v", err)
	}

	// Verify deleted
	_, err = svc.GetGpu(ctx, &v1alpha1.GetGpuRequest{Name: "gpu-del"})
	if err == nil {
		t.Error("Expected error for deleted GPU")
	}

	// Delete non-existent should fail
	_, err = svc.DeleteGpu(ctx, &v1alpha1.DeleteGpuRequest{Name: "gpu-nonexistent"})
	if err == nil {
		t.Error("Expected error for non-existent GPU")
	}
}
