---
slug: /llmd-qwen3-32b-matrixhub-cache
title: 使用 MatrixHub 为 llm-d 加速模型分发
description: 在 4 卡 llm-d 部署上实测 Qwen3-32B，对比 vLLM 从 Hugging Face 直连和从 MatrixHub 缓存加载权重时的发布耗时。
---

[llm-d](https://llm-d.ai/) 是 Kubernetes 原生的分布式推理栈，它把 vLLM 与 Envoy 路由层、Endpoint Picker 组合起来，在多个 model server 副本之间做前缀缓存感知和负载感知的调度。它解决了大规模推理的编排与路由问题，但在其之下还有一个更基础的问题：模型权重如何送达每一个推理副本。

现代大模型的权重动辄数十至数百 GB。推理服务的每一次扩容、每一次 Pod 重新调度、每一次滚动更新，都意味着一份完整的权重要重新传输到节点。当所有副本都直接从公网模型仓库拉取时，启动耗时便受制于公网带宽、限流与远端可用性；在离线或受管控的网络中，直连公网仓库甚至不可行。对生产推理而言，模型分发往往比模型服务本身更早成为瓶颈——真正拖慢启动的通常不是推理引擎，而是权重下载。

[MatrixHub](https://github.com/matrixhub-ai/matrixhub) 是开源、可私有部署的 AI 模型仓库，正是面向这一层设计的。它提供 Hugging Face 兼容接口：将 `HF_ENDPOINT` 指向 MatrixHub，vLLM、SGLang 等客户端仍按原始仓库名请求模型，而文件由集群内的缓存就近供给。首次请求时 MatrixHub 从上游拉取并落盘，之后对同一模型的请求直接命中缓存，无需再回公网。它同时提供私有模型托管、按项目隔离的权限与审计，以及离线环境下的可控分发。

两者结合形成清晰的分层：llm-d 负责推理控制面——部署、路由、调度与推理 API；MatrixHub 承担其下的模型分发层——模型从哪里来、如何缓存、如何在集群内就近供给。推理编排与模型分发各有其生命周期，将两者解耦后，平台团队既能获得 Hugging Face 式的模型访问体验，又不必让生产推理依赖公网仓库的可用性与带宽。

本文通过一组实测数据量化模型分发层的作用：在 llm-d 上部署同一模型两次，以权重来源为唯一变量——一次直连公网 Hugging Face 镜像，一次经集群内 MatrixHub 缓存——分别测量其到达可用状态所需的时间。

{/* truncate */}

## 实验范围

整个实验共有一套 [llm-d standalone Router](https://llm-d.ai/docs/dev/getting-started/quickstart) 前置基础设施。Router 包含 Envoy、Endpoint Picker（EPP）和通过 llm-d 标签发现 vLLM Pod 的 `InferencePool`。两个场景复用这套基础设施；Router 的安装与就绪等待发生在实验运行之外，不计入模型加载测量。最后的推理验证请求经过 EPP Service。工作负载固定为一个 vLLM Pod、四张 GPU，Tensor Parallel Size 为 4。

| 项目 | 固定值 |
|---|---|
| 模型 | `Qwen/Qwen3-32B` |
| 副本数 | 1 |
| Tensor Parallel Size | 4 |
| 每个 Pod 的 GPU 数 | 4 |
| llm-d 版本 | `7029aac48505752dd51344ce552acc81b0deb774` |
| Router Chart | `llm-d-router-standalone` `v0.9.0` |
| Model Server 镜像 | `m.daocloud.io/docker.io/vllm/vllm-openai:v0.22.0-cu129` |

请求路径为 `客户端 -> standalone Envoy -> EPP -> vLLM Pod`。单副本用于验证 MatrixHub 与完整 llm-d 请求链路的兼容性，不用于衡量 EPP 的调度收益；评估前缀缓存或负载感知路由时，应使用两个或更多 model server 副本。

实验包含两个直接面向使用者的场景：

- **Hugging Face 直连：** model server 使用上游 Hugging Face 地址。
- **MatrixHub 缓存命中：** 相同的部署通过 ConfigMap 获得指向 MatrixHub 的 `HF_ENDPOINT`，且 MatrixHub 中已经完整缓存该模型。

每个场景都使用新命名空间，以保证 Pod 本地缓存为空，同时保持模型、vLLM 参数和 GPU 资源请求一致。

## 实验执行过程

本次实验的全部可执行物料位于matrixhub仓库的 [`examples/llm-d-qwen3-32b`](https://github.com/matrixhub-ai/matrixhub/tree/main/examples/llm-d-qwen3-32b) 目录。克隆仓库后，按目标集群配置 `env.sh`：

```bash
MATRIXHUB_ENDPOINT="http://10.0.0.20:9527"
MATRIXHUB_NAMESPACE="matrixhub"
MATRIXHUB_DEPLOYMENT="matrixhub"
LLMD_NAMESPACE="matrixhub-llmd-qwen3-32b"
MODEL_NODE_SELECTOR="gpu-10-125-1-4"
LLMD_ROUTER_CHART="oci://ghcr.io/llm-d/charts/llm-d-router-standalone"
LLMD_ROUTER_CHART_VERSION="v0.9.0"
LLMD_ROUTER_RELEASE="optimized-baseline"
```

`MATRIXHUB_ENDPOINT` 为 llm-d model server Pod 可访问的 MatrixHub 地址。`MODEL_NODE_SELECTOR` 固定为指定节点，以确保两个场景运行于完全相同的硬件，排除调度差异带来的干扰。

在使用 GPU 前，先完成配置校验与清单渲染：

```bash
make check
make render
```

随后一次性安装共享的 llm-d Router：

```bash
make prepare-router
```

该步骤创建命名空间，安装 Envoy、EPP 与 `InferencePool`，并等待 Router 就绪，不计入任一场景的测量。

每个场景均先移除上一次的 model server 工作负载，再部署带空 `emptyDir` 缓存的新 Pod，并将时间戳、日志、Helm 状态与渲染后的清单写入 `artifacts/` 下按时间命名的目录。两个场景之间未执行 `make cleanup`，以保留共享 Router。每个场景重复执行三次；下面的截图取自其中一次代表性运行。

### 场景一：Hugging Face 直连

```bash
make run-direct
```

脚本等待 Deployment 完成滚动发布，随后经 EPP Service 发起一次 chat-completions 请求：

![make run-direct 完成并通过推理验证](./images/direct-make.png)

vLLM 日志确认本次运行的下载权重下载耗时：

![Hugging Face 直连场景的 vLLM 权重下载日志](./images/direct-vllm.png)

该次运行的 artifacts 目录下生成 `metrics.md`：

![Hugging Face 直连场景的 metrics.md](./images/direct-metrics.png)

### 场景二：MatrixHub 缓存命中

在本场景测量前，先将完整的 `Qwen/Qwen3-32B` 仓库缓存至 MatrixHub，并确认缓存完整。填充缓存的首次请求不构成缓存命中，未计入结果。

```bash
make run-matrixhub
```

本次运行同样经 EPP Service 完成：

![make run-matrixhub 完成并通过推理验证](./images/matrixhub-make.png)

vLLM 日志确认本次运行的下载权重下载耗时：

![MatrixHub 缓存命中场景的 vLLM 权重下载日志](./images/matrixhub-vllm.png)

对应的 `metrics.md`：

![MatrixHub 缓存命中场景的 metrics.md](./images/matrixhub-metrics.png)

## 测量指标

运行脚本记录以下内容：

- 从提交部署清单到 Pod 创建的时间；
- Pod Ready 时间；
- Deployment 完成滚动发布的总时间；
- 经 EPP Service 的推理请求验证结果；
- Router 的 Helm 状态；
- model server 日志、Pod 清单、事件，以及能够获取时的 MatrixHub 日志。

其中权重下载耗时来自 vLLM 自身的日志：

```text
INFO [weight_utils.py:603] Time spent downloading weights for Qwen/Qwen3-32B: 515.641089 seconds
INFO [weight_utils.py:922] Filesystem type for checkpoints: OVERLAY. Checkpoint size: 61.02 GiB.
```

下载速率由模型总大小除以该耗时得到，`Qwen/Qwen3-32B` 的 safetensors 权重约 62 GB。该指标只反映权重传输阶段，不包含 vLLM 加载权重到显存、编译 CUDA graph 等模型初始化开销。

## 实验结果

下面是一次实际执行的测量结果，每个场景重复三次，每次都使用空的 Pod 本地缓存。它来自单一集群，不能作为其他环境的性能承诺。

| 项目 | 值 |
|---|---|
| GPU | 4 × NVIDIA 5090 GPU，单节点 |
| 节点内存 | 978 GiB 可用 |
| 模型 | `Qwen/Qwen3-32B`，safetensors 约 62 GB |
| 容器文件系统 | OVERLAY（Pod 本地 `emptyDir` 缓存，每次运行前清空） |
| 直连 endpoint | `https://hf.m.daocloud.io` |
| MatrixHub endpoint | 通过 ConfigMap 注入 `HF_ENDPOINT` |

下表为每个场景三次运行的平均值。

| 来源 | Pod 创建 (s) | 权重下载 (s) | 下载速率 (MB/s) | Pod Ready (s) | 滚动发布完成 (s) | 推理验证 |
|---|---:|---:|---:|---:|---:|---|
| Hugging Face 直连 | 1 | 521.4 | 118.9 | 722 | 751 | 通过 |
| MatrixHub 缓存命中 | 1 | 144.6 | 428.8 | 373 | 397 | 通过 |

对比：

| 指标 | Hugging Face 直连 | MatrixHub 缓存命中 | 差异 |
|---|---:|---:|---|
| 权重下载耗时 | 521.4 s | 144.6 s | 减少 376.8 s（3.6× 加速） |
| 下载速率 | 118.9 MB/s | 428.8 MB/s | 提升 3.6× |
| Pod Ready | 722 s | 373 s | 减少 349 s（48.3%） |
| 滚动发布总时长 | 751 s | 397 s | 减少 354 s（47.1%） |

## 结果解读

两个场景的差异集中在权重传输阶段：下载耗时从 521.4 s 降到 144.6 s，减少 376.8 s；而端到端的滚动发布时间减少 354 s，二者基本吻合。这说明本次实验中 MatrixHub 带来的收益来自就近缓存带宽，而不是 vLLM 侧的加载或初始化逻辑变化。

下载之外的时间在两个场景中大致相同，对应 vLLM 将权重载入四张 GPU、按 Tensor Parallel Size 4 切分并完成初始化的固定开销。这部分不随模型来源改变，因此随着模型体积增大、下载占比上升，MatrixHub 的相对收益会更明显；反之在小模型上收益有限。

本结论只在上述环境成立。直连场景的下载速率取决于公网带宽和上游镜像站状态，MatrixHub 场景取决于缓存与节点之间的网络和存储介质，两者在其他环境中都可能显著不同。

完整的实验物料——kustomize 清单、Router values、运行脚本、前提条件与结果模板——均位于 [`examples/llm-d-qwen3-32b`](https://github.com/matrixhub-ai/matrixhub/tree/main/examples/llm-d-qwen3-32b)。
