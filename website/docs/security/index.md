---
sidebar_position: 1
---

# Security

This guide outlines the built-in security, authentication, authorization, and compliance features engineered within MatrixHub to safeguard enterprise AI model assets.

---

## 🔒 1. Multi-Tenant Project Isolation

MatrixHub segregates all resources into logical boundaries called **Workspace Projects**.
*   **Encapsulated Credentials**: API tokens, user memberships, and repository permissions are scoped strictly within the project. A token belonging to `project-a` cannot download or even query model metadata stored inside `project-b`.
*   **Network Segregation**: Allows mapping local GPU clusters to specific Projects, ensuring compute nodes only access approved weights.

---

## 👤 2. Authentication & Identity Providers (SSO)

We integrate seamlessly with standard enterprise user directories to ensure unified credential management:
*   **LDAP / Active Directory**: Authenticate engineers and administrators using their standard corporate user directory.
*   **OIDC / OAuth 2.0**: Integrate single-sign-on (SSO) providers (e.g. Okta, Keycloak, Ping Identity) to manage dashboard access securely.

---

## 🛡️ 3. Fine-Grained Role-Based Access Control (RBAC)

Access control inside projects is managed through three predefined functional roles:

| Project Role | Allowed Actions | Typical Assignment |
| :--- | :--- | :--- |
| **Owner** | Full admin control, invite/delete members, delete repositories, configure replication sync links. | Platform Engineers, Devops Leads |
| **Manager** | Upload weights, download models, edit repository descriptions, commit and toggle tag locks. | ML Engineers, Algorithm Researchers |
| **Reporter** | Read-only access to query metadata and download cached/locked model weights. | Automated GPU Compute Nodes, CI pipelines |

---

## 📋 4. Compliance Audits & Trail Logging

To satisfy strict regulatory audits (SOC2, ISO 27001, financial guidelines), MatrixHub records an **immutable audit trail** for all system-level and API-level events.

Every log entry captures:
*   **Who**: The authenticated user or API token ID.
*   **What**: The exact action (e.g., `model.upload`, `tag.lock`, `token.create`).
*   **Where**: Client IP address and geographical metadata.
*   **When**: Highly accurate cryptographic timestamp.

Audit logs cannot be modified or deleted by project Managers or Reporters, providing reliable forensic trails.

---

## 🦠 5. Malware Scanning & Code Integrity Checks

AI models (especially standard PyTorch pickle serializations) can act as arbitrary code execution vectors. 

MatrixHub integrates active vulnerability scanning:
*   **Malicious Code Scan**: Automatically scans uploaded weights (Safetensors and pickle checkpoints) upon upload to detect malicious system calls or payloads.
*   **Cryptographic Model Signing**: Generates cryptographic signatures for approved weights. Compute servers automatically verify these signatures upon fetching, rejecting any tampered weights.
