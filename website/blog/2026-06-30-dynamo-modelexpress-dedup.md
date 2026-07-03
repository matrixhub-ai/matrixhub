---
slug: /dynamo-modelexpress-dedup
title: Deduplicating model downloads across Dynamo workers with ModelExpress
description: Deploying two Dynamo vLLM workers with ModelExpress on a GPU Kubernetes cluster, showing that the second worker skips the model download entirely and only streams from the ModelExpress cache.
---

When scaling an inference service to multiple workers, every new worker downloads the full model from the model registry. For a 3 GB model this adds 30–40 seconds per worker; for a 70B model it can be 10+ minutes each.

[ModelExpress](https://docs.nvidia.com/dynamo/kubernetes-deployment/model-loading/model-express) is a model distribution cache layer in NVIDIA Dynamo. It sits between the workers and the model source (MatrixHub or Hugging Face). The first worker triggers a download into the ModelExpress cache. Every subsequent worker gets the model from that cache — no second download.

In this test we deploy two Dynamo vLLM workers for `Qwen/Qwen2.5-1.5B-Instruct` (~3 GB) and compare the model acquisition time of the first worker (cache miss) versus the second worker (cache hit).

<!-- truncate -->

## Environment

| Component | Configuration |
|---|---|
| GPU | HAMi vGPU |
| Model | Qwen/Qwen2.5-1.5B-Instruct (~3 GB) |
| ModelExpress | v0.3.0 |
| MatrixHub | self-hosted, Hugging Face-compatible endpoint |
| Storage mode | `NO_SHARED_STORAGE=1` (gRPC streaming) |

## How it works

```
┌──────────┐     ┌──────────────┐     ┌────────────┐
│ Worker 1 │────▶│ ModelExpress │────▶│ MatrixHub  │
│ Worker 2 │────▶│   (cache)    │     │  (registry) │
│ Worker N │────▶│              │     └────────────┘
└──────────┘     └──────────────┘
```

- **First request**: ModelExpress downloads the model from MatrixHub into its local cache, then streams the files to the requesting worker over gRPC.
- **Subsequent requests**: ModelExpress streams directly from its local cache. No download from MatrixHub.

Compared to a plain MatrixHub setup (Blog 1), the decode component adds three environment variables:

- `VLLM_PLUGINS=modelexpress` — enable the ModelExpress vLLM plugin
- `MODEL_EXPRESS_NO_SHARED_STORAGE=1` — use gRPC streaming instead of a shared filesystem
- `MODEL_EXPRESS_URL` — the ModelExpress server address

## Deployment files

### Worker 1: dgd-blog2-c-mx.yaml

```yaml
apiVersion: nvidia.com/v1beta1
kind: DynamoGraphDeployment
metadata:
  name: vllm-7b-c
  namespace: dynamo-system
spec:
  components:
    - name: Frontend
      type: frontend
      replicas: 1
      podTemplate:
        spec:
          containers:
            - name: main
              image: nvcr.io/nvidia/ai-dynamo/vllm-runtime:latest
              workingDir: /workspace
              env:
                - name: HF_ENDPOINT
                  value: "http://<matrixhub-endpoint>"
              command: ["python3", "-m", "dynamo.frontend"]
              args: ["--http-port", "8000"]
              resources:
                requests:
                  cpu: "2"
                  memory: "4Gi"
                limits:
                  cpu: "2"
                  memory: "4Gi"
    - name: decode
      type: decode
      replicas: 1
      podTemplate:
        spec:
          containers:
            - name: main
              image: nvcr.io/nvidia/ai-dynamo/vllm-runtime:latest
              workingDir: /workspace
              env:
                - name: HF_ENDPOINT
                  value: "http://<matrixhub-endpoint>"
                - name: VLLM_PLUGINS
                  value: "modelexpress"
                - name: MODEL_EXPRESS_NO_SHARED_STORAGE
                  value: "1"
                - name: MODEL_EXPRESS_URL
                  value: "http://<modelexpress-service>:8001"
              command: ["python3", "-m", "dynamo.vllm"]
              args:
                - --model
                - Qwen/Qwen2.5-1.5B-Instruct
                - --served-model-name
                - Qwen/Qwen2.5-1.5B-Instruct
                - --tensor-parallel-size
                - "1"
                - --gpu-memory-utilization
                - "0.85"
                - --max-model-len
                - "8192"
                - --no-enable-log-requests
              resources:
                requests:
                  cpu: "4"
                  memory: "16Gi"
                  nvidia.com/vgpu: "1"
                  nvidia.com/gpucores: "30"
                  nvidia.com/gpumem: "10000"
                limits:
                  cpu: "4"
                  memory: "16Gi"
                  nvidia.com/vgpu: "1"
                  nvidia.com/gpucores: "30"
                  nvidia.com/gpumem: "10000"
```

### Worker 2: dgd-blog2-c2-mx.yaml

Worker 2 is a separate DynamoGraphDeployment with the same configuration but a different name (`vllm-7b-c2`). The full YAML is identical except for `metadata.name`.

## Clear the ModelExpress cache

ModelExpress stores model files on a PVC (`local-path`). Restarting the pod alone does not remove the cached files. To start from a clean state:

```bash
kubectl exec -n model-express deploy/model-express-modelexpress \
  -- rm -rf /root/models--Qwen--Qwen2.5-1.5B-Instruct /root/blobs

kubectl rollout restart deployment/model-express-modelexpress -n model-express
kubectl rollout status deployment/model-express-modelexpress -n model-express
```

Verify:

```bash
kubectl exec -n model-express deploy/model-express-modelexpress \
  -- ls /root/models--Qwen--Qwen2.5-1.5B-Instruct
# Expected: ls: cannot access ... No such file or directory
```

## Deploy Worker 1

```bash
kubectl apply -f dgd-blog2-c-mx.yaml
kubectl get pods -n dynamo-system -o wide -w
```

Watch the decode pod logs:

```bash
kubectl logs -n dynamo-system -f <c-decode-pod>
```

```
2026-06-30T02:50:38 INFO dynamo_llm::hub: Successfully connected to ModelExpress server
2026-06-30T02:50:38 INFO modelexpress_client: Requesting model: Qwen/Qwen2.5-1.5B-Instruct from provider: HuggingFace
2026-06-30T02:50:38 INFO modelexpress_client: Model Qwen/Qwen2.5-1.5B-Instruct: Model download in progress
2026-06-30T02:51:16 INFO modelexpress_client: Model Qwen/Qwen2.5-1.5B-Instruct: Model download completed successfully
2026-06-30T02:51:16 INFO modelexpress_client: Shared storage disabled, streaming files from server for model Qwen/Qwen2.5-1.5B-Instruct
2026-06-30T02:51:16 INFO modelexpress_client: Streaming model Qwen/Qwen2.5-1.5B-Instruct files to "/home/dynamo/.model-express/cache" with chunk size 32768 bytes
2026-06-30T02:51:38 INFO modelexpress_client: Streaming complete: received 8 files (3098967011 bytes) for model Qwen/Qwen2.5-1.5B-Instruct
```

Worker 1 waited 37.8 seconds for ModelExpress to download the model from MatrixHub, then received the files over gRPC streaming in 21.9 seconds.

## Deploy Worker 2

After Worker 1 is ready, deploy Worker 2:

```bash
kubectl apply -f dgd-blog2-c2-mx.yaml
kubectl get pods -n dynamo-system -o wide -w
```

Watch the decode pod logs:

```bash
kubectl logs -n dynamo-system -f <c2-decode-pod>
```

```
2026-06-30T02:55:00 INFO dynamo_llm::hub: Successfully connected to ModelExpress server
2026-06-30T02:55:00 INFO modelexpress_client: Requesting model: Qwen/Qwen2.5-1.5B-Instruct from provider: HuggingFace
2026-06-30T02:55:00 INFO modelexpress_client: Model Qwen/Qwen2.5-1.5B-Instruct: Model already downloaded
2026-06-30T02:55:00 INFO modelexpress_client: Shared storage disabled, streaming files from server for model Qwen/Qwen2.5-1.5B-Instruct
2026-06-30T02:55:00 INFO modelexpress_client: Streaming model Qwen/Qwen2.5-1.5B-Instruct files to "/home/dynamo/.model-express/cache" with chunk size 32768 bytes
2026-06-30T02:55:23 INFO modelexpress_client: Streaming complete: received 8 files (3098967011 bytes) for model Qwen/Qwen2.5-1.5B-Instruct
```

Worker 2 sees `Model already downloaded` — ModelExpress skips the download entirely and streams directly from its local cache in 22.6 seconds.

## Verify the inference service

After both workers are ready, test that each can serve inference requests:

```bash
# Worker 1
kubectl exec -n dynamo-system <c-frontend-pod> -- \
  curl -s http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen2.5-1.5B-Instruct","messages":[{"role":"user","content":"hi"}],"max_tokens":20}' \
  | python3 -m json.tool

# Worker 2
kubectl exec -n dynamo-system <c2-frontend-pod> -- \
  curl -s http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen2.5-1.5B-Instruct","messages":[{"role":"user","content":"hello"}],"max_tokens":20}' \
  | python3 -m json.tool
```

Both return a normal response:

```json
{
    "choices": [
        {
            "message": {
                "content": "Hello! How can I assist you today?",
                "role": "assistant"
            },
            "finish_reason": "stop"
        }
    ],
    "model": "Qwen/Qwen2.5-1.5B-Instruct"
}
```

## Results

| | MX → MatrixHub download | gRPC streaming | Total model acquisition |
|---|---:|---:|---:|
| Worker 1 (cache miss) | 37.8 s | 21.9 s | **59.7 s** |
| Worker 2 (cache hit) | 0 s | 22.6 s | **22.6 s** |

For comparison, without ModelExpress (Blog 1):

| Source | Model acquisition |
|---|---:|
| Public Hugging Face | ~10 min 32 s |
| MatrixHub direct | 29 s |

Worker 2 saved the full 37.8-second MatrixHub download. With more workers, the saving multiplies: N workers share one download.

## Notes

**First-worker overhead.** The first worker through ModelExpress (59.7 s) is slower than a direct MatrixHub download (29 s), because the model passes through an extra gRPC streaming hop (~22 s). ModelExpress pays off when multiple workers need the same model.

**Streaming throughput.** The gRPC streaming stage transfers 3 GB in ~22 seconds (~137 MB/s). The current implementation uses a 32 KB chunk size with about 94,000 iterations. With shared storage (`NO_SHARED_STORAGE=0`), workers can mount the ModelExpress cache directory directly and skip streaming entirely — model acquisition drops to near zero for cached models.

**When to use what.**

| Scenario | Recommendation |
|---|---|
| Single worker | MatrixHub direct download (fastest) |
| Multiple workers scaling out | ModelExpress + MatrixHub |
| Shared filesystem available (NFS/Lustre) | ModelExpress shared_storage mode |
| No shared filesystem | ModelExpress NO_SHARED_STORAGE streaming mode |
