---
sidebar_position: 1
---

# Getting Started

Welcome to MatrixHub! This guide will help you get up and running quickly.

## Quick Start with Docker

Deploy MatrixHub locally in minutes using Docker Compose:

```bash
curl -fsSL https://bit.ly/4qqSZIG | docker compose -f - up -d
```

## Configure HF Endpoint

Point your Hugging Face tools to your MatrixHub instance:

```bash
export HF_ENDPOINT=https://your-matrixhub-instance
```

Then use your existing workflows — `huggingface_hub`, `transformers`, or `vllm` — without any code changes.

## Next Steps

- Learn about [core concepts](/docs/concepts) behind MatrixHub
- Explore [integration guides](/docs/integrations) for vLLM and SGLang
- Set up [access control](/docs/operations/project-management/members) for your team
