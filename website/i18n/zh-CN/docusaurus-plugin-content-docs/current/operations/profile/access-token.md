---
sidebar_position: 1
---

# 访问令牌 (Access Token)

访问令牌用于在命令行工具中完成身份认证。配置完成后，您可以通过 Hugging Face CLI 访问 MatrixHub 中需要认证的模型仓库。

## 前置条件

- 拥有有效的 MatrixHub 账号。
- 已具备目标私有项目中模型仓库的访问权限，例如 `my-matrixhub-project/test-mn`。如需上传模型，目标项目必须是非代理项目，且当前账号需具备该项目的管理员或开发者权限。
- 本地已安装 Hugging Face CLI，并且可以在终端中执行 `hf auth login`。
- 已获取 MatrixHub 的访问地址，例如 `http://127.0.0.1:3001`。

## 操作步骤

### 创建访问令牌

1. 登录 MatrixHub，进入 **个人中心** -> **访问令牌** 页面。

    ![访问令牌概览](./images/access-token-overview.jpg)

1. 点击 **创建访问令牌**。

    ![创建访问令牌](./images/access-token-create.jpg)

1. 填写令牌名称，选择过期时间，然后点击 **确认**。

1. 创建成功后，系统会显示令牌内容。请立即复制并妥善保存，关闭弹窗后将无法再次查看完整令牌。

    ![保存访问令牌](./images/access-token-save.png)

### 使用访问令牌

1. 在本地终端中配置 MatrixHub 服务端点：

    ```bash
    export HF_ENDPOINT="http://<your-matrixhub-endpoint>"
    ```

1. 执行登录命令：

    ```bash
    hf auth login
    ```

1. 按提示粘贴已保存的访问令牌，完成身份认证。

1. 执行下载命令，验证是否可以访问目标模型仓库：

    ```bash
    hf download my-matrixhub-project/test-mn
    ```

1. 如需下载指定 Tag 对应的模型版本，请在下载命令中通过 `--revision` 指定 Tag 名称：

    ```bash
    hf download my-matrixhub-project/test-mn --revision 0.0.1
    ```

1. 如需上传模型，请先在目标私有项目中创建模型仓库。该项目不能是代理项目。

    ```bash
    hf upload my-matrixhub-project/test-mn ./<local-model-dir> .
    ```

### 撤销访问令牌

1. 进入 **个人中心** -> **访问令牌** 页面。

1. 找到需要撤销的令牌，执行删除操作。

1. 令牌撤销后，使用该令牌认证的命令行会话将无法继续访问需要认证的资源。

:::note

- 访问令牌等同于账号凭据，请勿提交到代码仓库或分享给他人。
- 如果令牌泄露，请立即删除该令牌并重新创建。

:::

## 配置参数

| 参数 | 描述 |
|-----------|-------------|
| 名称 | 令牌的显示名称，建议使用能够体现用途的名称。 |
| 过期时间 | 令牌的有效期，可以设置为 **永不过期** 或指定日期。 |
| Token 值 | 用于命令行认证的密钥字符串，仅在创建成功后完整显示一次。 |
