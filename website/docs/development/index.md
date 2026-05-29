---
sidebar_position: 1
---

# Development

This guide describes how to set up, build, and run the MatrixHub frontend and backend services locally for development and debugging.

---

## 🛠️ Prerequisites

Ensure your local development environment has the following software installed:
*   **Go**: Version `1.23+` (For compiling and running the backend API services)
*   **Node.js**: Version `18+` (For building the frontend dashboard)
*   **pnpm**: Version `8+` (Preferred frontend package manager)
*   **Docker**: (Required to run the local MySQL test database)

---

## 🚀 Quick Start: Manual Launch

You can spin up the required database and launch both services manually to modify code and inspect live behavior.

### 1. Start a Local Test MySQL Container
Launch an isolated MySQL instance inside Docker on port `3306`:
```bash
docker run -d \
  --name matrixhub-mysql \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=matrixhub \
  -e MYSQL_USER=matrixhub \
  -e MYSQL_PASSWORD=changeme \
  -p 3306:3306 \
  mysql:8.4
```

### 2. Configure Database Environment Variables
Point your shell environment to the test database:
```bash
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&multiStatements=true&parseTime=true"
```

### 3. Spin Up the Go Backend API Server
Launch the backend apiserver component in hot-run mode:
```bash
# Triggers auto-migration and listens on port 3001
go run ./cmd/matrixhub apiserver
```

### 4. Spin Up the Frontend Dev Server
In a new terminal window, initialize and run the Vite-managed UI portal:
```bash
cd ui
pnpm install   # Run only on first startup
pnpm dev
```
The frontend dev portal will listen on `http://localhost:5173`. Any calls made to `/api/*` will automatically be proxied to the Go backend server.

---

## 🛠️ Automated Commands (Makefile)

We provide Makefile commands to automate your local development orchestration:

| Make Command | Description | Prerequisites |
| :--- | :--- | :--- |
| `make local-run` | Launches both backend API and frontend dev servers. | MySQL container must be running. |
| `make local-run-api` | Compiles and starts only the backend Go API server. | MySQL container must be running. |
| `make local-run-web` | Starts only the frontend Vite development UI server. | None (runs independently). |

---

## 💡 Practical Development Tips

### Database Auto-Migrations
*   Set `database.migrate: true` in your dev configuration to trigger automatic SQL table schema upgrades on boot.
*   Enable `debug: true` to output verbose database SQL queries to stdout for quick query optimization.

### Frontend Hot Reloading
*   Vite handles automatic hot-module replacement (HMR). Any modifications made inside the `./ui/` folder will immediately update in your browser without requiring server restarts.
*   Run `pnpm typecheck` before pushing code changes to ensure strict TypeScript type safety compliance.
