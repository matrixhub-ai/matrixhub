---
title: 系统集成
sidebar_position: 1
---

# 系统集成

MatrixHub 采用标准云原生协议设计，旨在无缝融入企业级 AI 基础设施生态，支持主流高性能推理引擎、高可用分布式对象存储底座以及 Kubernetes 云原生 CI/CD & GitOps 工作流。

---

## 🚀 高性能 GPU 推理引擎集成

MatrixHub 作为高性能代理缓存，可以直接拦截并拦截内网 GPU 节点的权重加载请求。在启动主流推理框架时，仅需注入重定向环境变量即可：

### 1. 与 vLLM 对接
将环境变量注入您的 vLLM 服务启动脚本，即可让推理算力节点直接拉取 MatrixHub 中的模型文件：
```bash
# 注入本地代理地址，启动兼容 OpenAI 规范的 vLLM 接口服务
HF_ENDPOINT=http://your-matrixhub-ip:3001 \
vllm serve Qwen/Qwen2.5-7B-Instruct \
  --port 8000 \
  --api-key my-secure-api-key
```

### 2. 与 SGLang 对接
同样地，SGLang 实例在启动时可以无缝拦截公网下载，享受高速局域网（10Gbps+）的缓存载入速度：
```bash
# 注入环境变量启动 SGLang 推理底座
HF_ENDPOINT=http://your-matrixhub-ip:3001 \
python3 -m sglang.launch_server \
  --model-path Qwen/Qwen2.5-7B-Instruct \
  --port 30000
```

---

## 🪣 分布式对象存储集成

MatrixHub 本身是存储解耦的。在生产部署中，我们强烈推荐您将模型权重和代理缓存持久化存储在具备多机冗余的对象存储集群（如 AWS S3、MinIO）中，而不是直接依赖单机磁盘或 NFS。

编辑您的主配置文件 `config/config.yaml` 引入 S3 存储引擎：

```yaml
# MatrixHub 生产存储连接配置
storage:
  mode: s3
  s3:
    endpoint: "play.min.io"             # MinIO 或 AWS S3 的访问地址
    bucket: "matrixhub-model-registry"  # 存储桶名称
    accessKey: "my-s3-access-key"       # 秘钥 ID
    secretKey: "my-s3-secret-key"       # 秘钥值
    secure: true                        # 启用 HTTPS 安全传输
    region: "us-east-1"
```

---

## ☸️ 云原生编排与 GitOps 集成

为了满足企业级“基础设施即代码 (IaC)”的自动化部署规范，MatrixHub 提供了官方 Helm Chart，完美适配各种主流 GitOps 管理软件。

### 1. 与 ArgoCD 的持续集成集成
您可以配置声明式的 ArgoCD Application 清单，一键同步与更新您的 MatrixHub 注册表集群：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: matrixhub-registry
  namespace: argocd
spec:
  project: default
  source:
    chart: matrixhub
    repoURL: oci://ghcr.io/matrixhub-ai/matrixhub
    targetRevision: 0.1.0
    helm:
      parameters:
        - name: apiserver.service.type
          value: NodePort
        - name: apiserver.storage.pvc.size
          value: 500Gi
  destination:
    server: "https://kubernetes.default.svc"
    namespace: matrixhub
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```
