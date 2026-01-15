# NVIDIA Device API

The NVIDIA Device API provides a Kubernetes-idiomatic Go SDK and Protobuf definitions for interacting with NVIDIA device resources.

## Repository Structure

| Module | Description |
| :--- | :--- |
| [`api/`](./api) | Protobuf definitions and Go types for the Device API. |
| [`client-go/`](./client-go) | Kubernetes-style generated clients, informers, and listers. |
| [`code-generator/`](./code-generator) | Tools for generating NVIDIA-specific client logic. |

---

## Getting Started

### Prerequisites

To build and contribute to this project, you need:

* **Go**: `v1.25+`
* **Protoc**: Required for protobuf generation.
* **golangci-lint**: Required for code quality checks.
* **Make**: Used for orchestrating build and generation tasks.

### Installation

Clone the repository and build the project:

```bash
git clone https://github.com/nvidia/nvsentinel.git
cd nvsentinel
make build
```

---

## Usage

The `client-go` module includes several examples for how to use the generated clients:

* **Standard Client**: Basic CRUD operations.
* **Shared Informers**: High-performance caching for controllers.
* **Watch**: Real-time event streaming via gRPC.

See the [examples](./client-go/examples) directory for details.

---

## Contributing

We welcome contributions! Please see:

- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Development Guide](DEVELOPMENT.md)

All contributors must sign their commits (DCO).

--- 

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

*Built by NVIDIA for GPU infrastructure management*
