---
sidebar_position: 1
---

# Integrations

MatrixHub is built to integrate seamlessly with standard ML systems, high-performance inference engines, cloud object storages, and Kubernetes deployment workflows.

---

## 🚀 GPU Inference Engines

MatrixHub acts as a private, high-speed cache endpoint for your serving nodes. By simply injecting the environment redirect, your serving engines load weights instantly.

### 1. vLLM Integration
To load cached model parameters into a vLLM serving instance, run the startup script with the redirected endpoint:
```bash
# Inject proxy endpoint and launch OpenAI compatible API server
HF_ENDPOINT=http://your-matrixhub-ip:3001 \
vllm serve Qwen/Qwen2.5-7B-Instruct \
  --port 8000 \
  --api-key my-secure-api-key
```

### 2. SGLang Integration
Similarly, route SGLang requests through MatrixHub to enjoy rapid cache-hit load times:
```bash
# Start SGLang engine with private caching endpoint
HF_ENDPOINT=http://your-matrixhub-ip:3001 \
python3 -m sglang.launch_server \
  --model-path Qwen/Qwen2.5-7B-Instruct \
  --port 30000
```

---

## 🪣 Object Storage Backends

MatrixHub is storage agnostic. In production, we highly recommend storing cached large models and private weights inside highly durable, distributed object storage clusters rather than local host folders.

Configure the storage parameters inside your `config/config.yaml`:

```yaml
# MatrixHub Production Storage Configuration
storage:
  mode: s3
  s3:
    endpoint: "play.min.io"             # MinIO or AWS S3 endpoint
    bucket: "matrixhub-model-registry"
    accessKey: "my-s3-access-key"
    secretKey: "my-s3-secret-key"
    secure: true                        # Use HTTPS
    region: "us-east-1"
```

---

## ☸️ Cloud-Native & GitOps Integrations

MatrixHub provides first-class support for Kubernetes deployment and GitOps configuration workflows.

### 1. ArgoCD / Helm Integration
Manage your registry deployment declaratively. Create an ArgoCD application manifest pointing to our official Helm chart parameters:

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
