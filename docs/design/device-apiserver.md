# Technical Specification: device-apiserver

## Overview
The Device API server validates and configures data for the hardware api objects which include GPUs. The API Server services gRPC operations and provides the frontend to a node’s shared hardware resource state through which all other local hardware-aware components interact.

---

## Architecture 
The server implements the Kubernetes API provider pattern but is optimized for node-local footprint.

### Storage Stack
To avoid the overhead of a full Etcd cluster, the data layer utilizes:
- **Kine**: An Etcd shim that translates Etcd v3 API calls into SQL queries.
- **SQLite**: The database engine.
  - **Default**: In-memory for ephemeral runtime state.
  - **Optional**: Persistent file-based storage for state that must survive restarts.

### State Semantics
- **ResourceVersion (RV)**: Managed by Kine/SQLite. Increments on every write to provide Optimistic Concurrency Control.
- **Generation**: Managed by the server. Increments only when `.Spec` is modified, signaling desired state changes.

---

## gRPC Interface Definition
The server exposes the standard CRUD+UpdateStatus+Patch+Watch interface for hardware resources.

| Method | Target | Scope |
| :--- | :--- | :--- |
| **CreateGpu** | Full Object | Spec/Metadata |
| **UpdateGpu** | **Spec Only** | Spec/Metadata |
| **PatchGpu** | Partial Object | Spec/Metadata |
| **UpdateGpuStatus** | **Status Only** | Status |
| **GetGpu** | Read-only | Single Resource |
| **ListGpus** | Read-only | All Resources |
| **WatchGpus** | Stream Events | All Resources |

---

## Resource Schema
All API objects follow the Kubernetes Resource Model (KRM) other than the following exceptions:

### Metadata
- `ObjectMeta`: A subset of `k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta`.
  - `Name`
  - `ResourceVersion`
  - `Namespace`
  - `UID`
  - `Generation`
  - `CreationTimestamp`

- `ListMeta`: A subset of `k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta`.
  - `ResourceVersion`

### Options
- `CreateOptions`: Not supported.

- `UpdateOptions`: Not supported.

- `DeleteOptions`: Not supported.

- `ListOptions`: A subset of `k8s.io/apimachinery/pkg/apis/meta/v1.ListOptions`.
  - `ResourceVersion`

---

## Validation & Concurrency
- **Immutability**: The server rejects updates attempting to change `metadata.name`, `metadata.uid` or `metadata.namespace`.
- **Optimistic Concurrency**: If `incoming.ResourceVersion` does not match `current.ResourceVersion`, the server returns a `storage.Conflict` error, forcing the client to re-read and try again.
- **No-Op**: Updates where the `incoming.Spec` matches `current.Spec` result in a successful return without a database write.

---

## Internal Mechanics

### Bootstrapping & Persistence
The server's lifecycle is tied to the Kine-managed SQLite instance:
- **In-Memory**: The SQLite database exists purely in memory. The `device-apiserver` starts with a blank slate. Another component (e.g., the device plugin) is responsible for re-discovering and re-registering the GPUs on every start.
- **On-Disk**: The SQLite database exists in a single ordinary disk file (e.g., `/var/lib/device-apiserver/state.db`). The `device-apiserver` starts from the last successfully persisted state.

### API Discovery & Registration
The `device-apiserver` uses a decentralized registration pattern to manage its API surface. During startup, available APIs are automatically discovered and registered with both the storage backend and gRPC server.

---

## Reliability
- **Database Integrity**: SQLite's WAL (Write-Ahead Logging) mode is enabled by default to allow multiple concurrent readers and a single writer.

---

## Observability
The server's observability stack is designed for production-grade monitoring.

### Prometheus Metrics
- **Build Metadata (`device_apiserver_build_info`)**: A constant `Gauge` containing labels for `version`, `revision`, `build_date`, and `goversion`, `compiler`, and `platform`.
- **Service Availability (`device_apiserver_service_status`)**: A `GuageVec` that tracks the serving state of internal sub-services (`1`: Serving / Ready, `2`: Not Serving / Storage Backend Disconnected).
- **gRPC Performance (`grpc_server_*`)**: Standard `Histogram`s and `Counter`s via the `grpcprom` provider.
- **Storage Backend (`kine_*`)**:
  - **`kine_sql_total`**: A `CounterVec` tracking the total number of SQL operations, labeled by `error_code`.
  - **`kine_sql_time_seconds`**: A `HistogramVec` providing the distribution of SQL execution times.
  - **`kine_compact_total`**: A `CounterVec` recording successful and failed history compactions.
  - **`kine_insert_errors_total`**: A `CounterVec` tracking retries due to unique constraint violations

### Admin & Reflection
- **gRPC Reflection**: Dynamic discovery of API schema.
- **Health Checks**: Standard `grpc.health.v1` for liveness and readiness probes.
- **Channelz**: Low-level socket and connection-level statistics.

---

## Security
// TODO

---

## Performance
// TODO

---
