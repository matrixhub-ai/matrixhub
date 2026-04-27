---
sidebar_position: 2
---

# Command Line Upload and Download

## Prerequisites

- A valid MatrixHub account.
- Joined the target project with read/write permissions for the model repository.
- Hugging Face CLI installed locally (`hf` command available).
- Network access to the MatrixHub service endpoint.

## Uploading Models

1. Log in to the platform, go to **Project Management**, and select the target project.
1. Open the **Model Repository** tab and click **Create Model**.

    ![Create Model](./images/create-model.jpg)

1. Fill in the model name, confirm creation, and enter the model details page.

    ![After Create Model](./images/after-create-model.jpg)

1. Configure the service endpoint in your local terminal.

    ```bash
    export HF_ENDPOINT="https://<your-matrixhub-endpoint>"
    ```

1. Use `hf upload` to upload the local model directory.

    ```bash
    hf upload <project-name>/<model-name> ./<local-model-dir>
    ```

1. Return to the model details page and refresh to confirm the uploaded files appear in the list.

    ![After Upload](./images/after-upload.jpg)

:::note

- If the model name is already taken, please choose a different name and try again.
- Uploading large models for the first time may take a while; please wait for the command to complete.

:::

## Downloading Models

1. Enter the target model details page and click **Download Model**.
1. Copy the download command from the popup and execute it in your local terminal.

    ```bash
    export HF_ENDPOINT="https://<your-matrixhub-endpoint>"
    hf download <project-name>/<model-name>
    ```

1. After the command completes, the terminal will output the download directory path.
1. Open the local download directory and verify that the model files are complete and usable.

## Proxy Project Upload and Download

1. Create a proxy project first (e.g., `proxy-demo`).
1. After creating a model in the proxy project, you can perform uploads and downloads via the command line against the proxy site.

    ```bash
    export HF_ENDPOINT="https://<your-matrixhub-endpoint>"
    ```

1. Download a model from the proxy project (example).

    ```bash
    hf download proxy-project/Qwen3-ASR-0.6B
    ```

1. Download a model from a proxy project mapped to a Hugging Face organization (e.g., `prajjwal1`) (example).

    ```bash
    hf download prajjwal1/bert-tiny
    ```

1. Upload a local model to the proxy project (example).

    ```bash
    hf upload proxy-project/tiny-model ./<local-model-dir>
    ```

:::note

- Once the proxy project is created, you can access proxy site model repositories using `hf download` and `hf upload`.
- `hf upload` requires both the remote repository and the local path: `hf upload <project>/<model> <local_path>`.

:::

## Model Files

    ![Model Files Interface](./images/model-file.png)

### Downloading Single Model Files

1. Enter the model details page and switch to the **Model Files** tab.
1. Click **Download** on the target file's row.
1. After the browser completes the download, open the file to verify the content.

### File Search and Browsing

1. Use the search box in the **Model Files** page to enter keywords (e.g., `.git`, `tokenizer`).
1. Observe the filtered results to ensure the returned files match expectations.
1. If there are many files, click **Load More** to view the full list.

### Branch/Version Switching

1. Enter the **Model Files** page of the model details.
1. Select the target branch (e.g., `main`, `test`) in the branch selector.
1. Verify that the file list after switching matches the content of that branch.

