---
sidebar_position: 1
---

# Integrations

MatrixHub is built to integrate seamlessly with standard ML systems and high-performance inference frameworks.

---

## 🚀 GPU Inference Engines

MatrixHub acts as a private, high-speed cache endpoint for your serving nodes. Set the `HF_ENDPOINT` redirect when starting an inference engine. For detailed integration instructions, see:

- **vLLM**

  Set `HF_ENDPOINT` in the vLLM runtime environment to load models through MatrixHub.

  [View the detailed vLLM integration guide](../guides/use-with-vllm.md)

- **SGLang**

  Set `HF_ENDPOINT` in the SGLang runtime environment to load models from the MatrixHub cache.

  [View the detailed SGLang integration guide](../../blog/sglang-matrixhub-cache-acceleration)

- **llm-d**

  Inject `HF_ENDPOINT` into the llm-d model server to distribute models through MatrixHub inside the cluster.

  [View the detailed llm-d integration guide](../../blog/llmd-qwen3-32b-matrixhub-cache)

- **Dynamo**

  Set `HF_ENDPOINT` in the Dynamo deployment so the inference runtime retrieves models through MatrixHub.

  [View the detailed Dynamo integration guide](../../blog/dynamo-matrixhub-integration)

---

## Model Distribution and P2P Acceleration

- **ModelExpress cache reuse**

  Use MatrixHub as the model source while ModelExpress caches model downloads for reuse across Dynamo workers.

  [View the ModelExpress cache integration guide](../../blog/dynamo-modelexpress-dedup)

- **Dynamo GPU P2P**

  Use MatrixHub for model-file distribution and ModelExpress to transfer loaded GPU weights between Dynamo workers over NIXL, UCX, and RDMA.

  [View the Dynamo P2P integration guide](../../blog/matrixhub-modelexpress-dynamo-p2p)
