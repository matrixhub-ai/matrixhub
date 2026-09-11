---
sidebar_position: 1
---

# Troubleshooting

This section covers common MatrixHub deployment issues and troubleshooting steps.

---

## 1. Database Issues

The default Docker Compose deployment uses SQLite. If it reports `database is
locked`, confirm that only one MatrixHub instance uses the database and that
`./data/matrixhub` is on a local filesystem.

For the optional MySQL deployment, check the MySQL container status and logs:

```bash
docker compose -f docker-compose.mysql.yml ps mysql
docker compose -f docker-compose.mysql.yml logs mysql
```

If the MySQL container has stopped, restart it:

```bash
docker compose -f docker-compose.mysql.yml restart mysql
```

The MySQL variant connects through the `mysql` service name. Its credentials are configured through environment variables in `docker-compose.mysql.yml`.

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
