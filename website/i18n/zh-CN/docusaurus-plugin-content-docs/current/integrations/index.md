---
title: 系统集成
sidebar_position: 1
---

# 系统集成

MatrixHub 采用标准云原生协议设计，旨在无缝融入企业级 AI 基础设施生态，支持主流高性能推理框架。

---

## 🚀 高性能 GPU 推理引擎集成

MatrixHub 作为高性能代理缓存，可以接收内网 GPU 节点的权重加载请求。在启动主流推理框架时，仅需注入重定向环境变量 `HF_ENDPOINT` 即可。详细集成方式请参阅：

- **vLLM**

  在 vLLM 启动环境中设置 `HF_ENDPOINT`，通过 MatrixHub 加载模型。

  [查看 vLLM 详细集成文档](../guides/use-with-vllm.md)

- **SGLang**

  在 SGLang 启动环境中设置 `HF_ENDPOINT`，从 MatrixHub 缓存加载模型。

  [查看 SGLang 详细集成文档](../../blog/sglang-matrixhub-cache-acceleration)

- **llm-d**

  将 `HF_ENDPOINT` 注入 llm-d 的模型服务，在集群内通过 MatrixHub 分发模型。

  [查看 llm-d 详细集成文档](../../blog/llmd-qwen3-32b-matrixhub-cache)

- **Dynamo**

  在 Dynamo 部署中设置 `HF_ENDPOINT`，让推理运行时通过 MatrixHub 获取模型。

  [查看 Dynamo 详细集成文档](../../blog/dynamo-matrixhub-integration)

---

## 模型分发与 P2P 加速

- **ModelExpress 缓存复用**

  以 MatrixHub 作为模型源，通过 ModelExpress 缓存模型下载，供多个 Dynamo Worker 复用。

  [查看 ModelExpress 缓存集成文档](../../blog/dynamo-modelexpress-dedup)

- **Dynamo GPU P2P**

  由 MatrixHub 负责模型文件分发，并通过 ModelExpress、NIXL、UCX 和 RDMA 在 Dynamo Worker 之间传输已加载的 GPU 权重。

  [查看 Dynamo P2P 集成文档](../../blog/matrixhub-modelexpress-dynamo-p2p)
