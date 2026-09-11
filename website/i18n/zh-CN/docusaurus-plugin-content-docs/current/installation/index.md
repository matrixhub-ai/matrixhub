---
title: 安装指南
sidebar_position: 1
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# 安装指南

我们支持两种官方安装方式：**Docker Compose**（适用于单机）和 **Helm Chart**（适用于 Kubernetes 集群）。

---

## 🐋 Docker Compose 部署

Docker Compose 是在独立虚拟机或服务器上部署 MatrixHub 最简单的方式。默认部署使用 SQLite，无需单独运行数据库服务。

下载已发布版本的 Docker Compose 文件并启动 MatrixHub：

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

如需使用其他版本，请将 `v0.2.0-rc.5` 替换为您要运行的版本。
如果端口 `3001` 已被占用，请在启动服务前设置 `MATRIXHUB_HTTP_PORT`，例如在 Linux/macOS 中运行 `export MATRIXHUB_HTTP_PORT=3002`，或在 PowerShell 中运行 `$env:MATRIXHUB_HTTP_PORT = "3002"`。

打开 MatrixHub Web 控制台：

```text
http://127.0.0.1:3001
```

SQLite 数据保存在 `./data/matrixhub`。请仅运行一个使用该数据库的 MatrixHub 实例，并将数据库保存在本地文件系统上。

如需使用 MySQL，请下载 [`docker-compose.mysql.yml`](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/docker-compose.mysql.yml) 和 [`config-mysql.yaml`](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/config-mysql.yaml)，然后运行：

```bash
MATRIXHUB_IMAGE_TAG="$MATRIXHUB_VERSION" docker compose -f docker-compose.mysql.yml up -d
```

公共 Demo 和默认 Helm 部署仍使用 MySQL。

---

## ☸️ Helm (Kubernetes) 部署

### 前提条件

目前，Helm Chart 支持使用 PVC 存储 MatrixHub 数据。未来版本计划支持兼容 S3 的对象存储。

请确保集群具有默认 StorageClass（运行 `kubectl get storageclass` 查看），或者为 Chart 创建的 PVC 显式配置存储。对于没有 StorageClass 的开发集群，请参阅[仅用于开发环境的本地存储配置](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/charts/matrixhub/README.md#development-only-local-storage-setup)。

### 安装 Chart

MatrixHub 将 Helm Chart 作为 OCI 制品发布到 GitHub Container Registry（`ghcr.io`）。

如需使用更新的稳定版本，请将 `0.1.1` 替换为您要运行的 Chart 版本：

```bash
export CHART_VERSION=0.1.1
export NAMESPACE=matrixhub
```

安装 Chart 并通过 `NodePort` 暴露服务：

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version "$CHART_VERSION" \
  --namespace "$NAMESPACE" --create-namespace \
  --set apiserver.service.type=NodePort
```

默认安装使用集群的默认 StorageClass。MatrixHub 数据 PVC 的默认容量为 `50Gi`，内置 MySQL PVC 的默认容量为 `8Gi`。如需更改 PVC 容量，请在命令中添加 `--set apiserver.storage.pvc.size=100Gi` 或 `--set mysql.persistence.size=20Gi`。

其他 StorageClass、已有 PVC 以及其他 Helm 配置，请参阅 [Helm Chart README](https://github.com/matrixhub-ai/matrixhub/blob/main/deploy/charts/matrixhub/README.md)。

### 访问 UI

使用上述 `NodePort` 方式安装后，打开：

```text
http://<node-ip>:30001
```

使用以下命令查找节点 IP：

```bash
kubectl get nodes -o wide
```

### 卸载

```bash
helm uninstall matrixhub --namespace "$NAMESPACE"
```

此命令会删除 Chart 创建的资源，包括默认 PVC。如需保留数据，请为 MatrixHub 数据使用已有 PVC，并使用外部数据库。

## 开始使用

安装完成后，可以从 [通过代理项目缓存模型](../guides/mirror-from-huggingface.md) 开始。
