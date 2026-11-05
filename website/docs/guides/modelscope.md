---
title: Use ModelScope with MatrixHub
sidebar_position: 5
---

# Use ModelScope with MatrixHub

MatrixHub supports ModelScope as an upstream model source and accepts model
downloads from the official ModelScope SDK and CLI.

## Connect an upstream

In **Registry Management → Create Registry**, select **ModelScope** and enter
`https://www.modelscope.cn` as the base URL. Public models need no credentials.
For private models, enter your upstream username and ModelScope SDK access token
and test the connection. Keep certificate verification enabled for the public
service.

Use this registry in a model pull rule or a proxy project. For example, a local
project named `models` can proxy the upstream namespace `Qwen`. Clients then
request `models/Qwen3-0.6B` from MatrixHub. ModelScope pull rules currently
support models. Dataset integration is outside this feature.

## Download a model

The following examples use the local project/model path. Replace the endpoint
with the base URL of your MatrixHub installation.

```bash
pip install modelscope modelscope-hub
export MODELSCOPE_ENDPOINT="https://matrixhub.example.com"
ms-hub download models/Qwen3-0.6B --local-dir ./Qwen3-0.6B
```

For a private MatrixHub project, set `MODELSCOPE_API_TOKEN` to a MatrixHub access
token created in your profile. The upstream token is configured on the registry.

```bash
export MODELSCOPE_API_TOKEN="<your-matrixhub-token>"
```

Python snapshot and single-file downloads use the same endpoint and token.

```python
import os
from modelscope import snapshot_download
from modelscope.hub.file_download import model_file_download

endpoint = 'https://matrixhub.example.com'
token = os.getenv('MODELSCOPE_API_TOKEN')

model_dir = snapshot_download(
    'models/Qwen3-0.6B', endpoint=endpoint, token=token,
)
config = model_file_download(
    'models/Qwen3-0.6B', 'config.json', endpoint=endpoint, token=token,
)
```

Branches, tags and commit hashes are accepted through the SDK's `revision`
argument and the CLI's `--revision` option. When `master` does not exist,
MatrixHub treats it as an alias for the default branch in these ModelScope
routes. Pin a commit hash for a stable snapshot.

Cold proxy reads fetch Git metadata before returning the file list. They do not
wait for the complete model to download. Full file GETs are verified before
being published to the LFS cache. Range responses are forwarded as byte ranges
and are not installed as complete cached files.

## Start vLLM

Choose **Use this model → vLLM → ModelScope** for instructions generated with
this instance's endpoint and the model's revision.

```bash
pip install vllm modelscope modelscope-hub
export VLLM_USE_MODELSCOPE=True
export MODELSCOPE_ENDPOINT="https://matrixhub.example.com"
vllm serve models/Qwen3-0.6B --revision main
```

Use the actual revision shown on the model page. vLLM releases can contain
additional HF metadata probes even when ModelScope downloads are enabled.
The MatrixHub instructions configure the ModelScope download path.
