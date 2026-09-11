---
sidebar_position: 1
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Installation

We support two official installation methods: **Docker Compose** (for single-node hosts) and **Helm Charts** (for Kubernetes clusters).

---

## 🐋 Docker Compose Deployment

Docker Compose is the easiest way to deploy MatrixHub on a standalone virtual machine or server. The default deployment uses SQLite and does not require a separate database service.

Download the Docker Compose files for a released version and start MatrixHub:

<Tabs groupId="operating-system">
<TabItem value="linux-macos" label="Linux / macOS" default>

```bash
export MATRIXHUB_VERSION=v0.2.0-rc.5

mkdir -p matrixhub && cd matrixhub

curl -fL \
  "https://raw.githubusercontent.com/matrixhub-ai/matrixhub/$MATRIXHUB_VERSION/deploy/docker-compose.yml" \
  -o docker-compose.yml
curl -fL \
  "https://raw.githubusercontent.com/matrixhub-ai/matrixhub/$MATRIXHUB_VERSION/deploy/config.yaml" \
  -o config.yaml

MATRIXHUB_IMAGE_TAG="$MATRIXHUB_VERSION" docker compose up -d
```

</TabItem>
<TabItem value="windows" label="Windows (PowerShell)">

```powershell
$env:MATRIXHUB_VERSION = "v0.2.0-rc.5"

New-Item -ItemType Directory -Force -Path "matrixhub" | Out-Null
Set-Location "matrixhub"

Invoke-WebRequest `
  -Uri "https://raw.githubusercontent.com/matrixhub-ai/matrixhub/$env:MATRIXHUB_VERSION/deploy/docker-compose.yml" `
  -OutFile "docker-compose.yml"
Invoke-WebRequest `
  -Uri "https://raw.githubusercontent.com/matrixhub-ai/matrixhub/$env:MATRIXHUB_VERSION/deploy/config.yaml" `
  -OutFile "config.yaml"

$env:MATRIXHUB_IMAGE_TAG = $env:MATRIXHUB_VERSION
docker compose up -d
```

</TabItem>
</Tabs>

Replace `v0.2.0-rc.5` with the version you want to run.
If port `3001` is already in use, set `MATRIXHUB_HTTP_PORT` before starting the stack, for example `export MATRIXHUB_HTTP_PORT=3002` on Linux/macOS or `$env:MATRIXHUB_HTTP_PORT = "3002"` in PowerShell.

Open the MatrixHub web console:

```text
http://127.0.0.1:3001
```

SQLite data is stored in `./data/matrixhub`. Run only one MatrixHub instance against this database and keep it on a local filesystem.

To use MySQL instead, download [`docker-compose.mysql.yml`](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/docker-compose.mysql.yml) and [`config-mysql.yaml`](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/config-mysql.yaml), then run:

```bash
MATRIXHUB_IMAGE_TAG="$MATRIXHUB_VERSION" docker compose -f docker-compose.mysql.yml up -d
```

The public demo and the default Helm deployment continue to use MySQL.
---

## ☸️ Helm (Kubernetes) Deployment

### Prerequisites

Currently, the Helm chart supports PVC-backed storage for MatrixHub data. S3-compatible object storage is planned for a future release.

Make sure your cluster has a default StorageClass (`kubectl get storageclass`), or explicit storage settings for the PVCs this chart creates. For development clusters without a StorageClass, see [development-only local storage setup](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/charts/matrixhub/README.md#development-only-local-storage-setup).

### Installing the Chart

MatrixHub publishes its Helm chart to GitHub Container Registry (`ghcr.io`) as an OCI artifact.

For a newer stable release, replace `0.1.1` with the chart version you want to run:

```bash
export CHART_VERSION=0.1.1
export NAMESPACE=matrixhub
```

Install the chart and expose the service via `NodePort`:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version "$CHART_VERSION" \
  --namespace "$NAMESPACE" --create-namespace \
  --set apiserver.service.type=NodePort
```

The default installation uses the cluster's default StorageClass. The MatrixHub data PVC defaults to `50Gi`, and the built-in MySQL PVC defaults to `8Gi`. To change PVC sizes, add `--set apiserver.storage.pvc.size=100Gi` or `--set mysql.persistence.size=20Gi` to the command.

For other storage classes, existing PVCs, and other Helm settings, see the [Helm chart README](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/charts/matrixhub/README.md).

### Access the UI

With the `NodePort` installation above, open:

```text
http://<node-ip>:30001
```

Find a node IP with:

```bash
kubectl get nodes -o wide
```

### Uninstall

```bash
helm uninstall matrixhub --namespace "$NAMESPACE"
```

This removes resources including the default PVCs created by the chart. To preserve data, use an existing PVC for MatrixHub data and an external database.

## Get started

After installation, start with
[Cache models through a proxy project](../guides/mirror-from-huggingface.md).
