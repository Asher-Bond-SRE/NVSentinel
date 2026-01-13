# NVIDIA Device API

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg?logo=go&logoColor=white)](https://golang.org/)

**Protocol Buffer definitions for NVIDIA device management APIs**

This repository contains the protocol buffer definitions and generated Go code for 
the NVIDIA Device API (`device.nvidia.com`). These APIs provide a standardized interface
for observing and managing GPU resources in Kubernetes environments.

## 📦 Contents

```
api/
├── proto/                    # Protocol Buffer definitions
│   └── device/v1alpha1/
│       └── gpu.proto         # GPU resource API definitions
├── gen/go/                   # Generated Go code
│   └── device/v1alpha1/
│       ├── gpu.pb.go         # Generated protobuf messages
│       └── gpu_grpc.pb.go    # Generated gRPC service stubs
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
└── Makefile                  # Build automation
```

## 🚀 Quick Start

### Using the Go Package

```bash
go get github.com/nvidia/nvsentinel/api@latest
```

### Import in Your Code

```go
import (
    v1alpha1 "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
)
```

### Example Usage

```go
package main

import (
    "context"
    "log"
    
    v1alpha1 "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // Connect to a GPU service
    conn, err := grpc.NewClient("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("failed to connect: %v", err)
    }
    defer conn.Close()

    client := v1alpha1.NewGpuServiceClient(conn)

    // List all GPUs
    resp, err := client.ListGpus(context.Background(), &v1alpha1.ListGpusRequest{})
    if err != nil {
        log.Fatalf("failed to list GPUs: %v", err)
    }

    for _, gpu := range resp.GpuList.Items {
        log.Printf("GPU: %s (UUID: %s)", gpu.Name, gpu.Spec.Uuid)
    }
}
```

## 🔧 API Overview

### GpuService

The `GpuService` provides an API for observing GPU resources:

| Method | Description |
|--------|-------------|
| `GetGpu` | Retrieves a single GPU resource by its unique name |
| `ListGpus` | Retrieves a list of all GPU resources |
| `WatchGpus` | Streams lifecycle events (ADDED, MODIFIED, DELETED) for GPU resources |

### GPU Resource Model

The GPU resource follows the Kubernetes Resource Model pattern (Spec/Status):

```protobuf
message Gpu {
  string name = 1;      // Unique logical identifier
  GpuSpec spec = 2;     // Identity and desired attributes
  GpuStatus status = 3; // Most recently observed state
}
```

## 🛠️ Development

### Prerequisites

- Go 1.25+
- Protocol Buffers compiler (`protoc`)
- [yq](https://github.com/mikefarah/yq) - YAML processor

### Regenerate Protocol Buffers

```bash
cd api
make protos-generate
```

### Build and Verify

```bash
cd api
make build
```

### Clean Generated Files

```bash
cd api
make protos-clean
```

## 🤝 Contributing

We welcome contributions! Please see:

- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Development Guide](DEVELOPMENT.md)

All contributors must sign their commits (DCO).

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

*Built by NVIDIA for GPU infrastructure management*
