---
slug: /sglang-matrixhub-cache-acceleration
title: Speeding up SGLang model startup with MatrixHub cache
description: A Qwen3-0.6B test comparing SGLang startup time when loading model files from MatrixHub cache versus directly from Hugging Face.
---

When starting an inference service locally or inside a private network, model download is often the slowest and least predictable step.

SGLang, Transformers, vLLM, and many other tools fetch model files through the Hugging Face Hub protocol. If every service pulls directly from public Hugging Face, startup time depends on public network bandwidth, rate limits, and remote availability.

In this test, we use `Qwen/Qwen3-0.6B` to compare two startup paths:

- SGLang pulls model files through MatrixHub's Hugging Face-compatible endpoint.
- SGLang pulls model files directly from Hugging Face.

{/* truncate */}

## Prepare the model cache in MatrixHub

First, create a Hugging Face Registry in MatrixHub and use it from a Proxy Project.

The Proxy Project is a Hugging Face-compatible proxy entrypoint. It is still empty right after creation. MatrixHub does not automatically sync all upstream models when the project is created.

![Configure a Hugging Face Registry](/img/blog/sglang-matrixhub/sglang8.png)

The cache is created when a client first requests model files. You can pre-warm it with `hf download`:

```bash
HF_ENDPOINT=http://127.0.0.1:3002

hf download Qwen/Qwen3-0.6B
```

The `hf` CLI sends the request to MatrixHub's Hugging Face-compatible API. If the files are not cached yet, MatrixHub pulls them from upstream Hugging Face and stores them locally. Later, SGLang, vLLM, or other clients can hit the MatrixHub cache directly without going back to Hugging Face.

After the cache is populated, the model detail page shows the cached files.

![Cached model files in MatrixHub](/img/blog/sglang-matrixhub/sglang2.png)

## Experiment 1: Start SGLang through MatrixHub

Before each run, remove the local Hugging Face cache to avoid measuring local cache hits:

```bash
rm -rf ~/.cache/huggingface/hub/models--Qwen--Qwen3-0.6B
```

Then start SGLang with `HF_ENDPOINT` pointing to MatrixHub:

```bash
SGLANG_USE_MLX=1 HF_ENDPOINT=http://127.0.0.1:3002 python -m sglang.launch_server \
  --model-path Qwen/Qwen3-0.6B \
  --host 0.0.0.0 \
  --port 30000 \
  --disable-cuda-graph
```

The key part is:

```bash
HF_ENDPOINT=http://127.0.0.1:3002
```

Here, `127.0.0.1:3002` is the Hugging Face-compatible endpoint exposed by MatrixHub.

![Start SGLang with the MatrixHub endpoint](/img/blog/sglang-matrixhub/sglang3.png)

The startup log confirms the endpoint used for model downloads:

```text
Hugging Face endpoint for model downloads: http://127.0.0.1:3002
```

The model files are then fetched and the server becomes ready:

```text
Fetching 7 files: 100%
Download complete: 100% 1.50G/1.50G
MLX model loaded in 3.39s
The server is fired up and ready to roll!
```

![SGLang started after loading from MatrixHub cache](/img/blog/sglang-matrixhub/sglang4.png)

From the screenshot:

| Stage | Time |
|---|---:|
| Command started | 21:23:16 |
| SGLang ready | 21:23:51 |
| Total | About 35 seconds |

## Verify the inference service

After the server is ready, call the OpenAI-compatible API:

```bash
curl http://127.0.0.1:30000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen/Qwen3-0.6B",
    "messages": [
      {"role": "user", "content": "你好，简单介绍一下你自己"}
    ],
    "max_tokens": 128,
    "temperature": 0.6
  }'
```

The model returns a normal response, which confirms that the model was loaded successfully and the inference service is usable.

![Call the SGLang OpenAI-compatible API](/img/blog/sglang-matrixhub/sglang5.png)

## Experiment 2: Start SGLang directly from Hugging Face

Next, run the same model with the same SGLang arguments, but point the endpoint back to Hugging Face:

```bash
SGLANG_USE_MLX=1 HF_ENDPOINT=https://huggingface.co python -m sglang.launch_server \
  --model-path Qwen/Qwen3-0.6B \
  --host 0.0.0.0 \
  --port 30000 \
  --disable-cuda-graph
```

![Pull model files directly from Hugging Face](/img/blog/sglang-matrixhub/sglang6.png)

This path also starts successfully, but it takes longer:

```text
MLX model loaded in 72.97s
The server is fired up and ready to roll!
```

![SGLang started after pulling from Hugging Face](/img/blog/sglang-matrixhub/sglang7.png)

From the screenshot:

| Stage | Time |
|---|---:|
| Command started | 21:26:31 |
| SGLang ready | 21:28:36 |
| Total | About 125 seconds |

## Results

Both runs cleared the local Hugging Face cache first. The model and SGLang arguments stayed the same. The only major difference was `HF_ENDPOINT`.

| Source | Endpoint | Command started | Server ready | Total |
|---|---|---:|---:|---:|
| MatrixHub cache | `http://127.0.0.1:3002` | 21:23:16 | 21:23:51 | About 35 seconds |
| Hugging Face | `https://huggingface.co` | 21:26:31 | 21:28:36 | About 125 seconds |

In this test, the MatrixHub path reduced startup time from about 125 seconds to about 35 seconds.

`Qwen3-0.6B` is a small model. With larger models, or with multiple machines repeatedly pulling the same models, a shared cache layer becomes much more valuable.

## Why MatrixHub helps

MatrixHub acts as a Hugging Face-compatible model cache layer:

```text
SGLang / vLLM / Transformers
        ↓
HF_ENDPOINT
        ↓
MatrixHub Proxy Project
        ↓
MatrixHub cache or upstream Hugging Face
```

The first request fills the cache. Later requests for the same model can be served by MatrixHub directly.

The client still uses the original Hugging Face repo name:

```text
Qwen/Qwen3-0.6B
```

And the integration only requires setting the endpoint:

```bash
export HF_ENDPOINT=http://127.0.0.1:3002
```

This gives several practical benefits:

- Fewer repeated downloads for the same model.
- More predictable startup time.
- A single model entrypoint for SGLang, vLLM, Transformers, and other Hugging Face-compatible clients.
- Better fit for private networks, shared development environments, and inference clusters.
