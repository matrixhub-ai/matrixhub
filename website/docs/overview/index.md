---
sidebar_position: 1
---

# Overview

MatrixHub is an open-source, self-hosted AI model registry engineered for large-scale enterprise inference. It serves as a drop-in private replacement for Hugging Face, purpose-built to accelerate **vLLM** and **SGLang** workloads.

This section provides a high-level overview of the MatrixHub project, its scope, roadmap, and operational success criteria.

## Key Target Workflows

To stay focused on what enterprise operations actually need, MatrixHub targets four essential workflows:

*   **Intranet Inference Acceleration** — Pull-once, serve-all caching for large model fan-out on local GPU clusters to eliminate external bandwidth bottlenecks.
*   **Air-Gapped Model Transfer** — Controlled export and import pipelines to safely ferry approved models into isolated and highly regulated networks.
*   **Enterprise Model Asset Governance** — Tag locking, promotion, audit trails, and CI/CD-friendly access control to ensure consistency from training to production.
*   **Cross-Region Distribution** — Policy-driven, chunked, and resumable replication between geographical data centers.

## Current Focus & Scope

We prioritize reliability and performance for large-model distribution and Hugging Face–compatible access, alongside Kubernetes/Helm deployment ergonomics.

### Project Roadmap & Milestones

The project's evolution is divided into clear operational milestones:

1.  **Milestone 0: Private Hub Baseline**
    *   Basic repository CRUD operations.
    *   Support for local and S3-compatible storage.
    *   API token authentication and a minimal Web UI.
    *   Hugging Face-compatible read path for core client libraries.
2.  **Milestone 1: Enterprise Distribution Baseline**
    *   Transparent proxy caching mode.
    *   Project and namespace isolation.
    *   Audit logging.
    *   Air-gapped export/import workflows.
    *   Initial replication engine supporting chunked transfer and resume.
3.  **Milestone 2: Production Governance**
    *   Granular Role-Based Access Control (RBAC).
    *   Tag locking and release promotion workflows.
    *   Storage quotas and automatic cleanup policies.
    *   LDAP, OIDC, and SSO identity integrations.
    *   Malware scanning and model integrity signing.
4.  **Milestone 3: Inference-Native Acceleration**
    *   Distribution optimization for GPU startup storms (P2P distribution, etc.).
    *   Deep Kubernetes-native integrations with vLLM and SGLang.
    *   Exploratory net-loading streaming directly to GPU weights.

## Project Scope Boundaries (Non-Goals)

To remain highly focused and maintain long-term reliability, MatrixHub explicitly does **not** aim to be:
*   A general-purpose MLOps platform.
*   A training orchestration or experiment tracking system.
*   A public community model-sharing platform.
*   Fully feature-equivalent to Hugging Face on day one (focus is strictly on the inference and distribution subset).

## Success Criteria

We measure the success of MatrixHub by how well it simplifies operations:
*   Inference clients can switch to MatrixHub with zero code changes (by simply redirecting `HF_ENDPOINT`).
*   A large internal GPU cluster can boot a 70B+ model without saturating external network links.
*   An air-gapped organization can move approved models through a safe, controlled import/export pipeline.
*   A production team can treat models as governed, immutable release artifacts rather than loose files.
