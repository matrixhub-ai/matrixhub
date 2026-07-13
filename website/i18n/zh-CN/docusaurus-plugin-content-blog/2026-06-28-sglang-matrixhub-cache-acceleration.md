---
slug: /sglang-matrixhub-cache-acceleration
title: 使用 MatrixHub 缓存加速 SGLang 模型启动
description: 通过 Qwen3-0.6B 实测 SGLang 从 MatrixHub 缓存和 Hugging Face 拉取模型时的启动耗时差异。
---

在本地或内网环境启动推理服务时，模型下载经常是最耗时、最不稳定的一步。

SGLang、Transformers、vLLM 等工具通常会通过 Hugging Face Hub 协议获取模型文件；如果每次都直接访问公网 Hugging Face，启动时间就会受到公网带宽、限流和远端可用性的影响。

这篇文章用 `Qwen/Qwen3-0.6B` 做一个简单测试，对比两种启动方式：

- SGLang 通过 MatrixHub 的 Hugging Face 兼容接口拉取模型。
- SGLang 直接从 Hugging Face 拉取模型。

{/* truncate */}

## 准备：在 MatrixHub 中缓存模型

首先在 MatrixHub 中配置 Hugging Face Registry，并创建一个 Proxy Project。

这里的 Proxy Project 可以理解成一个 Hugging Face 兼容的代理入口。创建完成时，它本身还是空的，并不会主动把上游模型全部同步下来。

![配置 Hugging Face Registry](/img/blog/sglang-matrixhub/sglang8.png)

真正的缓存发生在第一次访问模型文件时。可以使用 `hf download` 预热缓存：

```bash
HF_ENDPOINT=http://127.0.0.1:3002

hf download Qwen/Qwen3-0.6B
```

`hf` 会通过 MatrixHub 的 Hugging Face 兼容 API 请求 `Qwen/Qwen3-0.6B`。如果 MatrixHub 还没有这些文件，它会从上游 Hugging Face 拉取并保存；后续 SGLang、vLLM 或其它客户端再访问同一个模型时，就可以直接命中 MatrixHub 缓存，无需再回源。

缓存完成后，可以在 MatrixHub 的模型详情页看到对应的文件列表。

![MatrixHub 中已缓存的模型文件](/img/blog/sglang-matrixhub/sglang2.png)

## 实验一：从 MatrixHub 缓存启动 SGLang

每次测试前先清理本地 Hugging Face cache，避免命中本机缓存影响结果：

```bash
rm -rf ~/.cache/huggingface/hub/models--Qwen--Qwen3-0.6B
```

然后启动 SGLang，并把 `HF_ENDPOINT` 指向 MatrixHub：

```bash
SGLANG_USE_MLX=1 HF_ENDPOINT=http://127.0.0.1:3002 python -m sglang.launch_server \
  --model-path Qwen/Qwen3-0.6B \
  --host 0.0.0.0 \
  --port 30000 \
  --disable-cuda-graph
```

启动命令里最关键的是这一段：

```bash
HF_ENDPOINT=http://127.0.0.1:3002
```

这里的 `127.0.0.1:3002` 是 MatrixHub 暴露的 Hugging Face 兼容访问地址。

![使用 MatrixHub endpoint 启动 SGLang](/img/blog/sglang-matrixhub/sglang3.png)

启动日志中可以看到 SGLang 使用的模型下载 endpoint：

```text
Hugging Face endpoint for model downloads: http://127.0.0.1:3002
```

随后模型文件下载完成，服务启动成功：

```text
Fetching 7 files: 100%
Download complete: 100% 1.50G/1.50G
MLX model loaded in 3.39s
The server is fired up and ready to roll!
```

![从 MatrixHub 缓存下载并启动完成](/img/blog/sglang-matrixhub/sglang4.png)

从截图中的时间看：

| 阶段 | 时间 |
|---|---:|
| 启动命令开始 | 21:23:16 |
| SGLang ready | 21:23:51 |
| 总耗时 | 约 35 秒 |

## 验证推理服务可用

服务 ready 后，可以直接通过 OpenAI-compatible API 发起请求：

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

返回结果里可以看到模型正常生成回复。

![调用 SGLang OpenAI-compatible API](/img/blog/sglang-matrixhub/sglang5.png)

## 实验二：直接从 Hugging Face 启动

接下来用同样的模型、同样的 SGLang 启动参数，只把 endpoint 改回 Hugging Face：

```bash
SGLANG_USE_MLX=1 HF_ENDPOINT=https://huggingface.co python -m sglang.launch_server \
  --model-path Qwen/Qwen3-0.6B \
  --host 0.0.0.0 \
  --port 30000 \
  --disable-cuda-graph
```

![直接从 Hugging Face 拉取模型](/img/blog/sglang-matrixhub/sglang6.png)

最终 Hugging Face 路径也可以启动成功，但耗时明显更长：

```text
MLX model loaded in 72.97s
The server is fired up and ready to roll!
```

![Hugging Face 路径启动完成](/img/blog/sglang-matrixhub/sglang7.png)

从截图中的时间看：

| 阶段 | 时间 |
|---|---:|
| 启动命令开始 | 21:26:31 |
| SGLang ready | 21:28:36 |
| 总耗时 | 约 125 秒 |

## 结果对比

两次实验都清理了本地 Hugging Face cache，模型和启动参数保持一致，主要区别只有 `HF_ENDPOINT`。

| 模型来源 | Endpoint | 启动开始 | 服务 ready | 总耗时 |
|---|---|---:|---:|---:|
| MatrixHub 缓存 | `http://127.0.0.1:3002` | 21:23:16 | 21:23:51 | 约 35 秒 |
| Hugging Face | `https://huggingface.co` | 21:26:31 | 21:28:36 | 约 125 秒 |

在这次测试中，SGLang 从 MatrixHub 缓存启动约 35 秒，从 Hugging Face 启动约 125 秒。

`Qwen3-0.6B` 只是一个小模型。如果换成更大的模型，或者同一团队里有多台机器、多套推理服务反复拉同一批模型，缓存层的收益会更明显。

## MatrixHub 带来的价值

MatrixHub 在这个场景里提供的是一个 Hugging Face 兼容的模型缓存层：

```text
SGLang / vLLM / Transformers
        ↓
HF_ENDPOINT
        ↓
MatrixHub Proxy Project
        ↓
MatrixHub 本地缓存 / 上游 Hugging Face
```

第一次请求模型时，MatrixHub 会回源 Hugging Face 并缓存文件。

后续再次请求同一个模型时，客户端仍然使用原来的 Hugging Face repo 形式：

```text
Qwen/Qwen3-0.6B
```

只需要把 endpoint 指向 MatrixHub：

```bash
export HF_ENDPOINT=http://127.0.0.1:3002
```

这样做有几个直接好处：

- **减少重复下载**：同一个模型被缓存后，后续服务可以直接从 MatrixHub 获取。
- **提升启动稳定性**：减少对公网 Hugging Face 的实时依赖。
- **统一模型入口**：SGLang、vLLM、Transformers 等工具都可以通过 Hugging Face 兼容接口接入。
- **适合内网分发**：多台机器、多套推理服务可以共享同一份模型缓存。
