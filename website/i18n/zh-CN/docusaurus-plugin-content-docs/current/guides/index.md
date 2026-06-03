---
title: 操作指南
sidebar_position: 1
---

# 操作指南

本章节为您汇总了算法研究员、平台运维工程师和 AI 运维团队在日常使用 MatrixHub 时最常用的任务导向型手把手操作指南。

---

## 📦 模型仓库基础操作

了解如何管理、上传、下载和归档具体的大模型权重文件：

*   👉 **[模型的上传与下载](/docs/operations/model-repo/upload-download)**：掌握如何使用标准的 Hugging Face CLI 命令行、Python SDK 以及图形化 Web 控制台来高速读写模型资产。
*   👉 **[模型项目参数配置](/docs/operations/model-repo/project-setting)**：配置模型仓库的可视化参数、可变可见性、标签锁机制及相关的元数据描述。

---

## 👥 项目级隔离与团队治理

了解如何规划逻辑项目并为团队成员映射安全权限边界：

*   👉 **[项目的创建与销毁](/docs/operations/project-management/create-delete)**：了解如何创建安全项目，从而对不同算法团队的研发范围执行逻辑物理隔离。
*   👉 **[项目成员角色与权限映射](/docs/operations/project-management/members)**：邀请团队成员加入您的项目，并依据其实际职责精准赋予角色权限（拥有者 Owner、管理员 Manager、访问者 Reporter）。

---

## 🔑 个人账户安全与 CLI 密钥授权

为自动化工作流或个人开发机配置高强度调用凭证：

*   👉 **[API 访问令牌管理](/docs/operations/profile/access-token)**：了解如何生成、管理并定期轮换您的私有 API 访问令牌，以便对 `huggingface-cli` 命令行或 CI/CD 自动部署流水线执行授权。

---

## ⚙️ 平台系统高级运维管理

面向平台基础设施工程师的高级配置、账号治理及多地域数据分发指南：

*   👉 **[全局账号与安全管理](/docs/operations/platform-settings/user-management)**：全局用户账户治理、企业级 LDAP/SSO 单点登录对接及全局用户存储配额审计。
*   👉 **[仓库管理](/docs/operations/platform-settings/registry-management)**：配置平台访问外部模型源（如 Hugging Face、ModelScope 或企业内部源）的连接信息。
*   👉 **[跨地域大模型远程同步](/docs/operations/platform-settings/remote-sync)**：基于策略的跨数据中心大文件多线程并发异步复制机制，包含分块并发和断点续传的配置指南。
