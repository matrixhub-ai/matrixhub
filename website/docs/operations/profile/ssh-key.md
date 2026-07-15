# SSH Public Key

This page explains how to configure your SSH public key.

## View Existing SSH Keys

Before generating a new SSH key, check if you need to use an existing SSH key stored in the root directory of the local user.
For Linux and Mac, use the following command to view existing public keys. Windows users can use the
following command in WSL (requires Windows 10 or above) or Git Bash to view the generated public keys.

- **ED25519 Algorithm:**

    ```bash
    cat ~/.ssh/id_ed25519.pub
    ```

- **RSA Algorithm:**

    ```bash
    cat ~/.ssh/id_rsa.pub
    ```

If a long string starting with ssh-ed25519 or ssh-rsa is returned, it means that a local public key already exists.
You can skip [Generate SSH Key](#generate-ssh-key) and proceed directly to [Copy the Public Key](#copy-the-public-key).

## Generate SSH Key

If you cannot view the existing SSH key, it means that
there is no available SSH key locally and a new SSH key needs to be generated. Follow these steps:

1. Access the terminal (Windows users use [WSL](https://docs.microsoft.com/en-us/windows/wsl/install) or
   [Git Bash](https://gitforwindows.org/)), and run `ssh-keygen -t`.

2. Enter the key algorithm type and an optional comment.

    The comment will appear in the .pub file and can generally use the email address as the comment content.

    - To generate a key pair based on the `ED25519` algorithm, use the following command:

        ```bash
        ssh-keygen -t ed25519 -C "<comment>"
        ```

    - To generate a key pair based on the `RSA` algorithm, use the following command:

        ```bash
        ssh-keygen -t rsa -C "<comment>"
        ```

3. Press Enter to choose the SSH key generation path.

    Taking the ED25519 algorithm as an example, the default path is as follows:

    ```console
    Generating public/private ed25519 key pair.
    Enter file in which to save the key (/home/user/.ssh/id_ed25519):
    ```

    The default key generation path is `/home/user/.ssh/id_ed25519`, and the corresponding public key is `/home/user/.ssh/id_ed25519.pub`.

4. Set a passphrase for the key.

    ```console
    Enter passphrase (empty for no passphrase):
    Enter same passphrase again:
    ```

    The passphrase is empty by default, and you can choose to use a passphrase to protect the private key file. 
    If you do not want to enter a passphrase every time you access the repository using the SSH protocol,
    you can enter an empty passphrase when creating the key.

5. Press Enter to complete the key pair creation.

## Copy the Public Key

In addition to manually copying the generated public key information printed on the command line,
you can use the following commands to copy the public key to the clipboard, depending on your operating system.

- Windows (in [WSL](https://docs.microsoft.com/en-us/windows/wsl/install) or [Git Bash](https://gitforwindows.org/)):

    ```bash
    cat ~/.ssh/id_ed25519.pub | clip
    ```

- Mac:

    ```bash
    tr -d '\n'< ~/.ssh/id_ed25519.pub | pbcopy
    ```

- GNU/Linux (requires xclip):

    ```bash
    xclip -sel clip < ~/.ssh/id_ed25519.pub
    ```

## Set the Public Key on MatrixHub

1. Log in to the MatrixHub UI page and select **Profile** -> **SSH Public Key** -> **Import SSH Public Key**.

2. Fill the required information in the pop-up window and click **Confirm**.
