---
sidebar_position: 1
---

# Installation

This guide walks you through deploying MatrixHub in production. We support two official, production-ready installation methods: **Docker Compose** (for single-node hosts) and **Helm Charts** (for Kubernetes clusters).

---

## 📋 System Prerequisites

Before starting, ensure your target host/environment satisfies these requirements:
*   **Docker Compose Deployment**: Docker 20.10+ and Docker Compose v2.0+ installed.
*   **Helm (Kubernetes) Deployment**: Kubernetes cluster v1.19+ and Helm CLI v3.0+ configured.
*   **Database**: An accessible MySQL database instance (v5.7 or v8.0). *Note: Docker Compose and Helm default installations automatically spin up a built-in MySQL database.*

---

## 🐋 Method A: Docker Compose Deployment

Docker Compose is the easiest way to deploy MatrixHub on a standalone virtual machine or server.

### 1. Fetch Configuration Files
Download the official deployment compose configurations:
*   `docker-compose.yaml` (Defines the API server and MySQL container configuration)
*   `config.yaml` (Configures database DSN, log levels, and local storage bindings)

### 2. Start the Registry Service
Ensure both files are in the same directory and execute:
```bash
# Start all containers in background daemon mode
docker compose -f docker-compose.yaml up -d
```

### 3. Verification
Check the running state:
```bash
docker compose ps
```
The API server will listen on local host port `3001` (`http://localhost:3001`).

---

## ☸️ Method B: Helm (Kubernetes) Deployment

For enterprise-grade high availability, dynamic scaling, and direct cluster caching, install MatrixHub on Kubernetes using Helm.

Set the deployment environment variables first:
```bash
# Define your Helm chart target versions and target namespace
export CHART_VERSION=0.1.0  
export NAMESPACE=matrixhub
```

### Option 1: Install from OCI Registry (Recommended)
Our Helm charts are securely published to GitHub Container Registry (`ghcr.io`) as OCI artifacts:
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace
```

### Option 2: Install from Local Chart Source
If you have cloned the source repository:
```bash
helm install matrixhub ./deploy/charts/matrixhub \
  --namespace ${NAMESPACE} --create-namespace
```

---

## 🌐 Exposing the Service

### 1. ClusterIP (Default)
By default, the service is only exposed within the internal Kubernetes network on port `9527`. To access it externally for diagnostics, establish a port-forward:
```bash
export POD_NAME=$(kubectl get pods --namespace matrixhub -l app=matrixhub-apiserver -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward $POD_NAME 9527:9527 --namespace matrixhub
```

### 2. NodePort
To expose the registry permanently across all Kubernetes worker node IPs:
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort \
  --set apiserver.service.nodePort=30001
```

---

## 📦 Persistent Storage Configuration

MatrixHub uses standard **PersistentVolumeClaims (PVC)** to store cache data, fine-tuned weights, and MySQL database state.

By default, the Helm chart requests the following PVC resources:

| PVC Name | Mount Container Path | Default Size | Purpose |
| :--- | :--- | :--- | :--- |
| `<release>-apiserver-data` | `/data/matrixhub` | `50Gi` | Model files, artifacts, proxy cache |
| `<release>-mysql-pv-claim` | `/var/lib/mysql` | `8Gi` | Database states (when `mysql.builtIn=true`) |

### Customize Storage Class and Capacities
To customize PVC claims according to your cloud provider's storage classes, pass overrides to your helm install:
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.storage.pvc.storageClassName=gp3 \
  --set apiserver.storage.pvc.size=500Gi \
  --set mysql.persistence.storageClass=gp3 \
  --set mysql.persistence.size=50Gi
```

### Use an Existing Pre-Provisioned PVC
```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.storage.pvc.existingClaim=my-pre-provisioned-pvc
```
