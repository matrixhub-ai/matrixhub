# SSH 公钥

本页介绍如何配置 SSH 公钥。

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
