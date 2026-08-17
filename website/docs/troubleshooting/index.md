---
sidebar_position: 1
---

# Troubleshooting

This section covers common MatrixHub deployment issues and troubleshooting steps.

---

## 1. MySQL Connection Failure

If MatrixHub fails to start or its logs report a database connection error, check the MySQL container status and logs:

```bash
docker compose ps mysql
docker compose logs mysql
```

If the MySQL container has stopped, restart it:

```bash
docker compose restart mysql
```

Docker Compose connects to the database through the `mysql` service name. Its credentials are configured through environment variables in `docker-compose.yml`.

---

## 2. Port Conflicts

MatrixHub maps to host port `3001` by default. If startup reports that the port is already in use, change the host port with `MATRIXHUB_HTTP_PORT`:

```bash
MATRIXHUB_HTTP_PORT=3002 docker compose up -d
```

Then open MatrixHub at `http://127.0.0.1:3002`.

---

## 3. Service Health Check

If MatrixHub is inaccessible after deployment, first check whether the API Server is responding:

```bash
curl -i http://localhost:3001/healthz
```

The expected response is `HTTP 200` with `OK` in the response body. This endpoint only indicates that the API Server is running; it does not check the database or other dependencies.
