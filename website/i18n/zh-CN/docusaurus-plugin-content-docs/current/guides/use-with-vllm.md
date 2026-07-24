---
sidebar_position: 2
---

# vLLM 通过 MatrixHub 加载模型

## 目标

vLLM 通过 MatrixHub 加载模型并进行简单推理问答。

## 架构原理

<img
  src={require('./images/vllm-matrixhub-architecture.png').default}
  alt="vLLM 通过 MatrixHub 加载模型"
  width="481"
/>

## 前置条件

- [MatrixHub 已经部署](../installation/index.md)并且[缓存好模型](./mirror-from-huggingface.md)，假设 MatrixHub 地址为 `http://192.0.2.10:30001`
- 有符合模型运行条件的 GPU
- vLLM 运行节点和 MatrixHub 在同一个内网网段

## 部署 vLLM

### 方法一：用 nerdctl 或 Docker 命令启动容器来部署 vLLM

- 安装 GPU 驱动，详见 [Install NVIDIA Container Toolkit for Docker deployment](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)
- 部署 vLLM，详见 [deploy vLLM using Docker](https://docs.vllm.ai/en/latest/deployment/docker/)

以 nerdctl 命令为例部署 vLLM 并进入容器。

```shell
nerdctl stop qwen 2>/dev/null || true
nerdctl rm qwen 2>/dev/null || true
nerdctl run -d \
  --gpus device=0 \
  --shm-size 8G \
  --network host \
  --name qwen \
  --entrypoint sleep \
  docker.m.daocloud.io/vllm/vllm-openai:v0.18.0 \
  infinity

nerdctl exec -it qwen -- sh
```

### 方法二：在 Kubernetes 上部署 vLLM

- 安装 GPU 驱动，详见 [Install NVIDIA GPU Operator for Kubernetes deployment](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/getting-started.html)
- 部署 vLLM，详见 [deploy vLLM using Kubernetes](https://docs.vllm.ai/en/latest/deployment/k8s/)

部署 YAML 示例。

```yaml
kubectl apply -f - <<EOF
kind: Deployment
apiVersion: apps/v1
metadata:
  name: vllm-server
  labels:
    app: vllm
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vllm
  template:
    metadata:
      labels:
        app: vllm
    spec:
      volumes:
        - name: shm
          emptyDir:
            medium: Memory
            sizeLimit: 2Gi
      containers:
        - name: vllm
          image: docker.m.daocloud.io/vllm/vllm-openai:v0.18.0
          command:
            - sleep
          args:
            - infinity
          ports:
            - containerPort: 8000
              protocol: TCP
          resources:
            limits:
              memory: 64G
              nvidia.com/gpu: '1'
            requests:
              memory: 6G
              nvidia.com/gpu: '1'
          volumeMounts:
            - name: shm
              mountPath: /dev/shm
EOF
```

进入容器。

```shell
kubectl exec -it deploy/vllm-server -- sh
```

### 方法三：直接在 Python 环境上部署 vLLM

- 安装 GPU 驱动，详见 [Install NVIDIA GPU drivers for Python deployment](https://docs.nvidia.com/datacenter/tesla/driver-installation-guide/latest/)
- 部署 vLLM，详见 [deploy vLLM using Python](https://docs.vllm.ai/en/latest/getting_started/installation/gpu/#set-up-using-python)

## 启动 vLLM 并通过内网 MatrixHub 加载模型

确保已经进入 vLLM 运行环境，把 `HF_ENDPOINT` 设置成内网 MatrixHub 地址并启动 vLLM。

```shell
export HF_ENDPOINT="http://192.0.2.10:30001"
vllm serve Qwen/Qwen3-0.6B --max-model-len 1024
```

<img
  src={require('./images/vllm-matrixhub-loading.jpg').default}
  alt="vLLM 通过内网 MatrixHub 下载并加载 Qwen/Qwen3-0.6B"
  width="720"
/>

由图中 log 可知，vLLM 通过内网 MatrixHub 下载模型 `Qwen/Qwen3-0.6B` 并加载成功，模型权重大小约为 1.5 G，当时内网网速为 91 M/s，下载时间约为 16 秒。

## 与 vLLM 部署的模型进行对话

另开一个终端进入 vLLM 运行环境，调用 API 和模型进行简单对话。

```shell
curl "http://localhost:8000/v1/completions" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "Qwen/Qwen3-0.6B",
        "prompt": "San Francisco is a",
        "max_tokens": 20
      }'
```

模型返回应答。

![vLLM 模型应答](./images/vllm-completion-response.png)

## 总结

vLLM 只需要设置 `HF_ENDPOINT` 就可无缝对接内网 MatrixHub 来下载模型，下载时间由内网网速决定。
