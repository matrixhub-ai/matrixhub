---
sidebar_position: 1
---

# 快速开始

欢迎使用 MatrixHub！本指南将帮助您快速启动并运行。

## 使用 Docker 快速部署

使用 Docker Compose 在几分钟内本地部署 MatrixHub：

```bash
curl -fsSL https://bit.ly/4qqSZIG | docker compose -f - up -d
```

## 配置 HF Endpoint

将您的 Hugging Face 工具指向 MatrixHub 实例：

```bash
export HF_ENDPOINT=https://your-matrixhub-instance
```

之后，您可以继续使用现有的工作流 — `huggingface_hub`、`transformers` 或 `vllm` — 无需修改任何代码。

## 下一步

- 了解 MatrixHub 的[核心概念](/docs/concepts)
- 浏览 vLLM 和 SGLang 的[集成指南](/docs/integrations)
- 为您的团队设置[访问控制](/docs/operations/project-management/members)
