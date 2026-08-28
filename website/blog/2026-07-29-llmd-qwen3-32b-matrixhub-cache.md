---
slug: /llmd-qwen3-32b-matrixhub-cache
title: Accelerating model distribution for llm-d with MatrixHub
description: A Qwen3-32B experiment on a 4-GPU llm-d deployment, comparing rollout time when vLLM loads weights directly from Hugging Face versus from a populated MatrixHub cache.
---

[llm-d](https://llm-d.ai/) is a Kubernetes-native distributed inference stack. It pairs vLLM with an Envoy routing layer and an Endpoint Picker that makes prefix-cache-aware and load-aware scheduling decisions across model server replicas. It solves orchestration and routing for inference at scale — but underneath it sits a more basic problem: how the model weights reach each serving replica in the first place.

Modern LLM weights routinely run to tens or hundreds of gigabytes. Every scale-out, every rescheduled Pod, and every rollout re-transfers a full copy of those weights to the node. When every replica pulls directly from a public model hub, startup time is bounded by public bandwidth, rate limits, and remote availability; in air-gapped or regulated networks, pulling from a public hub may not be possible at all. For production inference, model distribution tends to become a bottleneck earlier than model serving — the slow part of startup is usually not the inference engine but the weight download.

[MatrixHub](https://github.com/matrixhub-ai/matrixhub) is an open-source, self-hosted AI model registry built for exactly this layer. It exposes a Hugging Face-compatible API: point `HF_ENDPOINT` at MatrixHub and clients such as vLLM and SGLang keep requesting models under their original repo names, while the files are served from a cache inside the cluster. The first request pulls from upstream and stores the files; subsequent requests for the same model hit the cache directly, without going back to the public network. MatrixHub also provides private model hosting, project-scoped access control and audit, and controlled distribution into offline environments.

Together they form a clean split. llm-d owns the inference control plane — deployment, routing, scheduling, and inference APIs. MatrixHub owns the model distribution layer beneath it — where models come from, how they are cached, and how they are served close to the workloads. Serving orchestration and artifact distribution have separate lifecycles; decoupling them lets platform teams keep a Hugging Face-style model experience without making production inference depend on public-hub availability or bandwidth.

This article quantifies what that distribution layer is worth. The same model was deployed on llm-d twice, with the weight source as the only variable — once directly from a public Hugging Face mirror, once through an in-cluster MatrixHub cache — and the time to reach a ready state was measured in each case.

{/* truncate */}

## Experiment scope

The experiment has one shared [llm-d Router standalone](https://llm-d.ai/docs/dev/getting-started/quickstart) prerequisite. The Router provides Envoy, the Endpoint Picker (EPP), and an `InferencePool` that discovers the llm-d-labeled vLLM Pod. Both scenarios reuse this infrastructure; Router installation and readiness occur outside the measured runs. The final inference verification traverses the EPP Service. The workload is fixed at one vLLM Pod, four GPUs, and tensor parallel size 4.

| Item | Fixed value |
|---|---|
| Model | `Qwen/Qwen3-32B` |
| Replicas | 1 |
| Tensor parallel size | 4 |
| GPUs per Pod | 4 |
| llm-d revision | `7029aac48505752dd51344ce552acc81b0deb774` |
| Router chart | `llm-d-router-standalone` `v0.9.0` |
| Model server image | `m.daocloud.io/docker.io/vllm/vllm-openai:v0.22.0-cu129` |

The request path is `client -> standalone Envoy -> EPP -> vLLM Pod`. With one model server replica, this validates MatrixHub with the real llm-d request path but does not measure the scheduling benefit of EPP. Use two or more replicas for a routing-performance experiment.

The comparison has two user-facing scenarios:

- **Hugging Face direct:** the model server uses the upstream Hugging Face endpoint.
- **MatrixHub cache hit:** the same deployment receives `HF_ENDPOINT` from a ConfigMap that points to a MatrixHub instance where the full model is already cached.

Each scenario uses a new namespace. This keeps the Pod-local cache empty without changing the model, vLLM configuration, or GPU request.

## Experiment execution

All executable material for this experiment is maintained in the repository's [`examples/llm-d-qwen3-32b`](https://github.com/matrixhub-ai/matrixhub/tree/main/examples/llm-d-qwen3-32b) directory. After cloning, `env.sh` was configured for the target cluster:

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

`MATRIXHUB_ENDPOINT` is the MatrixHub address reachable from the llm-d model server Pod. `MODEL_NODE_SELECTOR` was pinned to a specific node so that both scenarios ran on identical hardware, eliminating scheduling variance as a confounding factor.

Before any GPU was used, the configuration was validated and both manifests were rendered:

```bash
make check
make render
```

The shared llm-d Router was then installed once:

```bash
make prepare-router
```

This step creates the namespace, installs Envoy, EPP, and the `InferencePool`, and waits for Router readiness. It is excluded from both measurements.

Each scenario removes the previous model-server workload, deploys a new Pod with an empty `emptyDir` cache, and writes timestamps, logs, Helm status, and rendered manifests to a timestamped directory under `artifacts/`. `make cleanup` was not run between the two scenarios, since it removes the shared Router. Each scenario was repeated three times; the screenshots below are from one representative run.

### Scenario 1: Hugging Face direct

```bash
make run-direct
```

The runner waits for the Deployment rollout, then issues a chat-completions request through the EPP Service:

![make run-direct completing with a successful inference verification](./images/direct-make.png)

The vLLM log confirms the download endpoint and the weight-download duration for this run:

![vLLM weight-download log for the Hugging Face direct run](./images/direct-vllm.png)

A `metrics.md` file is written to the run's artifacts directory:

![metrics.md from the Hugging Face direct run](./images/direct-metrics.png)

### Scenario 2: MatrixHub cache hit

Before this measurement, the complete `Qwen/Qwen3-32B` repository was cached in MatrixHub and cache completeness was verified. The request that populates the cache does not constitute a cache hit and was not counted.

```bash
make run-matrixhub
```

The same run completes through the EPP Service:

![make run-matrixhub completing with a successful inference verification](./images/matrixhub-make.png)

The vLLM log shows the weights served through the MatrixHub endpoint:

![vLLM weight-download log for the MatrixHub cache-hit run](./images/matrixhub-vllm.png)

And the corresponding `metrics.md`:

![metrics.md from the MatrixHub cache-hit run](./images/matrixhub-metrics.png)

## Metrics

The runner records:

- time from manifest application to Pod creation;
- time to Pod Ready;
- total Deployment rollout time;
- inference verification through the EPP Service;
- Router Helm status; and
- model server logs, Pod manifests, events, and MatrixHub logs where available.

Weight-download duration comes from vLLM's own log line:

```text
INFO [weight_utils.py:603] Time spent downloading weights for Qwen/Qwen3-32B: 517.641089 seconds
INFO [weight_utils.py:922] Filesystem type for checkpoints: OVERLAY. Checkpoint size: 61.02 GiB.
```

Divide the model size by that duration for an effective download rate; the `Qwen/Qwen3-32B` safetensors weights are about 62 GB. This covers the weight-transfer stage only, not loading weights into GPU memory or compiling CUDA graphs.

Timing alone does not prove a MatrixHub cache hit. Retain log evidence showing which endpoint served the model files.

## Results

The following results come from one actual execution. Each scenario ran three times with an empty Pod-local cache. They come from a single cluster and are not a performance guarantee for other environments.

| Item | Value |
|---|---|
| GPU | 4 × NVIDIA 5090, single node |
| Node memory | 978 GiB available |
| Model | `Qwen/Qwen3-32B`, about 62 GB of safetensors |
| Container filesystem | OVERLAY (Pod-local `emptyDir` cache, emptied before each run) |
| Direct endpoint | `https://hf.m.daocloud.io` |
| MatrixHub endpoint | `HF_ENDPOINT` injected through a ConfigMap |

Values below are averages over three runs per scenario.

| Source | Pod created (s) | Weight download (s) | Download rate (MB/s) | Pod Ready (s) | Rollout completed (s) | Inference |
|---|---:|---:|---:|---:|---:|---|
| Hugging Face direct | 1 | 521.4 | 118.9 | 722 | 751 | passed |
| MatrixHub cache hit | 1 | 144.6 | 428.8 | 373 | 397 | passed |

Comparison:

| Metric | Hugging Face direct | MatrixHub cache hit | Difference |
|---|---:|---:|---|
| Weight download | 521.4 s | 144.6 s | −376.8 s (3.6× faster) |
| Download rate | 118.9 MB/s | 428.8 MB/s | 3.6× |
| Pod Ready | 722 s | 373 s | −349 s (48.3%) |
| Total rollout time | 751 s | 397 s | −354 s (47.1%) |

## Interpretation

The difference between the two scenarios is concentrated in the weight-transfer stage: download time fell from 521.4 s to 144.6 s, a reduction of 376.8 s, while end-to-end rollout time fell by 354 s. Those two figures match closely, which indicates the gain came from cache proximity and bandwidth rather than from any change in how vLLM loads or initializes the model.

Time outside the download was roughly the same in both scenarios. It corresponds to the fixed cost of loading weights into four GPUs, sharding at tensor parallel size 4, and completing initialization. That cost does not vary with the model source, so the relative benefit grows as models get larger and the download accounts for more of the total; on small models the benefit is limited.

These conclusions hold only for the environment above. The direct-path download rate depends on public bandwidth and upstream mirror conditions, and the MatrixHub path depends on the network and storage medium between the cache and the node. Both can differ substantially elsewhere.

The complete experiment — kustomize manifests, Router values, run scripts, prerequisites, and a results template — is available in [`examples/llm-d-qwen3-32b`](https://github.com/matrixhub-ai/matrixhub/tree/main/examples/llm-d-qwen3-32b).
