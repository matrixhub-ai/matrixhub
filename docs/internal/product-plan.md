# MatrixHub Product Plan

## Document Status

- Audience: internal team only
- Purpose: product planning and scope alignment
- Status: draft

## 1. Goal

MatrixHub is an open-source, self-hosted, Hugging Face-compatible model hub for enterprise inference infrastructure.

The primary goal is to make large-model distribution, governance, and replication practical for teams running vLLM and SGLang in private, air-gapped, and multi-region environments.

This product is not trying to replace every part of Hugging Face. It focuses on the infrastructure layer that enterprise inference teams actually need: private deployment, fast distribution, release management, replication, and policy control.

## 2. Positioning

### Product category

MatrixHub sits between public model hubs and general-purpose artifact registries.

- Compared with public model hubs, MatrixHub is private-deployment-first and enterprise-operations-first.
- Compared with general artifact registries such as Harbor, MatrixHub is optimized for Hugging Face-compatible model access and large-model distribution.
- Compared with existing open-source model hubs, MatrixHub should compete on completeness, operational reliability, and inference-oriented workflows rather than on breadth of unrelated features.

### Target users

- Platform engineers operating enterprise AI infrastructure
- ML infrastructure teams managing internal model distribution
- AI operations teams responsible for release control, audit, and replication
- Research teams working in isolated or regulated environments

### Product thesis

This is not the largest software category, so the winning strategy is not feature explosion. The winning strategy is to become the most usable and operationally complete open-source option in this specific category.

That means:

- strong support for the few workflows that matter most
- clear deployment story
- predictable behavior under large-scale distribution pressure
- long-term maintainability
- English-first external surface for global adoption

## 3. Problem Statement

Enterprise inference environments have a different set of constraints than public model-sharing platforms.

Common pain points:

- Large models are expensive and slow to download repeatedly from public endpoints.
- Production clusters may include dozens or hundreds of nodes starting at the same time.
- Air-gapped and regulated environments need controlled model import and export.
- Fine-tuned model versions become inconsistent across training, testing, and production.
- Cross-region teams need local access without saturating WAN bandwidth.
- Existing registries provide governance but are not optimized for Hugging Face-compatible model consumption.

## 4. Product Principles

- Inference-first, not training-platform-first
- Hugging Face-compatible where it matters operationally
- Optimize for very large artifacts and repeated access patterns
- Default to private deployment and enterprise controls
- Prefer simple, reliable workflows over broad but shallow functionality
- Keep the external product surface English-first

## 5. Key Use Cases

## Use Case 0: Intranet Inference Acceleration

### Scenario

An internal production environment runs a vLLM cluster with 100 GPU servers. A 70B model may exceed 130 GB. If every node pulls from the public Hugging Face endpoint independently, startup becomes slow and public bandwidth becomes a bottleneck.

### Desired workflow

1. Operators point all inference nodes to MatrixHub through `HF_ENDPOINT`.
2. The first request fetches the model from the public source and persists it locally.
3. Subsequent nodes read from MatrixHub over the local network instead of re-downloading from the internet.

### Product value

- reduce cold-start amplification
- reduce external bandwidth pressure
- provide a single operational control point for model access

## Use Case 1: Air-Gapped Model Transfer

### Scenario

Government, defense, or core financial environments need access to open models but operate in isolated networks.

### Desired workflow

1. An administrator deploys MatrixHub in a connected staging or DMZ environment.
2. Required models such as Llama or Qwen are mirrored into the staging hub.
3. The mirrored content is exported and transferred into the isolated production network.
4. The production hub imports the content.
5. Internal users access models through the same Hugging Face-compatible interface without internet access.

### Product value

- preserve air-gap boundaries
- reduce manual model handling overhead
- keep client workflows consistent across connected and disconnected environments

## Use Case 2: Enterprise Model Artifact Management

### Scenario

Multiple internal teams fine-tune models. Versions drift between training, testing, and production. Operators need fixed, governed releases instead of informal file sharing.

### Desired workflow

1. CI pipelines push fine-tuned models into a controlled release project.
2. Teams tag, promote, and lock approved versions.
3. Deployment systems pull by API token and fixed tag or digest.
4. Audit logs track who uploaded, promoted, or changed access.

### Product value

- turn models into governed production artifacts
- improve release reproducibility
- reduce accidental model drift across environments

## Use Case 3: Cross-Region Distribution

### Scenario

Global teams operate compute centers in different regions. Replicating tens of TB of weights and datasets over unstable WAN links is slow and operationally painful.

### Desired workflow

1. One region publishes a release-tagged model.
2. MatrixHub applies a replication policy to asynchronously sync the release to another region.
3. Transfers use chunking, resume, and retry so that network instability does not require restarting from scratch.
4. Users in the destination region pull from the local hub instead of the remote site.

### Product value

- lower cross-region latency
- reduce WAN congestion
- make global collaboration operationally manageable

## 6. Differentiation

### Against public SaaS model hubs

- private deployment
- better fit for internal clusters and controlled networks
- governance and release controls for enterprise operations
- replication and caching designed around infrastructure ownership

### Against general artifact registries

- Hugging Face-compatible access patterns
- model-aware caching and distribution workflows
- future optimization specifically for vLLM and SGLang

### Against existing open-source model hubs

The product should differentiate on operational completeness.

The four key use cases above are the evaluation bar. If an alternative cannot handle those workflows end-to-end, it is not a full solution for the target user.

## 7. Non-Goals

These should remain out of scope for the first product phases:

- a general-purpose MLOps platform
- training orchestration
- experiment tracking
- a public community model-sharing platform
- full Hugging Face feature parity on day one
- broad multi-platform compatibility that does not materially help vLLM, SGLang, or core enterprise workflows

## 8. Scope

### V0 must-have

- Hugging Face-compatible API subset required by vLLM, SGLang, and common Hugging Face clients
- model repository creation, upload, download, delete, and visibility controls
- support for large-file storage on local filesystem, NFS, and S3-compatible backends
- proxy cache mode for public Hugging Face sources
- basic Web UI for repository browsing and administration
- token-based access control
- project or namespace isolation
- basic audit logging
- export and import flow for air-gapped environments
- replication foundation with chunked transfer, resume, and retry
- deployment path for Docker Compose and Kubernetes Helm

### V1 should-have

- dataset repository support
- role-based access control with more granular permissions
- storage quotas and cleanup policies
- LDAP, OIDC, or SSO integration
- access statistics and usage trends
- security scanning for malicious model content
- model signing and signature verification
- release management concepts such as tag locking and promotion workflow
- CDN-friendly download acceleration

### Later or exploratory

- ModelScope compatibility where strategically useful
- OCI artifact packaging for models
- P2P distribution for startup storms
- direct-to-GPU loading patterns inspired by NetLoader
- Kubernetes-native acceleration components for vLLM and SGLang
- automatic upstream mirror selection based on geography or latency
- deeper integration with inference-serving ecosystems

## 9. Architecture Direction

This section describes a proposed architecture direction, not a locked implementation.

### Control plane

- API gateway and routing
- authentication and authorization
- repository metadata management
- project, policy, quota, and audit services
- Web UI and admin workflows

### Data plane

- large-file upload and download path
- cache population path
- replication path
- import and export path for air-gapped transfer
- optional acceleration path for future inference-native loading

### Storage and state

- object store or filesystem for large artifacts
- relational database for metadata, tokens, permissions, tags, and audit events
- Redis or in-memory cache for hot metadata and short-lived coordination
- message queue for replication, warm-up, cleanup, and retryable tasks

### Execution model

- stateless API services where possible
- asynchronous workers for replication, cache warm-up, cleanup, and transfer orchestration
- explicit retry, backoff, and dead-letter handling for long-running transfer tasks

### Design bias

- keep the control plane simple
- move heavy transfer work to asynchronous workers
- prefer object-native data handling over Git-first assumptions when necessary for very large artifacts

## 10. Open Questions and Risks

### API compatibility boundary

How much of the Hugging Face API needs to be implemented to support the real target workflows? Full parity is expensive and may not be necessary.

### Storage model

Should the product lean on Git plus LFS semantics, object-native semantics, or a hybrid abstraction? This decision affects performance, UX, and compatibility.

### Replication complexity

Cross-region replication sounds straightforward at the feature level but is operationally deep. Resume, retry, consistency, observability, and conflict handling must be designed carefully.

### Security posture

If MatrixHub is exposed on the public internet, the security baseline must be closer to an enterprise registry such as Harbor than to an internal-only developer tool.

### Performance proof

The product promise is strongly tied to speed. We need reproducible benchmark scenarios for first pull, hot cache fan-out, and cross-region sync.

### Scope control

There is a high risk of turning this into a broad AI platform. The team should protect the inference-distribution core and reject loosely related features unless they directly strengthen the core use cases.

## 11. Success Criteria

MatrixHub is succeeding if the following are true:

- an enterprise team can switch inference clients to MatrixHub with minimal client-side changes
- a large internal cluster can fan out one large model without repeated public downloads
- an air-gapped organization can move approved models through a controlled import and export process
- a production team can treat models as governed release artifacts rather than loose files
- cross-region replication is good enough to become part of normal operations instead of an exceptional manual task
- the open-source project is seen as the most complete self-hosted option for this category

## 12. Execution Milestones

### Milestone 0: private hub baseline

- basic repository CRUD
- local and S3-compatible storage
- API token authentication
- minimal Web UI
- Hugging Face-compatible read path for core clients

### Milestone 1: enterprise distribution baseline

- proxy cache mode
- project isolation
- audit logging
- air-gapped export and import
- initial replication worker with chunked transfer and resume

### Milestone 2: production governance

- stronger RBAC
- tag locking or release promotion
- quotas and cleanup
- LDAP or OIDC integration
- security scanning and signing foundation

### Milestone 3: inference-native acceleration

- distribution optimizations for startup storms
- deeper vLLM and SGLang integration
- evaluation of P2P, OCI packaging, and direct-load acceleration

## 13. Summary

MatrixHub should be built as a focused infrastructure product.

The core bet is that enterprise inference teams need a private, self-hosted, Hugging Face-compatible hub that is operationally complete for four workflows: intranet acceleration, air-gapped transfer, governed model releases, and cross-region distribution.

If the project stays disciplined around those workflows, it can become the strongest open-source option in this category even if the overall market remains relatively specialized.
