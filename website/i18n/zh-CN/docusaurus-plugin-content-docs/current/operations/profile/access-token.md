---
sidebar_position: 1
---

# 访问令牌 (Access Token)

## 前置条件

- 拥有有效的 MatrixHub 账号。
- 至少拥有一个私有仓库或公共仓库的访问权限（例如：`my-matrixhub-project/test-mn`）。
- 本地已安装 Hugging Face CLI，且终端中可以使用 `hf auth login` 命令。

## 操作步骤

### 创建访问令牌

1. 登录 MatrixHub 平台。进入 **个人中心** -> **访问令牌 (Access Token)** 页面。

    ![访问令牌概览](./images/access-token-overview.jpg)

1. 点击 **创建访问令牌**，填写名称（例如：`demo`），选择过期时间（例如：**永不过期** 或特定时长），然后点击 **确认**。

    ![创建访问令牌](./images/access-token-create.jpg)

1. 创建成功后，将弹出一个窗口显示 Token。**请立即复制并妥善保存**，因为它将不再显示。

    ![保存访问令牌](./images/access-token-save.png)

### 使用访问令牌

1. 使用您的 MatrixHub 地址在本地终端中配置服务端点：

    ```bash
    export HF_ENDPOINT="matrixhub.url" # 示例: http://127.0.0.1:xxx
    ```

1. 运行登录命令：

    ```bash
    hf auth login
    ```

1. 根据提示输入您保存的 Token 以完成身份验证。

1. 执行下载命令以验证对 **MatrixHub** 仓库的访问权限：

    ```bash
    hf download my-matrixhub-project/test-mn
    ```

### 撤销访问令牌

1. 进入 **个人中心** -> **访问令牌** 页面，找到目标 Token，并执行删除操作。

1. 撤销后，任何需要该 Token 进行身份验证的 CLI 操作都会提示您尚未登录或身份验证无效。

## 配置参数

| 参数 | 描述 |
|-----------|-------------|
| 名称 | Token 的描述性标识符，用于区分不同用途。 |
| 过期时间 | 可以设置为 **永不过期** 或自定义日期。超过该期限后将自动失效。 |
| Token 值 | 用于身份验证的实际密钥字符串。仅在创建时完整显示一次，必须立即复制并保存。 |
