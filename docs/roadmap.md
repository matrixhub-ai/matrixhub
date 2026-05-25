# MatrixHub roadmap

This page describes the **public, high-level** direction for MatrixHub. It is maintained in the repo alongside other contributor docs; timelines shift with releases and community feedback.

For day-to-day work, see [GitHub issues](https://github.com/matrixhub-ai/matrixhub/issues), [releases](https://github.com/matrixhub-ai/matrixhub/releases), and the documentation site at [matrixhub.ai](https://matrixhub.ai).

## Current focus

- Reliability and performance for large-model distribution and Hugging Face–compatible access
- Kubernetes and Helm deployment ergonomics
- Operational completeness for the four enterprise workflows listed below

## Key workflows we target

- **Intranet inference acceleration** — pull-once, serve-all caching for large model fan-out on GPU clusters
- **Air-gapped model transfer** — controlled export and import of approved models into isolated networks
- **Enterprise model artifact governance** — tag locking, promotion, audit, and CI/CD-friendly access
- **Cross-region distribution** — policy-driven, resumable replication between data centers

## Scope

### V0 must-have

- Hugging Face–compatible API subset required by vLLM, SGLang, and common HF clients
- Model repository creation, upload, download, delete, and visibility controls
- Large-file storage on local filesystem, NFS, and S3-compatible backends
- Proxy cache mode for public Hugging Face sources
- Basic Web UI for repository browsing and administration
- Token-based access control and project / namespace isolation
- Basic audit logging
- Export and import flow for air-gapped environments
- Replication foundation with chunked transfer, resume, and retry
- Deployment paths for Docker Compose and Kubernetes Helm

### V1 should-have

- Dataset repository support
- Role-based access control with more granular permissions
- Storage quotas and cleanup policies
- LDAP, OIDC, or SSO integration
- Access statistics and usage trends
- Security scanning for malicious model content
- Model signing and signature verification
- Release management concepts such as tag locking and promotion workflow
- CDN-friendly download acceleration

### Later or exploratory

- OCI artifact packaging for models
- P2P distribution for startup storms
- Direct-to-GPU loading patterns (NetLoader-style)
- Kubernetes-native acceleration components for vLLM and SGLang
- Automatic upstream mirror selection based on geography or latency
- Deeper integration with inference-serving ecosystems
- ModelScope compatibility where strategically useful

## Milestones

### M0 — Private hub baseline

- Basic repository CRUD
- Local and S3-compatible storage
- API token authentication
- Minimal Web UI
- Hugging Face–compatible read path for core clients

### M1 — Enterprise distribution baseline

- Proxy cache mode
- Project isolation
- Audit logging
- Air-gapped export and import
- Initial replication worker with chunked transfer and resume

### M2 — Production governance

- Stronger RBAC
- Tag locking or release promotion
- Quotas and cleanup
- LDAP or OIDC integration
- Security scanning and signing foundation

### M3 — Inference-native acceleration

- Distribution optimizations for startup storms
- Deeper vLLM and SGLang integration
- Evaluation of P2P, OCI packaging, and direct-load acceleration

## Non-goals

To stay focused on the workflows above, MatrixHub explicitly does **not** aim to be:

- A general-purpose MLOps platform
- A training orchestration or experiment tracking system
- A public community model-sharing platform
- Fully feature-equivalent to Hugging Face on day one
- A platform pursuing broad multi-platform compatibility that does not materially help vLLM, SGLang, or core enterprise workflows

## Success criteria

MatrixHub is succeeding when:

- An enterprise team can switch inference clients to MatrixHub with minimal client-side changes
- A large internal cluster can fan out one large model without repeated public downloads
- An air-gapped organization can move approved models through a controlled import and export process
- A production team can treat models as governed release artifacts rather than loose files
- Cross-region replication is good enough to become part of normal operations instead of an exceptional manual task
- The open-source project is seen as a complete self-hosted option for this category

## How we plan

- This page describes direction, not commitments to specific dates.
- Concrete work is tracked in [GitHub issues](https://github.com/matrixhub-ai/matrixhub/issues) and shipped via [releases](https://github.com/matrixhub-ai/matrixhub/releases).
- Feedback and proposals are welcome via issues or [GitHub Discussions](https://github.com/matrixhub-ai/matrixhub/discussions).
