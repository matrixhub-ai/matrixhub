---
title: 配置参考
sidebar_position: 1
---

# 配置参考

本章节为系统架构师、平台运维工程师提供 MatrixHub 核心环境变量表、配置文件参数定义、Helm Chart 部署参数对照表以及 Swagger SDK 编译参考。

---

## 🔌 1. 系统环境变量

配置 MatrixHub 核心容器运行时或客户端重定向的变量说明：

| 环境变量名称 | 可选参数值 | 默认值 | 作用描述说明 |
| :--- | :--- | :--- | :--- |
| `MATRIXHUB_DATABASE_DSN` | 符合标准 MySQL DSN 连接串规范的字符串。 | `""` | 用于连接外部的 MySQL 数据库实例（如：`user:pass@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&parseTime=true`）。 |
| `HF_ENDPOINT` | 外部可访问的 HTTP 端口地址。 | `https://huggingface.co` | 配置于推理引擎或开发终端上，用于将默认请求网关拦截重定向到 MatrixHub 代理网关上。 |

---

## ⚙️ 2. 主配置文件详解 (`config/config.yaml`)

用于定制化 API Server 运行行为、代理缓存模式、日志过滤及底层数据库连接池：

```yaml
# MatrixHub 配置文件标准 Schema 结构
debug: false                    # 设为 true 时，控制台将高精度打印所有 SQL 操作日志
logLevel: "warn"                # 全局日志过滤输出级别：debug / info / warn / error

apiServer:
  port: 9527                    # 后端 API 服务组件的监听端口
  database:
    driver: "mysql"             # 底层数据库驱动类型：mysql / postgres
    migrate: true               # 每次启动时，是否自动检查并执行 SQL 表结构更新升级
    migrationPath: "/etc/matrixhub/migrations" # SQL 迁移结构文件物理路径
    maxOpenConns: 100           # 数据库连接池最大打开连接数
    maxIdleConns: 10            # 数据库连接池最大空闲连接数
    connMaxLifetimeSeconds: 3600 # 数据库连接最大生命周期时间
```

---

## ☸️ 3. Helm Chart 部署参数对照表

在使用 Helm 在 Kubernetes 集群中部署 MatrixHub 时，您可以通过覆盖以下 values 参数定制集群配置：

| Values 配置参数键 | 参数描述说明 | 默认参数值 |
| :--- | :--- | :--- |
| `apiserver.replicaCount` | API 服务组件 Pod 的副本数量。 | `1` |
| `apiserver.image.registry` | 拉取 API 服务组件镜像的外部镜像托管源。 | `ghcr.io` |
| `apiserver.image.repository` | 镜像托管源中的仓库名称。 | `matrixhub-ai/matrixhub` |
| `apiserver.service.type` | 服务公开暴露类型：`ClusterIP`、`NodePort` 或 `LoadBalancer`。 | `ClusterIP` |
| `apiserver.service.nodePort` | 当暴露类型为 `NodePort` 时映射在宿主节点上的外部端口。 | `30001` |
| `apiserver.resources.limits` | 限制 API 服务组件 Pod 最大允许占用的宿主 CPU 和内存上限。 | `{cpu: 500m, memory: 512Mi}` |
| `global.storage.apiserver.builtIn` | 是否在安装 Chart 时拉取并运行内置的一键式 MySQL 数据库。 | `true` |
| `mysql.persistence.size` | 当使用内置数据库时，申请存储底座 PVC 磁盘卷的大小。 | `8Gi` |

---

## 🛠️ 4. Swagger API 客户端 SDK 生成

当在后端增加了新的 Swagger API 接口定义时，为了方便测试，可以通过以下 Make 指令快速生成对应的客户端测试 SDK：

```bash
# 执行 Swagger 解析并编译更新客户端 SDK 源码
make gen_openapi_sdk
```
新生成的 SDK 源码将被直接输出至本地仓库的 `./test/client/` 文件夹中。
