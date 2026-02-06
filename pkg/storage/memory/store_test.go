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

package memory

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
)

// codec is the shared codec used by all tests. UnstructuredJSONScheme handles
// encoding and decoding of unstructured.Unstructured objects without needing
// a registered scheme or concrete Go types.
var codec runtime.Codec = unstructured.UnstructuredJSONScheme

// newTestObject builds an *unstructured.Unstructured with the given name and
// namespace, suitable for storage in the test store.
func newTestObject(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "GPU",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}

func TestStore_CreateAndGet(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	obj := newTestObject("gpu-0", "default")
	out := &unstructured.Unstructured{}

	if err := s.Create(ctx, "/gpus/default/gpu-0", obj, out, 0); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify resourceVersion was set on the output object.
	rv := out.GetResourceVersion()
	if rv == "" {
		t.Fatal("expected resourceVersion to be set on out, got empty string")
	}

	if rv != "1" {
		t.Fatalf("expected resourceVersion '1', got %q", rv)
	}

	// Get the object back.
	got := &unstructured.Unstructured{}
	if err := s.Get(ctx, "/gpus/default/gpu-0", storage.GetOptions{}, got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.GetName() != "gpu-0" {
		t.Fatalf("expected name 'gpu-0', got %q", got.GetName())
	}

	if got.GetResourceVersion() != "1" {
		t.Fatalf("expected resourceVersion '1', got %q", got.GetResourceVersion())
	}
}

func TestStore_CreateDuplicate(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	obj := newTestObject("gpu-0", "default")

	if err := s.Create(ctx, "/gpus/default/gpu-0", obj, nil, 0); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := s.Create(ctx, "/gpus/default/gpu-0", obj, nil, 0)
	if err == nil {
		t.Fatal("expected error on duplicate Create, got nil")
	}

	if !storage.IsExist(err) {
		t.Fatalf("expected IsExist error, got: %v", err)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	got := &unstructured.Unstructured{}
	err := s.Get(ctx, "/gpus/default/gpu-missing", storage.GetOptions{}, got)

	if err == nil {
		t.Fatal("expected error on Get for missing key, got nil")
	}

	if !storage.IsNotFound(err) {
		t.Fatalf("expected IsNotFound error, got: %v", err)
	}
}

func TestStore_GetList(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	// Create 3 objects under the same prefix.
	for _, name := range []string{"gpu-0", "gpu-1", "gpu-2"} {
		obj := newTestObject(name, "default")
		if err := s.Create(ctx, "/gpus/default/"+name, obj, nil, 0); err != nil {
			t.Fatalf("Create %s failed: %v", name, err)
		}
	}

	list := &unstructured.UnstructuredList{}
	opts := storage.ListOptions{
		Recursive: true,
		Predicate: storage.SelectionPredicate{},
	}

	if err := s.GetList(ctx, "/gpus/default", opts, list); err != nil {
		t.Fatalf("GetList failed: %v", err)
	}

	if len(list.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(list.Items))
	}

	// Verify the list has a resource version.
	if list.GetResourceVersion() == "" {
		t.Fatal("expected list resourceVersion to be set")
	}
}

func TestStore_GuaranteedUpdate(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	obj := newTestObject("gpu-0", "default")
	if err := s.Create(ctx, "/gpus/default/gpu-0", obj, nil, 0); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	dest := &unstructured.Unstructured{}
	err := s.GuaranteedUpdate(ctx, "/gpus/default/gpu-0", dest, false, nil,
		func(input runtime.Object, res storage.ResponseMeta) (runtime.Object, *uint64, error) {
			u := input.(*unstructured.Unstructured)
			labels := u.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}

			labels["test-key"] = "test-value"
			u.SetLabels(labels)

			return u, nil, nil
		}, nil)
	if err != nil {
		t.Fatalf("GuaranteedUpdate failed: %v", err)
	}

	// Verify the label was persisted.
	got := &unstructured.Unstructured{}
	if err := s.Get(ctx, "/gpus/default/gpu-0", storage.GetOptions{}, got); err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}

	labels := got.GetLabels()
	if labels["test-key"] != "test-value" {
		t.Fatalf("expected label 'test-key'='test-value', got labels: %v", labels)
	}

	// Verify resourceVersion was incremented.
	if got.GetResourceVersion() != "2" {
		t.Fatalf("expected resourceVersion '2' after update, got %q", got.GetResourceVersion())
	}
}

func TestStore_GuaranteedUpdate_NotFound(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	dest := &unstructured.Unstructured{}
	err := s.GuaranteedUpdate(ctx, "/gpus/default/gpu-missing", dest, false, nil,
		func(input runtime.Object, res storage.ResponseMeta) (runtime.Object, *uint64, error) {
			return input, nil, nil
		}, nil)

	if err == nil {
		t.Fatal("expected error on GuaranteedUpdate for missing key with ignoreNotFound=false")
	}

	if !storage.IsNotFound(err) {
		t.Fatalf("expected IsNotFound error, got: %v", err)
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	obj := newTestObject("gpu-0", "default")
	if err := s.Create(ctx, "/gpus/default/gpu-0", obj, nil, 0); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	out := &unstructured.Unstructured{}
	err := s.Delete(ctx, "/gpus/default/gpu-0", out, nil, nil, nil, storage.DeleteOptions{})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if out.GetName() != "gpu-0" {
		t.Fatalf("expected deleted object name 'gpu-0', got %q", out.GetName())
	}

	// Verify the object is gone.
	got := &unstructured.Unstructured{}
	err = s.Get(ctx, "/gpus/default/gpu-0", storage.GetOptions{}, got)

	if err == nil {
		t.Fatal("expected NotFound error after delete, got nil")
	}

	if !storage.IsNotFound(err) {
		t.Fatalf("expected IsNotFound error, got: %v", err)
	}
}

func TestStore_DeleteNotFound(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	out := &unstructured.Unstructured{}
	err := s.Delete(ctx, "/gpus/default/gpu-missing", out, nil, nil, nil, storage.DeleteOptions{})

	if err == nil {
		t.Fatal("expected error on Delete for missing key, got nil")
	}

	if !storage.IsNotFound(err) {
		t.Fatalf("expected IsNotFound error, got: %v", err)
	}
}

func TestStore_Watch(t *testing.T) {
	s := NewStore(codec)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := s.Watch(ctx, "/gpus/default/", storage.ListOptions{})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	defer w.Stop()

	// Create an object; the watcher should receive an ADDED event.
	obj := newTestObject("gpu-0", "default")
	if err := s.Create(ctx, "/gpus/default/gpu-0", obj, nil, 0); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	select {
	case ev := <-w.ResultChan():
		if ev.Type != watch.Added {
			t.Fatalf("expected ADDED event, got %v", ev.Type)
		}

		u, ok := ev.Object.(*unstructured.Unstructured)
		if !ok {
			t.Fatalf("expected *unstructured.Unstructured, got %T", ev.Object)
		}

		if u.GetName() != "gpu-0" {
			t.Fatalf("expected event object name 'gpu-0', got %q", u.GetName())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}

func TestStore_Watch_Delete(t *testing.T) {
	s := NewStore(codec)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the object first, before starting the watch.
	obj := newTestObject("gpu-0", "default")
	if err := s.Create(ctx, "/gpus/default/gpu-0", obj, nil, 0); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	w, err := s.Watch(ctx, "/gpus/default/", storage.ListOptions{})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	defer w.Stop()

	// Delete the object; the watcher should receive a DELETED event.
	out := &unstructured.Unstructured{}
	if err := s.Delete(ctx, "/gpus/default/gpu-0", out, nil, nil, nil, storage.DeleteOptions{}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	select {
	case ev := <-w.ResultChan():
		if ev.Type != watch.Deleted {
			t.Fatalf("expected DELETED event, got %v", ev.Type)
		}

		u, ok := ev.Object.(*unstructured.Unstructured)
		if !ok {
			t.Fatalf("expected *unstructured.Unstructured, got %T", ev.Object)
		}

		if u.GetName() != "gpu-0" {
			t.Fatalf("expected event object name 'gpu-0', got %q", u.GetName())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DELETED watch event")
	}
}

func TestStore_Stats(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	for _, name := range []string{"gpu-0", "gpu-1"} {
		obj := newTestObject(name, "default")
		if err := s.Create(ctx, "/gpus/default/"+name, obj, nil, 0); err != nil {
			t.Fatalf("Create %s failed: %v", name, err)
		}
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.ObjectCount != 2 {
		t.Fatalf("expected ObjectCount 2, got %d", stats.ObjectCount)
	}
}

func TestStore_ReadinessCheck(t *testing.T) {
	s := NewStore(codec)

	if err := s.ReadinessCheck(); err != nil {
		t.Fatalf("ReadinessCheck failed: %v", err)
	}
}

func TestStore_GetCurrentResourceVersion(t *testing.T) {
	s := NewStore(codec)
	ctx := context.Background()

	rv0, err := s.GetCurrentResourceVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentResourceVersion failed: %v", err)
	}

	if rv0 != 0 {
		t.Fatalf("expected initial resourceVersion 0, got %d", rv0)
	}

	// Create two objects; each should increment the revision.
	for _, name := range []string{"gpu-0", "gpu-1"} {
		obj := newTestObject(name, "default")
		if err := s.Create(ctx, "/gpus/default/"+name, obj, nil, 0); err != nil {
			t.Fatalf("Create %s failed: %v", name, err)
		}
	}

	rv2, err := s.GetCurrentResourceVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentResourceVersion failed: %v", err)
	}

	if rv2 != 2 {
		t.Fatalf("expected resourceVersion 2 after two creates, got %d", rv2)
	}
}

func TestStore_ImplementsInterface(t *testing.T) {
	// Compile-time check that *Store satisfies storage.Interface.
	var _ storage.Interface = (*Store)(nil)
}
