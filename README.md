# NVSentinel

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

We welcome contributions! Here's how to get started:

**Ways to Contribute**:
- 🐛 Report bugs and request features via [issues](https://github.com/NVIDIA/NVSentinel/issues)
- 🧭 See what we're working on in the [roadmap](ROADMAP.md)
- 📝 Improve documentation
- 🧪 Add tests and increase coverage
- 🔧 Submit pull requests to fix issues
- 💬 Help others in [discussions](https://github.com/NVIDIA/NVSentinel/discussions)

**Getting Started**:
1. Read the [Contributing Guide](CONTRIBUTING.md) for guidelines
2. Check the [Development Guide](DEVELOPMENT.md) for setup instructions
3. Browse [open issues](https://github.com/NVIDIA/NVSentinel/issues) for opportunities

All contributors must sign their commits (DCO). See the contributing guide for details.

## 💬 Support

- 🐛 **Bug Reports**: [Create an issue](https://github.com/NVIDIA/NVSentinel/issues/new)
- ❓ **Questions**: [Start a discussion](https://github.com/NVIDIA/NVSentinel/discussions/new?category=q-a)
- 🔒 **Security**: See [Security Policy](SECURITY.md)

### Stay Connected

- ⭐ **Star this repository** to show your support
- 👀 **Watch** for updates on releases and announcements
- 🔗 **Share** NVSentinel with others who might benefit

--- 

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

*Built with ❤️ by NVIDIA for GPU infrastructure reliability*
