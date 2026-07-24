---
sidebar_position: 3
---

# Cache models through a proxy project

## Goal

This guide introduces on-demand proxy caching. On the first client request, MatrixHub downloads and caches the model from Hugging Face. Later requests load the model directly from the MatrixHub cache.

## Architecture

<img
  src={require('./images/mirror-from-huggingface-architecture.png').default}
  alt="MatrixHub caches Hugging Face models on demand through a proxy project"
  width="370"
/>

## Prerequisites

- [MatrixHub is deployed](../installation/index.md). This guide uses `http://192.0.2.10:30001` as the example MatrixHub address.
- Keep the client and MatrixHub on the same internal network when possible.

## Ensure that a target registry and proxy project exist

Open the MatrixHub address in a browser and sign in.

- Username: `admin`
- Password: `changeme`

<img
  src={require('./images/mirror-matrixhub-login.png').default}
  alt="MatrixHub sign-in page"
  width="360"
/>

After signing in, open **Personal Center** and change the password as soon as possible.

![Open Personal Center from the user menu](./images/mirror-profile-menu.png)

### Create a target registry

Select the username in the upper-right corner, then select **Platform Settings**.

![Open Platform Settings](./images/mirror-platform-settings.png)

Select **Registry Management**, then select **Create Target Registry**.

![Create a target registry from Registry Management](./images/mirror-registry-management.png)

Select the Hugging Face provider. Enter `hf-mirror` (or `huggingface.co`) as the registry name and `https://hf-mirror.com` (or `https://huggingface.co`) as the target URL. Enable **Verify Remote Certificate**, then select **Test Connection**.

![Configure and test the Hugging Face target registry](./images/mirror-create-target-registry.png)

Select **Confirm**. The target registry is created.

### Create a proxy project

Select **Project Management**, then select **Create Project**.

![Create a project from Project Management](./images/mirror-project-management.png)

Enter `Qwen` as the project name, select **Public**, enable the proxy, select the `hf-mirror` target registry created earlier, enter `Qwen` as the proxy organization, then select **Confirm**.

<img
  src={require('./images/mirror-create-proxy-project.png').default}
  alt="Configure the Qwen proxy project"
  width="304"
/>

Select **Qwen** in the project list to open the project.

![Qwen proxy project in the project list](./images/mirror-proxy-project-created.png)

The project does not contain a model yet.

![Empty Qwen proxy project](./images/mirror-proxy-project-empty.png)

## Download a model through MatrixHub on the internal network

### Install the Hugging Face hf CLI

See the [Hugging Face CLI guide](https://huggingface.co/docs/huggingface_hub/guides/cli) for installation information.

```bash
curl -LsSf https://hf.co/cli/install.sh | bash
```

### Point HF_ENDPOINT to MatrixHub

```shell
export HF_ENDPOINT="http://192.0.2.10:30001"
```

### Download Qwen3-0.6B for the first time

```shell
time hf download Qwen/Qwen3-0.6B
```

The first download took a long time because upstream bandwidth was limited: 119 minutes.

![Initial model download through MatrixHub](./images/mirror-initial-download.png)

View the downloaded model files and their size.

```shell
ls ~/.cache/huggingface/hub/models--Qwen--Qwen3-0.6B/
du -sh ~/.cache/huggingface/hub/models--Qwen--Qwen3-0.6B/
```

![Qwen3-0.6B in the local Hugging Face cache](./images/mirror-local-cache.png)

The model detail page now shows the model weights and other files.

![Qwen3-0.6B files in MatrixHub](./images/mirror-model-files.png)

### Download Qwen3-0.6B again

Delete the locally downloaded model files, then download the model from MatrixHub again.

```shell
rm -rf ~/.cache/huggingface/
export HF_ENDPOINT="http://192.0.2.10:30001"
time hf download Qwen/Qwen3-0.6B
```

![Model download served from the MatrixHub cache](./images/mirror-cached-download.png)

The download took approximately 15 seconds, showing that the model was cached in MatrixHub.

## Conclusion

The first on-demand download depends on upstream network speed. Later downloads come directly from the MatrixHub cache and are much faster.
