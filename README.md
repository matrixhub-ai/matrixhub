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

* **Flexible Storage**: Supports PVC-backed local file system and NFS storage today, with S3-compatible object storage planned for a future release.
* **Reliable Replication**: Policy-driven, chunked transfers ensure data consistency even over unstable global networks.
* **Cloud-Native Design**: Optimized for Kubernetes with official **Helm charts** and horizontal scaling capabilities.

## 🌐 Live Demo

Try MatrixHub instantly at **[demo.matrixhub.ai](https://demo.matrixhub.ai/)** — no setup required.

Sign in with the public demo credentials:

| Username | Password |
| --- | --- |
| `admin` | `changeme` |

> The demo is for evaluation only and may be reset at any time.

## 🚀 Quick Start

### Docker Compose Deployment

Download the Docker Compose files for a released version and start MatrixHub:

```bash
export MATRIXHUB_VERSION=v0.1.1

mkdir -p matrixhub
cd matrixhub

curl -fL https://raw.githubusercontent.com/matrixhub-ai/matrixhub/${MATRIXHUB_VERSION}/deploy/docker-compose.yml \
  -o docker-compose.yml
curl -fL https://raw.githubusercontent.com/matrixhub-ai/matrixhub/${MATRIXHUB_VERSION}/deploy/config.yaml \
  -o config.yaml

MATRIXHUB_IMAGE_TAG="${MATRIXHUB_VERSION}" docker compose up -d
```

For a newer stable release, replace `v0.1.1` with the version you want to run.
If port `3001` is already in use, set `MATRIXHUB_HTTP_PORT` before starting the stack, for example `MATRIXHUB_HTTP_PORT=3002`.

Open the MatrixHub web console:

```text
http://127.0.0.1:3001
```

Sign in with the default local credentials:

| Username | Password |
| --- | --- |
| `admin` | `changeme` |

Change the default password before exposing the instance outside your local machine.

To stop the local stack:

```bash
docker compose down
```

### Helm (Kubernetes) Deployment

#### Prerequisites

Currently, the Helm chart supports PVC-backed storage for MatrixHub data. S3-compatible object storage is planned for a future release.

Make sure your cluster has a default StorageClass (`kubectl get storageclass`), or explicit storage settings for the PVCs this chart creates. For development clusters without a StorageClass, see [development-only local storage setup](deploy/charts/matrixhub/README.md#development-only-local-storage-setup).

#### Installing the Chart

MatrixHub publishes its Helm chart to GitHub Container Registry (`ghcr.io`) as an OCI artifact.

For a newer stable release, replace `0.1.1` with the chart version you want to run:

```bash
export CHART_VERSION=0.1.1
export NAMESPACE=matrixhub
```

Install the chart and expose the service via `NodePort`:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort
```

The default installation uses the cluster's default StorageClass. The MatrixHub data PVC defaults to `50Gi`, and the built-in MySQL PVC defaults to `8Gi`. To change PVC sizes, add `--set apiserver.storage.pvc.size=100Gi` or `--set mysql.persistence.size=20Gi` to the command.

For other storage classes, existing PVCs, and other Helm settings, see the [Helm chart README](deploy/charts/matrixhub/README.md).

#### Access the UI

With the `NodePort` installation above, open:

```text
http://<node-ip>:30001
```

Find a node IP with:

```bash
kubectl get nodes -o wide
```

#### Uninstall

```bash
helm uninstall matrixhub --namespace ${NAMESPACE}
```

This removes resources including the default PVCs created by the chart. To preserve data, use an existing PVC for MatrixHub data and an external database.

## Contributing and Development

We welcome contributions. 

See [CONTRIBUTING.md](CONTRIBUTING.md) for issues, pull requests, reviews, DCO, and release note requirements.

See [Development Guide](docs/development.md) for local setup, testing, and generated code. 

## Community and Support

Join us in [GitHub Discussions](https://github.com/matrixhub-ai/matrixhub/discussions)
or the [CNCF Slack `#matrixhub`](https://cloud-native.slack.com/archives/C0A8UKWR8HG)
for questions, ideas, and support.

- [Documentation site](https://matrixhub.ai)
- [Roadmap](docs/roadmap.md)

## Security

Please report vulnerabilities according to [SECURITY.md](SECURITY.md).

## License

MatrixHub is licensed under the [Apache License 2.0](LICENSE).
