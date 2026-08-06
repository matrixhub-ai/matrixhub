---
title: 概览
sidebar_position: 1
---

# 项目概览

MatrixHub 是一个面向企业级推理基础设施的开源模型中心，支持自托管并兼容 Hugging Face。

## 目标用户

- 🏗️ **平台工程师**：负责企业 AI 基础设施的运营。
- 📦 **机器学习基础设施团队**：管理内部模型分发。
- 🛡️ **AI 运维团队**：负责发布控制、审计和复制。
- 🔬 **研究团队**：在隔离或受监管的环境中工作。

## 使用场景

- 🚀 **内网推理加速**：仅需缓存一次模型，即可将其高效分发到内部 GPU 集群。
- 🔐 **物理隔离环境中的模型交付**：将审核通过的模型传输到隔离或受监管的环境中。
- 📦 **企业模型治理**：控制模型发布、审计操作，并管理模型从开发到生产的版本。
- 🌍 **多区域复制**：通过可断点续传的方式在数据中心之间复制模型。

## 核心能力

### 🚀 模型分发

- **兼容 Hugging Face 的访问方式**：将 `HF_ENDPOINT` 指向 MatrixHub，即可使用 `hf download`、`hf upload`，并对接 vLLM、SGLang、Dynamo 等推理引擎。
- **按需代理缓存**：首次请求时缓存 Hugging Face 公共模型。
- **仓库工作流**：创建和管理模型仓库。

### 🛡️ 访问与项目管理

- **项目隔离**：按项目和权限组织仓库。
- **身份验证与角色**：管理用户、角色、令牌和机器人账号。
- **Git 访问**：通过 Git、Git LFS 和 HTTP 访问仓库。

### 🌍 部署与复制

- **复制策略**：手动或定时执行拉取和推送同步，并具备断点续传基础能力。
- **灵活存储**：使用本地 PVC 或 NFS 存储。
- **部署方式**：通过 Docker Compose 或 Kubernetes Helm Chart 部署。

计划中的能力请参阅 [MatrixHub 路线图](https://github.com/matrixhub-ai/matrixhub/blob/main/ROADMAP.md)。

## 在线体验

无需安装，立即访问 **[demo.matrixhub.ai](https://demo.matrixhub.ai/)** 体验 MatrixHub。

使用公开演示账号登录：用户名 `admin`，密码 `changeme`。演示环境仅供评估，每六小时重置一次，也可能因维护而临时重置。

## 快速开始

MatrixHub 使用 **Docker Compose** 或 **Kubernetes** 部署非常简单。整个基础设施开源且对社区完全免费。

👉 **准备好开始了吗？** 按照[快速开始指南](../getting-started/)部署并开始使用 MatrixHub。
