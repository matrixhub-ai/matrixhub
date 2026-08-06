# MatrixHub Roadmap

## Overview

MatrixHub is an open-source, self-hosted, Hugging Face–compatible model hub for enterprise inference infrastructure.

## Four Key Workflows We Target

- **Intranet inference acceleration** — pull-once, serve-all caching for large model fan-out on GPU clusters
- **Air-gapped model transfer** — controlled export and import of approved models into isolated networks
- **Enterprise model artifact governance** — tag locking, promotion, audit, and CI/CD-friendly access
- **Cross-region distribution** — policy-driven, resumable replication between data centers

## Phases

### Phase 1: Key Workflows and Enterprise Feature Baseline

- Hugging Face–compatible API for vLLM, SGLang, and common Hugging Face clients (`hf` download/upload via `HF_ENDPOINT`)
- Pull-once, serve-all proxy cache for public Hugging Face sources
- Projects and model repositories with CRUD, visibility controls, and tags
- Users, roles, access keys, and robot accounts
- Web UI for repository browsing and administration
- Pull- and push-mode replication with manual or scheduled sync policies, including MatrixHub-to-MatrixHub replication
- Git access to model repositories over HTTP/SSH (clone, pull, tag, push) with SSH key management
- Large-file storage on local filesystem, NFS, and S3-compatible backends
- Replication foundation with chunked transfer, resume, and retry
- Deployment via Docker Compose, Helm, and Kubernetes

### Phase 2: Inference-Native Acceleration and Ecosystem Integration

- S3-compatible backends support for large-file storage
- XET support for large-file download
- Reliability and performance for large-model distribution
- Visibility into in-progress uploads and downloads
- P2P pre-warm acceleration across nodes via Dragonfly
- Preset recommended models with Hugging Face mirror support
- Ecosystem documentation and examples for MatrixHub integration with vLLM, SGLang, llm-d, Dynamo, ModelExpress, and the Kubernetes ecosystem
- Run:ai Model Streamer support, with MatrixHub as the governed, streamer-friendly model source for vLLM

### Phase 3: Enterprise Feature Expansion and Production Governance

- Dataset support
- Audit logging
- Role-based access control with more granular permissions
- LDAP, OIDC, or SSO integration
- Storage quotas
- Cleanup policies
- Access statistics and usage trends
- Security scanning for malicious model content
- Model signing and signature verification
- Release management concepts such as tag locking and promotion workflow
- CDN-friendly download acceleration

## Later or Exploratory

- ModelScope compatibility where strategically useful
- OCI artifact packaging for models
- More P2P distribution approach for startup storms
- More direct-to-GPU loading patterns (NetLoader-style) approach
- Kubernetes-native acceleration components for vLLM and SGLang
- Automatic upstream mirror selection based on geography or latency
- Deeper integration with inference-serving ecosystems

## Non-Goals

- A general-purpose MLOps platform
- A training orchestration or experiment tracking system
- A public community model-sharing platform
- Full Hugging Face feature parity on day one

## How We Plan

This roadmap describes direction, not commitments to specific dates. Concrete work is tracked in [GitHub issues](https://github.com/matrixhub-ai/matrixhub/issues) and [milestones](https://github.com/matrixhub-ai/matrixhub/milestones), and shipped via [releases](https://github.com/matrixhub-ai/matrixhub/releases).
