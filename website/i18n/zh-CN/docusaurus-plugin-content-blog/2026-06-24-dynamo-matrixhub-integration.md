---
slug: /dynamo-matrixhub-integration
title: Dynamo 与 MatrixHub 集成实验
description: 在带 GPU 的 Kubernetes 集群上让 Dynamo 通过内网 MatrixHub 拉取模型权重，并与外网 Hugging Face 对比首次下载耗时。
---

我们做了两组实验，验证内网 MatrixHub 对 Dynamo 推理服务首次拉取模型权重的加速效果。

- **实验一**：在带 GPU 的 Kubernetes 集群上部署 Dynamo，并使用内网 MatrixHub 拉取模型权重。部署后会得到一个兼容 OpenAI 接口的推理服务，可以对 `qwen3-0.6b` 模型发对话请求。
- **实验二**：用同样的方式再实验一次，改成从外网 Hugging Face 拉取模型权重，对两次实验的首次下载模型时间作对比。

{/* truncate */}

## 实验一

### 环境背景

- **Dynamo 平台已安装** —— 集群里已部署 Dynamo operator，能识别第二步的部署文件并自动拉起服务。
- **GPU 节点** —— 一块 NVIDIA A800 80GB，可切成 10 份 vGPU。GPU 节点装了 HAMi（GPU 虚拟化组件），它把一整块物理 GPU 切成多份，让多个服务共享同一张卡。
- **内网模型 Hub（MatrixHub）** —— 内网部署了模型权重仓库 MatrixHub（`matrixhub.internal:30001`），模型权重从这里下载，不走公网；且提前用 `hf download` 命令在 MatrixHub 上缓存好 `chenyang-qwen/qwen3-0.6b` 模型。
- **网络可达** —— 集群 `<cluster-node>` 能拉取容器镜像（nvcr.io），也能访问内网模型权重源 MatrixHub（`matrixhub.internal:30001`）。

需先确认上述条件，尤其是 **Dynamo 平台**和 **vGPU** 是否已就绪。

### 开始前你需要

- 集群的 kubeconfig 文件
- 一台装了 `kubectl` 的电脑

### 第一步：连接集群

打开终端，设置 kubeconfig（每次新开终端都要做一次）：

```bash
# 把下面替换成你的 kubeconfig 文件实际路径
export KUBECONFIG=<你的-kubeconfig-文件路径>
```

验证能连上：

```bash
kubectl get nodes
```

能看到节点列表就 OK。

### 第二步：准备部署文件

新建一个文件 `dgd-vllm-vgpu.yaml`，内容如下：

```yaml
apiVersion: nvidia.com/v1alpha1
kind: DynamoGraphDeployment
metadata:
  name: vllm-qwen-vgpu
  namespace: dynamo-system
spec:
  services:
    Frontend:
      componentType: frontend
      replicas: 1
      resources:
        requests:
          cpu: "2"
          memory: "4Gi"
        limits:
          cpu: "2"
          memory: "4Gi"
      extraPodSpec:
        mainContainer:
          image: nvcr.io/nvidia/ai-dynamo/vllm-runtime:1.1.1
          workingDir: /workspace
          env:
            - { name: HF_ENDPOINT, value: "http://matrixhub.internal:30001" }
          command: ["python3", "-m", "dynamo.frontend"]
          args: ["--http-port", "8000"]

    decode:
      componentType: worker
      subComponentType: decode
      replicas: 1
      resources:
        requests:
          cpu: "4"
          memory: "16Gi"
          custom:
            nvidia.com/vgpu: "1"          # 1 个 vGPU 切片
            nvidia.com/gpumem: "10000"    # 显存上限, MB (~10GB)
            nvidia.com/gpucores: "30"     # 算力上限, 0-100
        limits:
          cpu: "4"
          memory: "16Gi"
          custom:
            nvidia.com/vgpu: "1"
            nvidia.com/gpumem: "10000"
            nvidia.com/gpucores: "30"
      extraPodSpec:
        mainContainer:
          image: nvcr.io/nvidia/ai-dynamo/vllm-runtime:1.1.1
          workingDir: /workspace
          env:
            - { name: HF_ENDPOINT, value: "http://matrixhub.internal:30001" }
          command: ["python3", "-m", "dynamo.vllm"]
          args:
            - --model
            - chenyang-qwen/qwen3-0.6b
            - --served-model-name
            - chenyang-qwen/qwen3-0.6b
            - --tensor-parallel-size
            - "1"
            - --gpu-memory-utilization
            - "0.85"
            - --max-model-len
            - "8192"
            - --no-enable-log-requests
```

注意：环境变量 `HF_ENDPOINT` 指定了内网 MatrixHub 地址 `http://matrixhub.internal:30001`。

如果以后要改东西，常用的几处：

| 想改什么 | 改哪里 |
|---|---|
| 换模型 | 两处 `chenyang-qwen/qwen3-0.6b` 都换成新模型名 |
| 加显存 | 两处 `nvidia.com/gpumem: "10000"` 改大（单位 MB） |
| 加算力 | 两处 `nvidia.com/gpucores: "30"` 改大（最大 100） |

### 第三步：部署

```bash
kubectl apply -f dgd-vllm-vgpu.yaml
```

### 第四步：等它起来

```bash
kubectl -n dynamo-system get pods -l nvidia.com/dynamo-graph-deployment-name=vllm-qwen-vgpu -w
```

等到两个 Pod 都显示 `1/1 Running`（第一次部署要拉镜像，可能等几分钟）。

![两个 Pod Running](/img/blog/dynamo-matrixhub/pods-running-1.png)

![两个 Pod Running](/img/blog/dynamo-matrixhub/pods-running-2.png)

**看服务日志**：

```bash
kubectl -n dynamo-system logs <Pod名> --tail=50
```

由于内网部署了 MatrixHub 且已缓存了模型（`HF_ENDPOINT` 指向 MatrixHub），模型下载时间约为 10 秒：

![模型下载约 10 秒](/img/blog/dynamo-matrixhub/download-10s.png)

### 第五步：测试服务

先拿到 frontend 的 Pod 名字：

```bash
kubectl -n dynamo-system get pods | grep frontend
```

用这个名字测试（把 `<前端Pod名>` 换成上面查到的）：

```bash
kubectl -n dynamo-system exec <前端Pod名> -- curl -s http://localhost:8000/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"chenyang-qwen/qwen3-0.6b","messages":[{"role":"user","content":"用一句话介绍自己"}],"max_tokens":64}'
```

能看到模型返回一段中文回复，就说明部署成功了。

![对话测试返回](/img/blog/dynamo-matrixhub/chat-test.png)

## 实验二

其它步骤和实验一一样，只要把部署 YAML 里的环境变量 `HF_ENDPOINT` 去掉，就会默认从外网 Hugging Face 拉取模型权重。

没有部署 MatrixHub（从外网 Hugging Face 下模型）时查看 log，模型下载时间约为 6 分钟：

![Hugging Face 下载约 6 分钟](/img/blog/dynamo-matrixhub/hf-download-6min.png)

## 实验数据对比（首次下载模型时间对比）

针对「内网部署了 MatrixHub 且已缓存了模型」和「没有部署 MatrixHub」两种情况，各环节实测耗时参考（`qwen3-0.6b` 模型），**首次下载模型时间对比见下表**：

| 环节 | 实验一（内网 MatrixHub 且已缓存模型） | 实验二（没有部署 MatrixHub） |
|---|---|---|
| 拉容器镜像（10GB+） | 秒级（节点已缓存） | 秒级（节点已缓存） |
| 下载模型权重 | **约 10 秒（从内网 MatrixHub 缓存下）** | **约 6 分钟（从外网 Hugging Face 下）** |
| vLLM 引擎启动 + 模型加载 | 1～2 分钟 | 1～2 分钟 |

## 结论

内网 MatrixHub 对 Dynamo 首次下载模型权重起到了很大的加速作用。
