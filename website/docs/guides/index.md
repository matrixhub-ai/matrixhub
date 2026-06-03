---
sidebar_position: 1
---

# Guides

This section provides task-oriented, step-by-step guides for developers, platform engineers, and AI operations teams working with MatrixHub.

---

## 📦 Model Repository Operations

Learn how to manage and distribute actual model files and weight repositories:

*   👉 **[Upload & Download Models](/docs/operations/model-repo/upload-download)**: Learn how to upload custom fine-tuned weights or download cached weights using standard Hugging Face tools, the Python SDK, and Web UI.
*   👉 **[Model Project Settings](/docs/operations/model-repo/project-setting)**: Configure visibility parameters, tag locks, and metadata descriptors for your model repositories.

---

## 👥 Project & Member Governance

Manage access control scopes and team collaborations:

*   👉 **[Project Lifecycle](/docs/operations/project-management/create-delete)**: Step-by-step guide on creating secure logical projects to isolate different research teams or model domains.
*   👉 **[Member Roles & RBAC Mapping](/docs/operations/project-management/members)**: Invite members to your workspace projects and assign specific permission roles (Owner, Manager, Reporter).

---

## 🔑 User Profiles & CLI Authentication

Generate secure credentials for command-line tools:

*   👉 **[Access Tokens](/docs/operations/profile/access-token)**: Learn how to create, configure, and rotate private API tokens to authenticate `huggingface-cli` or pipeline scripts.

---

## ⚙️ Platform Administration

Enterprise settings and multi-region replication controls for platform engineers:

*   👉 **[User & Account Management](/docs/operations/platform-settings/user-management)**: Manage global accounts, set usage quotas, and configure corporate single sign-on (SSO/LDAP) configurations.
*   👉 **[Registry Management](/docs/operations/platform-settings/registry-management)**: Configure upstream connection endpoints and credentials to access external registry sources (Hugging Face, ModelScope).
*   👉 **[Cross-Region Remote Sync](/docs/operations/platform-settings/remote-sync)**: Configure asynchronous, policy-driven remote replication links between geographical data center hubs.
