---
title: 本地开发
sidebar_position: 1
---

# 本地开发

本指南旨在协助算法研究员与平台工程师在本地搭建、编译和运行 MatrixHub 的前端控制台与后端 API 服务，以便进行高效的二次开发与调试。

---

## 🛠️ 二次开发先决条件

请确保您的本地开发环境已经安装并配置好以下底座软件：
*   **Go 语言环境**：版本 `1.23+`（用于编译和运行 Go 后端 API Server 服务）
*   **Node.js 环境**：版本 `18+`（用于构建前端控制台看板）
*   **pnpm 包管理器**：版本 `8+`（推荐的前端依赖包管理利器）
*   **Docker 运行时**：（用于一键运行本地 MySQL 测试数据库）

---

## 🚀 快速上手：手动分步启动

您可以手动在本地开启测试数据库，并分别运行前端和后端服务，这对于观察代码改动及接口调试最为直观。

### 1. 在 Docker 中运行测试 MySQL 数据库
在本地端口 `3306` 上一键拉取并启动独立的测试数据库：
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

### 2. 配置本地数据库环境变量
在您的命令行 shell 中注入 DSN 连接凭证：
```bash
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&multiStatements=true&parseTime=true"
```

### 3. 运行 Go 后端 API 接口服务
启动后端主程序，API 接口服务会自动读取环境变量执行 SQL 表结构初始化迁移：
```bash
# 启动 apiserver 服务，默认将监听本地 3001 端口
go run ./cmd/matrixhub apiserver
```

### 4. 运行前端看板开发服务器
新开一个终端窗口，进入前端目录安装依赖并开启开发服务器：
```bash
cd ui
pnpm install   # 仅在首次启动时需要执行
pnpm dev
```
前端本地开发服务器将在 `http://localhost:5173` 启动。Vite 会自动将 `/api/*` 开头的网络接口调用反向代理到您正在运行的 Go 后端服务上（`http://127.0.0.1:3001`）。

---

## 🛠️ 自动化便捷命令 (Makefile)

为简化开发调试中的微服务编排，我们在项目根目录下封装了快捷 Makefile 工具：

| Makefile 快捷指令 | 动作描述说明 | 依赖的前置条件 |
| :--- | :--- | :--- |
| `make local-run` | 一键同时开启后端 API Server 与前端 Vite 开发服务器。 | 本地 MySQL 测试容器必须已处于运行状态。 |
| `make local-run-api` | 仅编译并启动后端 Go API 服务。 | 本地 MySQL 测试容器必须已处于运行状态。 |
| `make local-run-web` | 仅启动前端 Vite 本地看板开发服务器。 | 无（无需依赖任何 Go 或 MySQL 状态即可单独运行调试前端 UI）。 |

---

## 💡 二次开发实用调试技巧

### 数据库自动迁移与 SQL 调试
*   在您的本地配置文件中设置 `database.migrate: true`，API 服务每次启动时都会扫描并执行增量的 SQL 升级变更。
*   将 `debug: true` 打开，您的服务控制台终端会实时打印所有底层的 SQL 执行过程，方便定位慢查询或索引问题。

### 前端热重载与类型强校验
*   Vite 具备极速的热模块替换（HMR）能力，任何在 `./ui/` 目录下对页面的修改都会在浏览器上秒级更新，无需任何重启。
*   在提交或向仓库发起 PR 前，请在前端目录执行 `pnpm typecheck` 以确保整个代码的 TypeScript 静态类型完全安全无误。
