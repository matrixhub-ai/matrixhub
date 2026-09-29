# SSH 公钥

SSH 公钥用于 Git 身份认证。配置完成后，无需输入访问令牌即可通过 SSH 克隆和推送模型仓库。

## 前置条件

- 拥有有效的 MatrixHub 账号，并且具备目标模型仓库的访问权限，例如 `my-matrixhub-project/test-mn`。
- 本地已安装 Git。如果仓库包含大模型文件，还需安装 Git LFS。
- 已通过 Docker Compose 或 Helm 对外暴露 MatrixHub SSH 服务。

## 查看已有 SSH 公钥

在生成新的 SSH 公钥之前，请先检查本地用户根目录中是否已有可用的 SSH 公钥。

对于 Linux 和 macOS，可使用以下命令查看已有的公钥。Windows 用户可在 WSL（需要 Windows 10 或更高版本）或 Git Bash 中使用以下命令查看已生成的公钥。

- **ED25519 算法：**

    ```bash
    cat ~/.ssh/id_ed25519.pub
    ```

- **RSA 算法：**

    ```bash
    cat ~/.ssh/id_rsa.pub
    ```

如果返回了一段以 `ssh-ed25519` 或 `ssh-rsa` 开头的长字符串，则说明本地已存在 SSH 公钥。

此时可跳过[生成 SSH 公钥](#生成-ssh-公钥)，直接前往[复制公钥](#复制公钥)。

## 生成 SSH 公钥

如果无法查看到已有的 SSH 公钥，则表示本地尚未生成可用的 SSH 公钥，需要重新生成。请按照以下步骤操作：

1. 打开终端（Windows 用户请使用 [WSL](https://docs.microsoft.com/en-us/windows/wsl/install) 或 [Git Bash](https://gitforwindows.org/)），执行 `ssh-keygen -t` 命令。

2. 输入密钥算法类型，并可选择添加注释（Comment）。

    注释会写入 `.pub` 公钥文件中，通常建议使用电子邮箱地址作为注释内容。

    - 使用 `ED25519` 算法生成密钥对：

        ```bash
        ssh-keygen -t ed25519 -C "<comment>"
        ```

    - 使用 `RSA` 算法生成密钥对：

        ```bash
        ssh-keygen -t rsa -C "<comment>"
        ```

3. 按 Enter 键选择 SSH 公钥的保存路径。

    以 ED25519 算法为例，默认路径如下：

    ```console
    Generating public/private ed25519 key pair.
    Enter file in which to save the key (/home/user/.ssh/id_ed25519):
    ```

    默认私钥保存路径为 `/home/user/.ssh/id_ed25519`，对应的公钥保存路径为 `/home/user/.ssh/id_ed25519.pub`。

4. 为密钥设置口令（Passphrase）。

    ```console
    Enter passphrase (empty for no passphrase):
    Enter same passphrase again:
    ```

    默认情况下，口令为空。你也可以为私钥设置口令，以增强私钥文件的安全性。

    如果希望每次通过 SSH 协议访问代码仓库时无需输入口令，可在创建密钥时直接留空。

5. 按 Enter 键完成密钥对的创建。

## 复制公钥

除了手动复制命令行输出的公钥内容外，还可以根据不同操作系统使用以下命令将公钥复制到剪贴板。

- Windows（在 [WSL](https://docs.microsoft.com/en-us/windows/wsl/install) 或 [Git Bash](https://gitforwindows.org/) 中）：

    ```bash
    cat ~/.ssh/id_ed25519.pub | clip
    ```

- macOS：

    ```bash
    tr -d '\n'< ~/.ssh/id_ed25519.pub | pbcopy
    ```

- GNU/Linux（需安装 xclip）：

    ```bash
    xclip -sel clip < ~/.ssh/id_ed25519.pub
    ```

## 在 MatrixHub 中设置公钥

1. 登录 MatrixHub UI，依次选择 **个人中心** -> **SSH 公钥** -> **导入 SSH 公钥**。

2. 在弹出的窗口中填写相关信息，然后单击 **确定**。

## 使用 SSH 公钥

SSH 仓库路径格式为 `<项目>/<模型>.git`。请根据部署方式使用对应的 SSH 地址。

### Docker Compose

Docker Compose 默认将 SSH 服务暴露在 `2222` 端口。请将 `<matrixhub-host>` 替换为 MatrixHub 所在主机的 IP 地址或域名：

```bash
git clone -c core.sshCommand="ssh -p 2222" git@<matrixhub-host>:my-matrixhub-project/test-mn.git
```

如果启动 MatrixHub 时设置了 `MATRIXHUB_SSH_PORT`，请使用配置的端口替换 `2222`。

### Helm

如果 Helm 部署使用默认的 `NodePort` 配置，请使用 Kubernetes 节点 IP 和 `30022` 端口：

```bash
git clone -c core.sshCommand="ssh -p 30022" git@<node-ip>:my-matrixhub-project/test-mn.git
```

可以运行 `kubectl get nodes -o wide` 查看节点 IP。如果修改了 `apiserver.service.sshNodePort`，请使用配置的端口替换 `30022`。

### 可选：配置 SSH 以便长期使用

如果需要经常使用 MatrixHub，可以在 `~/.ssh/config` 中添加别名，并填写部署对应的 SSH 端口（Docker Compose 默认为 `2222`，Helm NodePort 默认为 `30022`）：

```text
Host matrixhub
  HostName <matrixhub-host-or-node-ip>
  Port <ssh-port>
  User git
```

之后可以使用更短的命令克隆仓库：

```bash
git clone matrixhub:my-matrixhub-project/test-mn.git
```

克隆完成后，可以使用标准 Git 命令提交并推送修改：

```bash
cd test-mn
git add .
git commit -m "Update model"
git push
```
