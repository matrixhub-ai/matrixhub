# MatrixHub Helm Chart

MatrixHub is an open-source, self-hosted AI model registry engineered for large-scale enterprise inference. It serves as a drop-in private replacement for Hugging Face, purpose-built to accelerate vLLM and SGLang workloads.

## Introduction

This chart bootstraps a [MatrixHub](https://github.com/matrixhub-ai/matrixhub) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.8+ for OCI registry support
- A default StorageClass, or explicit storage settings for the PVCs this chart creates

## Installing the Chart

Set the install target:

```bash
export CHART_VERSION=0.1.1
export NAMESPACE=matrixhub
```

### Install from OCI Registry

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace
```

To expose MatrixHub through a NodePort during installation:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort
```

### Install from Local Chart

Use this when developing from a local checkout:

```bash
helm install matrixhub ./deploy/charts/matrixhub \
  --namespace ${NAMESPACE} --create-namespace
```

To expose MatrixHub through a NodePort during installation:

```bash
helm install matrixhub ./deploy/charts/matrixhub \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort
```

## Uninstalling the Chart

To uninstall/delete the `matrixhub` deployment:

```bash
helm uninstall matrixhub --namespace ${NAMESPACE}
```

This removes resources created by the release, including the default PVCs created by the chart. To preserve MatrixHub data across uninstall/reinstall, use `apiserver.storage.pvc.existingClaim`; to preserve database data, use an external database.

## Configuration

The following table lists the configurable parameters of the MatrixHub chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `apiserver.replicaCount` | Number of replicas | `1` |
| `apiserver.labels` | Deployment labels | `{app: matrixhub-apiserver}` |
| `apiserver.podAnnotations` | Pod annotations | `{}` |
| `apiserver.podLabels` | Pod labels | `{}` |
| `apiserver.image.registry` | Image registry | `ghcr.io` |
| `apiserver.image.repository` | Image repository | `matrixhub-ai/matrixhub` |
| `apiserver.image.tag` | Image tag | `""` (release charts set this to the app image tag) |
| `apiserver.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `apiserver.image.pullSecrets` | Image pull secrets | `[]` |
| `apiserver.service.type` | Kubernetes service type | `ClusterIP` |
| `apiserver.service.port` | Service port | `9527` |
| `apiserver.service.nodePort` | NodePort (when type=NodePort) | `30001` |
| `apiserver.service.annotations` | Service annotations | `{}` |
| `apiserver.resources.limits.cpu` | CPU limit | `500m` |
| `apiserver.resources.limits.memory` | Memory limit | `512Mi` |
| `apiserver.resources.requests.cpu` | CPU request | `50m` |
| `apiserver.resources.requests.memory` | Memory request | `128Mi` |
| `apiserver.debug` | Debug mode | `false` |
| `apiserver.logLevel` | Log level (debug/info/warn/error) | `warn` |
| `apiserver.port` | API server port | `9527` |
| `apiserver.database.driver` | Database driver (mysql/postgres) | `mysql` |
| `apiserver.database.accessType` | Database access type | `readwrite` |
| `apiserver.database.maxOpenConns` | Max open connections | `100` |
| `apiserver.database.maxIdleConns` | Max idle connections | `10` |
| `apiserver.database.connMaxLifetimeSeconds` | Connection max lifetime | `3600` |
| `apiserver.database.connMaxIdleSeconds` | Connection max idle time | `1800` |
| `apiserver.database.migrate` | Enable database migration | `true` |
| `apiserver.database.migrationPath` | Migration path | `/etc/matrixhub/migrations` |
| `apiserver.database.dsn` | Custom database DSN | `""` |
| `apiserver.storage.mode` | Storage mode for MatrixHub data (`pvc` or `none`) | `pvc` |
| `apiserver.storage.pvc.storageClass` | StorageClass for MatrixHub data PVC | `""` (use default StorageClass) |
| `apiserver.storage.pvc.size` | MatrixHub data PVC size | `50Gi` |
| `apiserver.storage.pvc.existingClaim` | Existing PVC for MatrixHub data | `""` |
| `mysql.registry` | MySQL image registry | `docker.io` |
| `mysql.repository` | MySQL image repository | `library/mysql` |
| `mysql.tag` | MySQL image tag | `8.4` |
| `mysql.pullPolicy` | MySQL pull policy | `IfNotPresent` |
| `mysql.persistence.size` | PVC size | `8Gi` |
| `mysql.persistence.storageClass` | Storage class | `""` (default) |
| `mysql.pullSecrets` | MySQL pull secrets | `[]` |
| `mysql.rootPassword` | MySQL root password | `password` |
| `global.imagePullSecrets` | Global image pull secrets | `[]` |
| `global.imageRegistry` | Global image registry override | `""` |
| `global.storage.apiserver.builtIn` | Use built-in MySQL | `true` |
| `global.busybox.image.repository` | Busybox image repository | `busybox` |
| `global.busybox.image.tag` | Busybox image tag | `latest` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.image.tag=v0.1.1
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  -f values.yaml
```

## Storage

MatrixHub uses PersistentVolumeClaims to persist data. Currently only PVC is supported as the storage backend; S3-compatible storage will be supported in a future release.

By default, the chart creates the following PVCs:

| PVC | Container Mount Path | Default Size | Purpose |
|-----|----------------------|--------------|---------|
| `<release>-apiserver-data` | `/data/matrixhub` | `50Gi` | Model artifacts and cache |
| `<release>-mysql-pv-claim` | `/var/lib/mysql` | `8Gi` | Built-in MySQL data |

### Default StorageClass

The default installation relies on the cluster's default StorageClass. Check it with:

```bash
kubectl get storageclass
```

If one StorageClass is marked as `(default)`, no extra storage flags are required.

### Specify a StorageClass

If your cluster has a StorageClass but it is not marked as default, pass it explicitly:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort \
  --set apiserver.storage.pvc.storageClass=<storage-class> \
  --set mysql.persistence.storageClass=<storage-class>
```

The MatrixHub data PVC defaults to `50Gi`. You can override PVC sizes during install:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort \
  --set apiserver.storage.pvc.storageClass=<storage-class> \
  --set apiserver.storage.pvc.size=100Gi \
  --set mysql.persistence.storageClass=<storage-class> \
  --set mysql.persistence.size=20Gi
```

### Use an Existing PVC

To reuse an existing PVC for MatrixHub data:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort \
  --set apiserver.storage.pvc.existingClaim=my-existing-pvc
```

This only replaces the MatrixHub data PVC. The built-in MySQL database still needs a default StorageClass or `mysql.persistence.storageClass`. To avoid the built-in MySQL PVC, use an external database.

### Use an External Database

By default, the chart deploys a built-in MySQL database with persistent storage. To use an external database, set:

```yaml
global:
  storage:
    apiserver:
      builtIn: false

apiserver:
  database:
    dsn: "your-custom-dsn-string"
```

### Development-only Local Storage Setup

For local development or throwaway test clusters that do not have a StorageClass, the simplest option is Rancher's local-path provisioner:

```bash
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.36/deploy/local-path-storage.yaml
kubectl -n local-path-storage rollout status deploy/local-path-provisioner
kubectl annotate storageclass local-path \
  storageclass.kubernetes.io/is-default-class=true \
  --overwrite
kubectl get storageclass
```

Use this for local or test environments only. For production, use a storage provider appropriate for your cluster, such as a cloud CSI driver, Longhorn, OpenEBS, Ceph, or NFS CSI.

## Exposing the Service

### ClusterIP (default)

The service is exposed as ClusterIP and accessible from within the cluster:

```bash
kubectl port-forward svc/matrixhub 3001:9527 --namespace ${NAMESPACE}
```

### NodePort

To expose the service via NodePort:

```bash
helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version ${CHART_VERSION} \
  --namespace ${NAMESPACE} --create-namespace \
  --set apiserver.service.type=NodePort
```
