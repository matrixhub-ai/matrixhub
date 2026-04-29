# MatrixHub

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/matrixhub-ai/matrixhub)

**MatrixHub** is an open-source, self-hosted AI model registry engineered for large-scale enterprise inference. It serves as a drop-in private replacement for Hugging Face, purpose-built to accelerate **vLLM** and **SGLang** workloads.

## 💡 Why MatrixHub?

MatrixHub streamlines the transition from public model hubs to production-grade infrastructure:

* **Zero-Wait Distribution**: Eliminate bandwidth bottlenecks with a **"Pull-once, serve-all"** cache, enabling 10Gbps+ speeds across 100+ GPU nodes simultaneously.
* **Air-Gapped Delivery**: Securely ferry models into isolated networks while maintaining a native `HF_ENDPOINT` experience for researchers—no internet required.
* **Private AI model Registry**: Centralize fine-tuned weights with **Tag locking** and CI/CD integration to guarantee absolute consistency from development to production.
* **Global Multi-Region Sync**: Automate asynchronous, resumable replication between data centers for high availability and low-latency local access.

## 🛠️ Core Features

### 🚀 High-Performance Distribution

* **Transparent HF Proxy**: Switch to private hosting with zero code changes by simply redirecting your endpoint.
* **On-Demand Caching**: Automatically localizes public models upon the first request to slash redundant traffic.
* **Inference Native**: Native support for **P2P distribution**, OCI artifacts, and **NetLoader** for direct-to-GPU weight streaming.

### 🛡️ Enterprise Governance & Security

* **RBAC & Multi-Tenancy**: Project-based isolation with granular permissions and seamless LDAP/SSO integration.
* **Audit & Compliance**: Full traceability with comprehensive logs for every upload, download, and configuration change.
* **Integrity Protection**: Built-in malware scanning and content signing to ensure models remain untampered.

### 🌍 Scalable Infrastructure

* **Storage Agnostic**: Compatible with local file systems, NFS, and S3-compatible backends (MinIO, AWS, etc.).
* **Reliable Replication**: Policy-driven, chunked transfers ensure data consistency even over unstable global networks.
* **Cloud-Native Design**: Optimized for Kubernetes with official **Helm charts** and horizontal scaling capabilities.

## 🚀 Quick Start

### Docker Compose Deployment

Use Docker Compose with the provided configuration files:

- `website/static/deploy/docker/docker-compose.yaml`
- `website/static/deploy/docker/config.yaml`

Make sure `docker-compose.yaml` and `config.yaml` are in the same folder, then start the service:

```bash
docker compose -f docker-compose.yaml up -d
```

Default service endpoint:

```text
http://127.0.0.1:3001
```

### Helm (Kubernetes) Deployment

MatrixHub provides two Helm installation methods — from a local chart or from the OCI registry.

Set the install target first (used in all commands below):

```bash
export CHART_VERSION=<chart-version>  
export NAMESPACE=matrixhub
```

#### Option A: Install from Local Chart

```bash
helm install matrixhub ./deploy/charts/matrixhub \
  --namespace ${NAMESPACE} --create-namespace
```

#### Option B: Install from OCI Registry

Charts are published to GitHub Container Registry (`ghcr.io`) as OCI artifacts.

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace
```

#### Expose the Service

Expose it via `NodePort`:

```bash
helm install matrixhub ./deploy/charts/matrixhub \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort
# or with OCI:
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort
```

#### Persistent Storage (PVC)

MatrixHub uses PersistentVolumeClaims to persist data. Currently only PVC is supported as the storage backend; S3-compatible storage will be supported in a future release.

By default, the chart creates the following PVCs:

| PVC | Mount Path | Default Size | Purpose |
|-----|-----------|--------------|---------|
| `<release>-apiserver-data` | `/data/matrixhub` | `50Gi` | Model artifacts & cache |
| `<release>-mysql-pv-claim` | `/var/lib/mysql` | `8Gi` | Built-in MySQL data (only when `global.storage.apiserver.builtIn=true`, which is the default) |

**Customize storage class and size:**

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.storage.mode=pvc \
  --set apiserver.storage.pvc.size=50Gi \
  --set mysql.persistence.size=20Gi
```

**Use an existing PVC:**

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.storage.pvc.existingClaim=my-existing-pvc
```

## 📚 Docs

- [Documentation site](https://matrixhub.ai)
- [Development guide](docs/development.md)

## Community, discussion, contribution, and support

Slack is our primary channel for community discussion, contribution coordination, and support. You can reach the maintainers and community at:

- [Slack](https://cloud-native.slack.com/archives/C0A8UKWR8HG)
