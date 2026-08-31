---
sidebar_position: 1
---

# Security

This guide covers MatrixHub's current project-level access isolation, personal and robot credentials, and role-based access control.

---

## 🔒 1. Multi-Tenant Project Isolation

MatrixHub uses a **Project** as the boundary for model resources and permissions.

*   **Project Visibility**: All users can view and pull models from public projects. Private projects are available only to project members and platform administrators. Pushing and deleting still require the appropriate project role.
*   **Personal Access Tokens**: Represent the owning user, are not bound to a single project, and use that user's access scope and role permissions in each project.
*   **Robot Accounts**: Project permissions can apply to all projects or selected projects. In selected-project mode, the configured project permissions apply only within those projects; existing read-only access to public projects is unaffected.

Configuration references: [Create and Delete Projects](../operations/project-management/create-delete.md), [Personal Access Tokens](../operations/profile/access-token.md), and [Robot Accounts](../operations/platform-settings/robot-accounts.md).

---

## 🛡️ 2. Fine-Grained Role-Based Access Control (RBAC)

MatrixHub separates **platform-level roles** from **project-level roles**. A platform administrator (`admin`) has global platform permissions, while access to models within a project is controlled by three project roles:

| Role | Permission Scope |
| :--- | :--- |
| **Project Viewer** | View the project and member list, and view and pull models. Cannot push or delete models or manage project members. |
| **Project Editor** | Includes Project Viewer permissions and can push models. Cannot delete models or manage project members. |
| **Project Admin** | Includes Project Editor permissions and can add or remove members, change member roles, update project settings, and delete models and the project. |
| **Platform Admin (`admin`)** | Can use platform administration features and perform all operations within every project. |

Every user who can sign in to MatrixHub can create a project and automatically becomes the Project Admin of that project. Members cannot remove themselves from the current project.

Configuration reference: [Project Members and Roles](../operations/project-management/members.md).

---

## 🚧 3. Planned Security Capabilities

The following capabilities are not available in the current release and are listed in the [MatrixHub Roadmap](https://github.com/matrixhub-ai/matrixhub/blob/main/ROADMAP.md):

*   LDAP, OIDC, or SSO integration
*   Audit logging
*   Malicious model content scanning, model signing, and signature verification
