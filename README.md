# MatrixHub

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

### One-Line Deployment (Recommended)

The easiest way to deploy MatrixHub is using our CLI tool with npx:

```bash
npx @matrixhub/cli start
```

This will automatically:
- Check Docker installation
- Pull the latest MatrixHub image
- Start the container with optimal settings
- Show you the access URL

**Additional CLI commands:**

```bash
npx @matrixhub/cli status    # Check container status
npx @matrixhub/cli logs -f   # Follow logs in real-time
npx @matrixhub/cli stop      # Stop the container
npx @matrixhub/cli restart   # Restart the container
npx @matrixhub/cli update    # Update to latest version
```

**Custom configuration:**

```bash
npx @matrixhub/cli start -p 8080 -d ~/my-data
```

See the [CLI documentation](packages/cli/README.md) for more options.

### Docker Deployment

Alternatively, deploy MatrixHub directly with Docker:

```bash
docker run -d -p 9527:9527 -v $PWD/data:/data ghcr.io/matrixhub-ai/matrixhub:main
```

Access MatrixHub at `http://localhost:9527`.

## Community, discussion, contribution, and support

Slack is our primary channel for community discussion, contribution coordination, and support. You can reach the maintainers and community at:

- [Slack](https://cloud-native.slack.com/archives/C0A8UKWR8HG)
