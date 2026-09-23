---
title: 采用情况
sidebar_position: 7
---

# 采用情况

MatrixHub 是一个面向企业级 AI 推理基础设施的开源、自托管、兼容 Hugging Face 的模型仓库。

## 采用者

此列表仅包含同意公开其组织名称的组织。如需添加或移除你的组织，请提交更新此页面的拉取请求。

**[DaoCloud](https://daocloud.io/)** 在其 Token Factory 平台上以私有模型中心的方式运行 MatrixHub，支持 Hugging Face 兼容访问、内网模型缓存、项目级仓库和运维工作流。

## 外部集成文档

[vLLM 的 MatrixHub 文档](https://docs.vllm.ai/en/latest/models/supported_models/#matrixhub)说明了如何将 `HF_ENDPOINT` 指向 MatrixHub 实例，使 `vllm serve` 通过内网而非公共 Hugging Face Hub 获取模型权重。

[NVIDIA Dynamo 的 MatrixHub 模型加载指南](https://docs.nvidia.com/dynamo/dev/kubernetes/model-deployment/model-loading/matrix-hub)展示了如何配置 Kubernetes 部署，使 Dynamo 的前端和 worker 组件通过 MatrixHub 加载已缓存的模型权重。

[llm-d 的自托管模型仓库指南](https://llm-d.ai/docs/dev/operations/model-loading-and-startup#self-hosted-registry-with-matrixhub)说明了如何在 `modelserver.env` 中配置 `HF_ENDPOINT`，使 llm-d 的模型服务 Pod 通过 MatrixHub 加载已缓存的模型权重。

## MatrixHub 参考资料

- [目标用户与使用场景](../overview/)
- [集成指南](../integrations/)
- [技术设计文档](https://github.com/matrixhub-ai/matrixhub/blob/main/docs/design/README.md)

## 添加你的组织

如需添加你的组织，请提交更新此页面的拉取请求，欢迎贡献。
