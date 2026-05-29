---
title: 安装指南
sidebar_position: 1
---

# 安装指南

本指南将协助您在生产环境中部署 MatrixHub 服务。我们官方提供并维护两种满足生产级高可用要求的部署方式：**Docker Compose 部署**（适用于单机虚拟主机或裸金属服务器）和 **Helm Chart 部署**（适用于 Kubernetes 云原生集群）。

---

## 📋 系统先决条件

在开始安装前，请确认您的服务器或部署环境满足以下底座要求：
*   **Docker Compose 部署**：已安装 Docker 20.10+ 及 Docker Compose v2.0+。
*   **Helm (Kubernetes) 部署**：可访问且配置就绪的 Kubernetes 集群 v1.19+，以及安装完毕的 Helm 客户端 CLI v3.0+。
*   **数据库**：可正常访问的 MySQL 实例（v5.7 或 v8.0）。*注：Docker Compose 和 Helm 的默认一键安装参数中均已内置自动拉取并配置内置 MySQL 的逻辑，您无需手动准备。*

---

## 🐋 方式 A：Docker Compose 单机部署

Docker Compose 是在单台物理机或虚拟机上部署并调试 MatrixHub 最为敏捷和推荐的方式。

### 1. 下载配置文件
获取官方预配置的部署模板文件：
*   `docker-compose.yaml` （定义 API 容器服务和 MySQL 容器配置）
*   `config.yaml` （包含数据库 DSN 字符串、日志过滤级别及本地磁盘路径绑定映射）

### 2. 启动注册表服务
确保这两个配置文件放置在同一个目录中，然后执行启动指令：
```bash
# 在后台守护进程（daemon）模式下启动所有容器服务
docker compose -f docker-compose.yaml up -d
```

### 3. 验证运行状态
检查容器的运行健康度：
```bash
docker compose ps
```
容器启动完毕后，您可直接通过本地的 `3001` 端口访问 API 服务（`http://localhost:3001`）。

---

## ☸️ 方式 B：Helm (Kubernetes) 集群部署

对于需要多节点动态水平伸缩、高可用保证、GPU 算力集群直接共享本地模型缓存的生产级场景，推荐使用 Helm 在 Kubernetes 集群中部署。

请先设置部署环境变量，方便拷贝命令：
```bash
# 设置您的 Helm Chart 目标部署版本以及命名空间
export CHART_VERSION=0.1.0  
export NAMESPACE=matrixhub
```

### 选项 1：直接通过 OCI 镜像注册表安装（推荐）
我们的 Helm Chart 作为标准的 OCI 制品发布于 GitHub 容器注册表（`ghcr.io`）中，可以直接无缝拉取：
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace
```

### 选项 2：通过本地 Chart 源码安装
如果您已经将 MatrixHub 源码克隆到了本地：
```bash
helm install matrixhub ./deploy/charts/matrixhub \
  --namespace ${NAMESPACE} --create-namespace
```

---

## 🌐 暴露服务入口

### 1. 使用 ClusterIP（默认）
默认情况下，Chart 中的服务暴露类型为 ClusterIP，仅在 Kubernetes 集群内部网络中可访问（默认端口为 `9527`）。若需在本地调试，可通过 port-forward 建立端口转发：
```bash
export POD_NAME=$(kubectl get pods --namespace matrixhub -l app=matrixhub-apiserver -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward $POD_NAME 9527:9527 --namespace matrixhub
```

### 2. 使用 NodePort 暴露
若需在集群外通过任何工作节点的 IP 永久访问 MatrixHub 服务：
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort \
  --set apiserver.service.nodePort=30001
```

---

## 📦 存储持久化配置

MatrixHub 采用标准的 **PersistentVolumeClaims (PVC)** 资源来长久保存缓存的公共模型权重、私有微调参数以及 MySQL 数据库状态。

默认安装下，Helm 默认声明并申请以下 PVC 存储：

| PVC 声明名称 | 容器挂载路径 | 默认容量大小 | 存储用途 |
| :--- | :--- | :--- | :--- |
| `<release>-apiserver-data` | `/data/matrixhub` | `50Gi` | 保存大模型物理文件、模型元数据及代理缓存 |
| `<release>-mysql-pv-claim` | `/var/lib/mysql` | `8Gi` | 保存数据库表数据（仅当 `mysql.builtIn=true` 时生效） |

### 动态自定义存储类与存储容量大小
为了适配您云厂商的云盘存储类（StorageClass，如 AWS gp3、阿里云 alicloud-disk ），在安装时可以覆盖默认参数：
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.storage.pvc.storageClassName=gp3 \
  --set apiserver.storage.pvc.size=500Gi \
  --set mysql.persistence.storageClass=gp3 \
  --set mysql.persistence.size=50Gi
```

### 绑定已有的预配置 PVC 卷
如果您之前已经申请好并分配了专用的高吞吐 PVC 存储卷：
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.storage.pvc.existingClaim=my-pre-provisioned-pvc
```
