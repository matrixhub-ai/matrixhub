---
sidebar_position: 1
---

# Troubleshooting

This section provides diagnostic handbooks, commands, and standard configurations to troubleshoot and resolve common issues encountered during MatrixHub deployments.

---

## 💾 1. MySQL Connection Failure

If the backend API server fails to initialize because it cannot reach or connect to the database, follow these diagnostic steps:

### Step A: Verify Container Status
If you are running the built-in database via Docker:
```bash
# Check if the MySQL database container is running
docker ps | grep matrixhub-mysql
```
If it is stopped, restart it:
```bash
docker restart matrixhub-mysql
```

### Step B: Inspect Database Logs
If the container keeps crashing or refusing connections, check the startup logs:
```bash
docker logs matrixhub-mysql
```

### Step C: Validate the Connection String (DSN)
Ensure your environment variable connection DSN string matches your target host and credentials:
```bash
# Example verification of database connection string
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&parseTime=true"
```

---

## 🔌 2. Local Port occupied / Port Conflicts

If services fail to start because local network ports are occupied by other running background processes:

### Backend Port Conflicts (Default: 9527 / 3001)
If the default API server port is occupied, you can change the target listening port inside your configuration file:
*   Open `config/config.yaml`.
*   Change the value of `apiServer.port` to another open port (e.g. `9627`).

### Frontend Port Conflicts (Default: 5173)
If the Vite dev server port is occupied, specify a custom port during local frontend execution:
```bash
cd ui
pnpm dev --port 3000
```

---

## 📦 3. Local Dependency Issues

If compilation fails due to corrupted local modules, lockfiles, or node modules:

### Resolving Go Backend Dependency Breakages
Force a clean tidy of Go module hashes and download dependencies:
```bash
go mod tidy
go mod download
```

### Resolving Frontend npm/pnpm Module Breakages
Wipe local build files and lockfiles, and run a fresh installation:
```bash
cd ui
# Delete local caches
rm -rf node_modules pnpm-lock.yaml
# Run fresh install
pnpm install
```

---

## 🔍 4. Service Health Checks

Ensure the backend API server is fully functional and loading database tables by querying the health API:

```bash
# Curl the standard API health check route
curl -i http://localhost:3001/health
```
A healthy server will return a `200 OK` status response confirming that connections and internal database components are fully alive.
