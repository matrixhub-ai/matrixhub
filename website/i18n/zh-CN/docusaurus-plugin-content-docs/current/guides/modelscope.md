---
title: 在 MatrixHub 中使用 ModelScope
sidebar_position: 5
---

# 在 MatrixHub 中使用 ModelScope

MatrixHub 支持将 ModelScope 配置为上游模型源，也支持使用官方 ModelScope SDK 和 CLI 下载 MatrixHub 中的模型。

## 配置上游仓库

在 **仓库管理 → 创建目标仓库** 中选择 **ModelScope（魔搭）**，根地址填写 `https://www.modelscope.cn`。公开模型无需凭据。访问私有模型时，填写魔搭用户名和 SDK 访问令牌，并测试连接。访问魔搭公共服务时保持证书验证开启。

在模型拉取规则或代理项目中选择该仓库。例如，本地项目 `models` 可以代理上游组织 `Qwen`，客户端向 MatrixHub 请求的模型路径为 `models/Qwen3-0.6B`。本次 ModelScope 拉取规则支持模型，数据集接入待后续实现。

## 下载模型

以下命令使用本地项目名和模型名。请将端点替换为实际 MatrixHub 服务的根地址。

```bash
pip install modelscope modelscope-hub
export MODELSCOPE_ENDPOINT="https://matrixhub.example.com"
ms-hub download models/Qwen3-0.6B --local-dir ./Qwen3-0.6B
```

访问 MatrixHub 私有项目时，在个人设置中创建 MatrixHub 访问令牌，设置到 `MODELSCOPE_API_TOKEN`。上游魔搭令牌由服务端的仓库配置管理。

```bash
export MODELSCOPE_API_TOKEN="<your-matrixhub-token>"
```

Python SDK 的整仓和单文件下载共用相同端点及令牌。

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

SDK 的 `revision` 参数和 CLI 的 `--revision` 参数支持分支、标签和提交哈希。仓库中没有实际的 `master` 时，MatrixHub 的 ModelScope 接口将它解析为默认分支。需要固定快照时，请指定提交哈希。

冷缓存代理会先获取 Git 元数据并返回文件列表。完整文件 GET 校验通过后写入 LFS 缓存。Range 请求按字节范围转发，局部响应不会被写成完整缓存文件。

## 使用 vLLM

选择 **使用此模型 → vLLM → ModelScope**，页面会根据当前实例地址和模型分支生成命令。

```bash
pip install vllm modelscope modelscope-hub
export VLLM_USE_MODELSCOPE=True
export MODELSCOPE_ENDPOINT="https://matrixhub.example.com"
vllm serve models/Qwen3-0.6B --revision main
```

请以模型页面显示的实际分支为准。部分 vLLM 版本在启用 ModelScope 下载后仍会额外探测 HF 元数据。MatrixHub 的使用引导配置的是 ModelScope 下载路径。
