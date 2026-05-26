---
title: 快速开始
sidebar_position: 1
---

# 快速开始

欢迎使用 MatrixHub！本指南将帮助您快速上手。

## 使用 Docker 快速启动

使用 Docker Compose 在本地几分钟内部署 MatrixHub：

```bash
curl -fsSL https://bit.ly/4qqSZIG | docker compose -f - up -d
```

## 配置 HF 端点

将您的 Hugging Face 工具指向您的 MatrixHub 实例：

```bash
export HF_ENDPOINT=https://your-matrixhub-instance
```

然后使用您现有的工作流 —— `huggingface_hub`、`transformers` 或 `vllm` —— 无需更改任何代码。

## 后续步骤

- 了解 MatrixHub 背后的[核心概念](/docs/concepts)
- 探索 vLLM 和 SGLang 的[集成指南](/docs/integrations)
- 为您的团队设置[访问控制](/docs/operations/project-management/members)
