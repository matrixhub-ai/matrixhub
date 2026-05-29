---
title: 故障排除
sidebar_position: 1
---

# 故障排除

本章节为您总结了在部署、调试或运维 MatrixHub 过程中最常见的技术问题、排查命令及标准诊断步骤，协助您快速恢复服务。

---

## 💾 1. MySQL 数据库连接失败

如果后端 API Server 服务在启动时报错，提示无法访问或连接到底层数据库，请按照以下步骤排查：

### 步骤 A：检查 MySQL 容器状态
如果您使用的是 Docker 运行的内置数据库：
```bash
# 检查 MySQL 容器的健康运行状态
docker ps | grep matrixhub-mysql
```
如果容器被意外停止，请尝试重新启动：
```bash
docker restart matrixhub-mysql
```

### 步骤 B：检查数据库容器日志
如果容器频繁崩溃或拒绝外部连接，请检查容器启动日志以获取报错细节：
```bash
docker logs matrixhub-mysql
```

### 步骤 C：验证数据库连接字符串 (DSN)
确保您环境或配置文件中声明的 DSN 字符串中的主机 IP、端口和凭证完全正确：
```bash
# 检查您注入的环境变量是否正确
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&parseTime=true"
```

---

## 🔌 2. 本地端口被占用 / 端口冲突

如果您在启动服务时收到端口冲突报错，说明本地默认的网络端口已被其他后台程序抢占：

### 后端服务端口冲突 (默认端口：9527 / 3001)
如果默认端口被抢占，您可以直接在本地配置文件中进行修改：
*   打开 `config/config.yaml` 配置文件。
*   将 `apiServer.port` 参数修改为其他闲置的端口（例如 `9627`）。

### 前端开发端口冲突 (默认端口：5173)
如果 Vite 本地开发服务器启动时提示端口占用，您可以在启动前端时传入指定的端口：
```bash
cd ui
pnpm dev --port 3000
```

---

## 📦 3. 本地编译与依赖冲突

如果在拉取代码后编译报错，或者本地 Node 模块哈希冲突：

### 解决 Go 后端依赖冲突
清空本地缓存并强制重新拉取同步 Go Modules：
```bash
go mod tidy
go mod download
```

### 解决前端 Node Modules 或锁文件冲突
彻底删除本地缓存的文件夹和锁文件，执行干净的全新下载：
```bash
cd ui
# 清理缓存文件
rm -rf node_modules pnpm-lock.yaml
# 执行全新依赖下载安装
pnpm install
```

---

## 🔍 4. 服务健康状态诊断

如果想确认后端 API 服务是否成功编译并正常拉取了数据库，可以直接访问内置的健康检查接口：

```bash
# 在终端发起健康检查 API 请求
curl -i http://localhost:3001/health
```
正常健康的实例会返回 `200 OK` 状态响应，并以 JSON 形式返回内部各组件数据库底座的联通状态。
