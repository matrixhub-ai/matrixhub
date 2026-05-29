---
sidebar_position: 1
---

# Getting Started

Welcome to MatrixHub! MatrixHub streamlines the transition from public model hubs to private, enterprise-grade AI infrastructure. This guide gets you up and running in minutes.

---

## 🚀 Step 1: Deploy MatrixHub

Deploy a local MatrixHub registry instance using **Docker Compose** in seconds:

```bash
# Fetch and launch MatrixHub container services in background
curl -fsSL https://bit.ly/4qqSZIG | docker compose -f - up -d
```

Your private registry is now running locally on port `3001`:
*   **Service API Endpoint**: `http://127.0.0.1:3001`
*   **Web Console**: Access the control panel via `http://localhost:3001`

---

## 🛠️ Step 2: Configure Client Redirection

To pull models through MatrixHub instead of the public internet, you only need to redirect the endpoint environment variable. There is **zero code change** required in your codebases.

Set the global environment variable:

```bash
# Point Hugging Face API queries to MatrixHub
export HF_ENDPOINT=http://127.0.0.1:3001
```

---

## 💻 Step 3: Verify Your Integrations

Once the `HF_ENDPOINT` is redirected, any tool that depends on the official Hugging Face Hub library will automatically fetch models and cache them inside MatrixHub.

### Option A: Using Hugging Face CLI
Download a model cache directly from CLI:
```bash
# Download a public model (e.g. Qwen2.5-0.5B) through the local registry proxy
huggingface-cli download Qwen/Qwen2.5-0.5B
```

### Option B: Using Python Transformers
```python
import os
# Ensure environment variable is set
os.environ["HF_ENDPOINT"] = "http://127.0.0.1:3001"

from transformers import AutoModelForCausalLM, AutoTokenizer

# MatrixHub intercepts this call, pulls it from public HF, caches it locally,
# and returns the weights. Subsequent requests take milliseconds over LAN.
model_id = "Qwen/Qwen2.5-0.5B-Instruct"
tokenizer = AutoTokenizer.from_pretrained(model_id)
model = AutoModelForCausalLM.from_pretrained(model_id)
```

### Option C: Launching a vLLM Server
Launch high-throughput GPU inference with vLLM, loading weights from your local cache:
```bash
# Run vLLM with private endpoint redirection
HF_ENDPOINT=http://127.0.0.1:3001 \
python3 -m vllm.entrypoints.openai.api_server \
  --model Qwen/Qwen2.5-7B-Instruct \
  --port 8000
```

---

## 🔍 Step 4: Confirm Cache Status

To confirm that MatrixHub is successfully caching models:
1.  Open your browser and navigate to the MatrixHub Web Console at `http://localhost:3001`.
2.  Click on the **Repositories** tab.
3.  You will see your cached models (e.g. `Qwen/Qwen2.5-0.5B`) marked as synced and available locally for your entire GPU cluster.
