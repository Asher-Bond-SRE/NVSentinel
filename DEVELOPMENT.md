# NVIDIA Device API Development Guide

This guide covers development setup and workflows for contributing to the NVIDIA Device API.

## 📋 Table of Contents

- [Getting Started](#-getting-started)
- [Project Structure](#-project-structure)
- [Development Workflow](#-development-workflow)
- [Protocol Buffer Development](#-protocol-buffer-development)
- [Code Standards](#-code-standards)

## 🚀 Getting Started

### Prerequisites

**Required Tools:**

- [Go 1.25+](https://golang.org/dl/) - See `.versions.yaml` for exact version
- [Protocol Buffers Compiler](https://grpc.io/docs/protoc-installation/) (`protoc`)
- [yq](https://github.com/mikefarah/yq) - YAML processor for version management

**Install protoc (macOS):**

```bash
brew install protobuf
```

**Install protoc (Linux):**

```bash
# Download from https://github.com/protocolbuffers/protobuf/releases
PROTOC_VERSION=33.0
curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local
```

**Install yq:**

```bash
# macOS
brew install yq

# Linux
wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/local/bin/yq
chmod +x /usr/local/bin/yq
```

### Quick Setup

```bash
git clone https://github.com/nvidia/device-api.git
cd device-api/api
make protos-generate
make build
```

### Tool Version Management

Tool versions are managed in `.versions.yaml`. View current versions:

```bash
cat .versions.yaml
```

## 🏗️ Project Structure

```
device-api/
├── api/
│   ├── proto/                    # Protocol Buffer source definitions
│   │   └── device/v1alpha1/
│   │       └── gpu.proto         # GPU resource API (device.nvidia.com/v1alpha1)
│   ├── gen/go/                   # Generated Go code (do not edit)
│   │   └── device/v1alpha1/
│   │       ├── gpu.pb.go         # Generated protobuf messages
│   │       └── gpu_grpc.pb.go    # Generated gRPC stubs
│   ├── bin/                      # Local tool binaries
│   ├── go.mod
│   ├── go.sum
│   └── Makefile
├── .github/                      # GitHub Actions and templates
├── .versions.yaml                # Tool version definitions
├── .golangci.yml                 # Go linting configuration
├── README.md
├── DEVELOPMENT.md                # This file
├── LICENSE
├── SECURITY.md
└── CODE_OF_CONDUCT.md
```

## 🔄 Development Workflow

### Daily Development

1. **Make Changes to Proto Files**

   ```bash
   # Edit proto definitions
   vim api/proto/device/v1alpha1/gpu.proto
   ```

2. **Regenerate Code**

   ```bash
   cd api
   make protos-generate
   ```

3. **Verify Build**

   ```bash
   make build
   ```

4. **Commit Changes**

   ```bash
   git add .
   git commit -s -m "feat: add new GPU field"
   git push origin your-branch
   ```

### Available Make Targets

```bash
cd api
make help           # Show all available targets
make all            # Generate protos and build
make protos-generate # Generate Go code from .proto files
make protos-clean   # Remove generated code
make build          # Build the Go module
```

## 🔧 Protocol Buffer Development

### Adding a New Message

1. **Edit the proto file:**

   ```protobuf
   // api/proto/device/v1alpha1/gpu.proto
   
   message NewMessage {
     string field1 = 1;
     int32 field2 = 2;
   }
   ```

2. **Regenerate:**

   ```bash
   make protos-generate
   ```

3. **Verify the generated code compiles:**

   ```bash
   make build
   ```

### Adding a New Service Method

1. **Add the RPC to the service:**

   ```protobuf
   service GpuService {
     // Existing methods...
     
     // New method
     rpc NewMethod(NewMethodRequest) returns (NewMethodResponse);
   }
   
   message NewMethodRequest {
     // Request fields
   }
   
   message NewMethodResponse {
     // Response fields
   }
   ```

2. **Regenerate and verify:**

   ```bash
   make protos-generate
   make build
   ```

### Proto Style Guidelines

- Use `snake_case` for field names
- Use `CamelCase` for message and service names
- Include documentation comments for all messages and fields
- Follow [Google's Protocol Buffer Style Guide](https://protobuf.dev/programming-guides/style/)

## 📏 Code Standards

### Commit Messages

Follow conventional commits:

```
feat: add new GPU condition type
fix: correct timestamp handling in conditions
docs: update API documentation
```

### Signed Commits

All commits must be signed off (DCO):

```bash
git commit -s -m "Your commit message"
```

### License Headers

All source files must include the Apache 2.0 license header.

### Proto File Header

```protobuf
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

syntax = "proto3";
// ... rest of proto file
```

## 🐛 Troubleshooting

### protoc not found

```bash
# Verify protoc is installed
which protoc
protoc --version

# If not found, install it (see Prerequisites)
```

### yq not found

```bash
# Install yq
brew install yq  # macOS
# or see https://github.com/mikefarah/yq for Linux
```

### Generated code not compiling

```bash
# Clean and regenerate
make protos-clean
make protos-generate
make build
```

### Wrong protoc-gen-go version

```bash
# Remove local binaries and let make reinstall
rm -rf api/bin/
make protos-generate
```

## 📞 Getting Help

- **Issues**: [Create an issue](https://github.com/NVIDIA/device-api/issues/new)
- **Questions**: [Start a discussion](https://github.com/NVIDIA/device-api/discussions)
- **Security**: See [SECURITY.md](SECURITY.md)

---

Happy coding! 🚀
