---
title: 故障排除
sidebar_position: 1
---

# 故障排除

本章节介绍 MatrixHub 部署中的常见问题及排查方法。

---

## 1. MySQL 连接失败

如果 MatrixHub 无法启动，或日志中出现数据库连接错误，请检查 MySQL 容器的状态和日志：

```bash
docker compose ps mysql
docker compose logs mysql
```

如果 MySQL 容器已经停止，可以重新启动：

```bash
docker compose restart mysql
```

Docker Compose 默认使用服务名 `mysql` 连接数据库，账号和密码由 `docker-compose.yml` 中的环境变量配置。

---

## 2. 端口冲突

MatrixHub 默认映射到宿主机端口 `3001`。如果启动时提示端口已被占用，通过 `MATRIXHUB_HTTP_PORT` 修改宿主机端口：

```bash
MATRIXHUB_HTTP_PORT=3002 docker compose up -d
```

修改后通过 `http://127.0.0.1:3002` 访问 MatrixHub。

---

## 3. 服务健康检查

如果部署完成后无法访问 MatrixHub，可以先检查 API Server 是否正在响应：

```bash
curl -i http://localhost:3001/healthz
```

正常响应为 `HTTP 200`，响应正文为 `OK`。该接口只表示 API Server 正在运行，不检查数据库等依赖组件的状态。
