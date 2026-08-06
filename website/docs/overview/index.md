---
sidebar_position: 1
---

# Overview

MatrixHub is an open-source, self-hosted, Hugging Face-compatible model hub for enterprise inference infrastructure.

## Target Users

- 🏗️ **Platform Engineers**: Operate enterprise AI infrastructure.
- 📦 **ML Infrastructure Teams**: Manage internal model distribution.
- 🛡️ **AI Operations Teams**: Own release control, audit, and replication.
- 🔬 **Research Teams**: Work in isolated or regulated environments.

## Use Cases

- 🚀 **Intranet Inference Acceleration**: Cache models once and distribute them efficiently across internal GPU clusters.
- 🔐 **Air-Gapped Model Delivery**: Transfer approved models into isolated or regulated environments.
- 📦 **Enterprise Model Governance**: Control releases, audit activity, and manage model versions from development to production.
- 🌍 **Multi-Region Replication**: Replicate models between data centers with resumable transfers.

## Key Capabilities

### 🚀 Model Distribution

- **Hugging Face-Compatible Access**: Point `HF_ENDPOINT` to MatrixHub to use `hf download` and `hf upload`, and connect inference engines such as vLLM, SGLang, and Dynamo.
- **On-Demand Proxy Cache**: Cache public Hugging Face models on first request.
- **Repository Workflows**: Create and manage model repositories.

### 🛡️ Access and Project Management

- **Project Isolation**: Organize repositories by project and permissions.
- **Authentication and Roles**: Manage users, roles, tokens, and robot accounts.
- **Git Access**: Access repositories through Git, Git LFS, and HTTP.

### 🌍 Deployment and Replication

- **Replication Policies**: Run manual or scheduled pull and push synchronization, with a foundation for resumable transfers.
- **Flexible Storage**: Use local PVC or NFS storage.
- **Deployment Options**: Deploy with Docker Compose or Kubernetes Helm charts.

For planned capabilities, see the [MatrixHub roadmap](https://github.com/matrixhub-ai/matrixhub/blob/main/ROADMAP.md).

## Live Demo

Try MatrixHub instantly at **[demo.matrixhub.ai](https://demo.matrixhub.ai/)**—no setup required.

Sign in with the public demo credentials: username `admin`, password `changeme`. The demo is for evaluation only, resets every six hours, and may also be reset during maintenance.

## Getting Started

MatrixHub is easy to deploy using **Docker Compose** or **Kubernetes**. The entire infrastructure is open source and free for the community.

👉 **Ready to get started?** Follow the [Getting Started guide](../getting-started/) to deploy MatrixHub and begin using it.
