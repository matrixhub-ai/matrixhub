---
sidebar_position: 1
---

# Reference

This section provides configuration schemas, environment variables, Helm values, and OpenAPI generation guides for MatrixHub developers and platform engineers.

---

## 🔌 1. Environment Variables

Configure MatrixHub container execution or client redirection using the variables below:

| Environment Variable | Allowed Values | Default Value | Description |
| :--- | :--- | :--- | :--- |
| `MATRIXHUB_DATABASE_DSN` | Custom standard MySQL DSN connection string. | `""` | Connection string for external MySQL (e.g. `user:pass@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&parseTime=true`). |
| `HF_ENDPOINT` | External HTTP registry URL address. | `https://huggingface.co` | Redirects standard AI inference engines and HF clients to cache weights inside MatrixHub. |

---

## ⚙️ 2. Core Configurations (`config/config.yaml`)

Specify server behavior, local caching engines, and database connections. A typical development configuration is outlined below:

```yaml
# MatrixHub System Configuration Schema
debug: false                    # Set true to print verbose database SQL logs
logLevel: "warn"                # Logging granularity: debug / info / warn / error

apiServer:
  port: 9527                    # Listening port for backend API requests
  database:
    driver: "mysql"             # Driver type: mysql / postgres
    migrate: true               # Auto trigger SQL database schema updates on boot
    migrationPath: "/etc/matrixhub/migrations"
    maxOpenConns: 100
    maxIdleConns: 10
    connMaxLifetimeSeconds: 3600
```

---

## ☸️ 3. Helm Values Reference Chart

Customize deployment parameters when installing the official MatrixHub Kubernetes chart:

| Value Key | Parameter Description | Default Value |
| :--- | :--- | :--- |
| `apiserver.replicaCount` | Number of running apiserver pod replicas. | `1` |
| `apiserver.image.registry` | Target registry to pull the apiserver container. | `ghcr.io` |
| `apiserver.image.repository` | Repository name inside the registry. | `matrixhub-ai/matrixhub` |
| `apiserver.service.type` | Kubernetes service type: `ClusterIP`, `NodePort`, or `LoadBalancer`. | `ClusterIP` |
| `apiserver.service.nodePort` | Static external port assigned to each node (type `NodePort`). | `30001` |
| `apiserver.resources.limits` | Hard resource limitations allowed for the API server pod. | `{cpu: 500m, memory: 512Mi}` |
| `global.storage.apiserver.builtIn` | Uses the built-in chart-managed MySQL instance. | `true` |
| `mysql.persistence.size` | PVC disk capacity requested for the built-in database state. | `8Gi` |

---

## 🛠️ 4. Generate OpenAPI Clients

If the Swagger schema parameters are changed during API server updates, you can easily regenerate the corresponding testing client SDK:

```bash
# Clean and compile standard swagger schema definitions
make gen_openapi_sdk
```
The SDK files will be compiled and output to the `./test/client/` directory.
