---
title: 快速开始
sidebar_position: 1
---

# 快速开始

欢迎使用 MatrixHub！MatrixHub 能够协助企业顺利将 AI 工作负载从公网公共仓库平滑迁移至私有、合规且高可用的企业内网级基础设施。本指南将引导您在几分钟内完成部署并开始使用。

---

## 🚀 步骤 1：部署 MatrixHub 实例

您可以使用 **Docker Compose**，在一行命令内迅速拉取并启动本地私有模型注册表服务：

```bash
# 在后台下载并运行 MatrixHub 容器集群
curl -fsSL https://bit.ly/4qqSZIG | docker compose -f - up -d
```

默认情况下，服务将在本地的 `3001` 端口上提供：
*   **后端 API 端点**：`http://127.0.0.1:3001`
*   **Web 控制台**：通过浏览器访问 `http://localhost:3001` 打开管理面板。

---

## 🛠️ 步骤 2：配置客户端重定向

要通过您的 MatrixHub 缓存代理实例来拉取并分发模型，您仅需要更改客户端侧的环境变量。整个过程中，**无需修改您项目代码中的任何一行逻辑**。

请在您的开发或 GPU 推理节点终端设置全局环境变量：

```bash
# 将 Hugging Face 的请求网关重定向为您的 MatrixHub 实例地址
export HF_ENDPOINT=http://127.0.0.1:3001
```

---

## 💻 步骤 3：验证客户端对接

在环境变量 `HF_ENDPOINT` 重定向完毕后，任何基于官方 Hugging Face Hub 库构建的算法工具，在下载模型时都会由 MatrixHub 自动拦截、下载、持久化缓存，并本地分发。

### 方式 A：使用 Hugging Face CLI 命令行
直接在命令行终端拉取并缓存模型：
```bash
# 通过本地 MatrixHub 缓存服务器下载公网开源模型（例如 Qwen2.5-0.5B）
huggingface-cli download Qwen/Qwen2.5-0.5B
```

### 方式 B：使用 Python Transformers 库
```python
import os
# 确保在运行前注入了重定向环境变量
os.environ["HF_ENDPOINT"] = "http://127.0.0.1:3001"

from transformers import AutoModelForCausalLM, AutoTokenizer

# MatrixHub 将自动拦截此请求：
# 1. 首次调用时：从公网源静默拉取并持久化缓存在本地磁盘/存储上。
# 2. 后续节点调用时：完全从本地高速局域网（10Gbps+）加载，冷启动瞬间完成。
model_id = "Qwen/Qwen2.5-0.5B-Instruct"
tokenizer = AutoTokenizer.from_pretrained(model_id)
model = AutoModelForCausalLM.from_pretrained(model_id)
```

### 方式 C：启动 vLLM 大模型推理服务
配置 `HF_ENDPOINT` 启动 vLLM 接口服务，享受本地毫秒级权重的拉取体验：
```bash
# 设置环境变量，一键启动 vLLM 服务
HF_ENDPOINT=http://127.0.0.1:3001 \
python3 -m vllm.entrypoints.openai.api_server \
  --model Qwen/Qwen2.5-7B-Instruct \
  --port 8000
```

---

## 🔍 步骤 4：确认缓存和治理状态

想要确认模型是否已经成功缓存：
1.  在浏览器中打开 MatrixHub 的 Web 控制台：`http://localhost:3001`。
2.  进入 **Repositories (模型仓库)** 选项卡。
3.  您会看到刚才拉取的模型（例如 `Qwen/Qwen2.5-0.5B`）已自动注册，且状态显示为“已同步”，现在，您的整个内网 GPU 算力集群都已可以直接秒级访问它了！
