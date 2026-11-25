import { Category, type Model } from '@matrixhub/api-ts/v1alpha1/model.pb'

import { getLabelsByCategory } from '@/features/models/models.utils'

export const USE_MODEL_ENGINES = ['transformers', 'vllm', 'sglang'] as const

export type ModelDownloadProvider = 'hf' | 'modelscope'

export function resolveModelDownloadProvider(registryType?: string): ModelDownloadProvider {
  return registryType === 'REGISTRY_TYPE_MODELSCOPE' ? 'modelscope' : 'hf'
}

/** Native ModelScope's master default needs no explicit version in its guide. */
export function resolveModelGuideRevision(provider: ModelDownloadProvider | undefined, revision?: string): string | undefined {
  return provider === 'modelscope' && revision === 'master' ? undefined : revision
}

export type UseModelEngine = (typeof USE_MODEL_ENGINES)[number]

/** Model capability the generated Transformers snippets are specialised for. */
export type UseModelTask = 'text-generation' | 'image-text-to-text'

export const ENGINE_LABELS: Record<UseModelEngine, string> = {
  transformers: 'Transformers',
  vllm: 'vLLM',
  sglang: 'SGLang',
}

export const HF_TOKEN_PLACEHOLDER = '<your-matrixhub-token>'

export const ENGINE_DOC_URLS: Record<UseModelEngine, string> = {
  transformers: 'https://huggingface.co/docs/transformers/index',
  vllm: 'https://docs.vllm.ai/en/latest/',
  sglang: 'https://docs.sglang.ai/',
}

export interface UseModelSnippet {
  code: string
  lang: 'bash' | 'python'
}

/**
 * Pick the task the snippets should target from the model's TASK labels.
 * Only text generation and image-text-to-text are supported; anything else
 * requires the user to choose a supported task explicitly.
 */
export function resolveUseModelTask(model: Pick<Model, 'labels'>): UseModelTask | null {
  const tasks = getLabelsByCategory(model.labels, Category.TASK).map(task => task.toLowerCase())

  if (tasks.some(task => task === 'image-text-to-text' || task === 'image-to-text')) {
    return 'image-text-to-text'
  }

  if (tasks.includes('text-generation')) {
    return 'text-generation'
  }

  return null
}

export function buildEnvSnippet(hfEndpoint: string, provider?: ModelDownloadProvider): UseModelSnippet {
  if (provider === 'modelscope') {
    return {
      lang: 'bash',
      code: `export VLLM_USE_MODELSCOPE=True\nexport MODELSCOPE_ENDPOINT="${hfEndpoint}"\nexport MODELSCOPE_API_TOKEN="${HF_TOKEN_PLACEHOLDER}"`,
    }
  }

  return {
    lang: 'bash',
    code: `${provider === 'hf' ? 'export VLLM_USE_MODELSCOPE=False\n' : ''}export HF_ENDPOINT="${hfEndpoint}"\nexport HF_TOKEN="${HF_TOKEN_PLACEHOLDER}"`,
  }
}

export function buildInstallSnippet(engine: UseModelEngine, provider?: ModelDownloadProvider): UseModelSnippet {
  const commands: Record<UseModelEngine, string> = {
    transformers: 'pip install -U transformers torch accelerate',
    vllm: 'pip install vllm',
    sglang: 'pip install "sglang[all]"',
  }

  return {
    lang: 'bash',
    code: engine === 'vllm' && provider === 'modelscope' ? `${commands[engine]} modelscope modelscope-hub` : commands[engine],
  }
}

export function buildServeSnippet(engine: Exclude<UseModelEngine, 'transformers'>, modelPath: string, revision?: string): UseModelSnippet {
  const commands = {
    vllm: `vllm serve "${modelPath}"${revision ? ` --revision "${revision}"` : ''} --port 8000`,
    sglang: `python -m sglang.launch_server \\\n    --model-path "${modelPath}"${revision ? ` --revision "${revision}"` : ''} \\\n    --host 0.0.0.0 --port 30000`,
  }

  return {
    lang: 'bash',
    code: commands[engine],
  }
}

/** Localised placeholder texts embedded in the generated snippets. */
export interface SnippetPrompts {
  textPrompt: string
  imageUrl: string
  imagePrompt: string
}

const SERVE_PORTS: Record<Exclude<UseModelEngine, 'transformers'>, number> = {
  vllm: 8000,
  sglang: 30000,
}

/** OpenAI-compatible chat completion request; both vLLM and SGLang expose this API. */
export function buildTestRequestSnippet(
  engine: Exclude<UseModelEngine, 'transformers'>,
  modelPath: string,
  task: UseModelTask,
  prompts: SnippetPrompts,
): UseModelSnippet {
  const content = task === 'image-text-to-text'
    ? `[
        {"type": "text", "text": "${prompts.imagePrompt}"},
        {"type": "image_url", "image_url": {"url": "${prompts.imageUrl}"}}
      ]`
    : `"${prompts.textPrompt}"`

  return {
    lang: 'bash',
    code: `curl -X POST "http://localhost:${SERVE_PORTS[engine]}/v1/chat/completions" \\
  -H "Content-Type: application/json" \\
  --data '{
    "model": "${modelPath}",
    "messages": [
      {
        "role": "user",
        "content": ${content}
      }
    ]
  }'`,
  }
}

export function buildDockerSnippet(
  engine: Exclude<UseModelEngine, 'transformers'>,
  hfEndpoint: string,
  modelPath: string,
  provider?: ModelDownloadProvider,
  revision?: string,
): UseModelSnippet {
  if (engine === 'vllm' && provider === 'modelscope') {
    const serve = buildServeSnippet('vllm', modelPath, revision).code

    return {
      lang: 'bash',
      code: `docker run --runtime nvidia --gpus all \\
  -v ~/.cache/modelscope:/root/.cache/modelscope \\
  --env "VLLM_USE_MODELSCOPE=True" \\
  --env "MODELSCOPE_ENDPOINT=${hfEndpoint}" \\
  --env "MODELSCOPE_API_TOKEN=$MODELSCOPE_API_TOKEN" \\
  -p 8000:8000 \\
  --ipc=host \\
  --entrypoint /bin/bash \\
  vllm/vllm-openai:latest \\
  -lc 'python3 -m pip install modelscope modelscope-hub && exec ${serve} --host 0.0.0.0'`,
    }
  }
  const commands = {
    vllm: `docker run --runtime nvidia --gpus all \\
  -v ~/.cache/huggingface:/root/.cache/huggingface \\
  --env "VLLM_USE_MODELSCOPE=False" \\
  --env "HF_ENDPOINT=${hfEndpoint}" \\
  --env "HF_TOKEN=$HF_TOKEN" \\
  -p 8000:8000 \\
  --ipc=host \\
  vllm/vllm-openai:latest \\
  --model "${modelPath}"${revision ? ` --revision "${revision}"` : ''}`,
    sglang: `docker run --gpus all \\
  --shm-size 32g \\
  -p 30000:30000 \\
  -v ~/.cache/huggingface:/root/.cache/huggingface \\
  --env "HF_ENDPOINT=${hfEndpoint}" \\
  --env "HF_TOKEN=$HF_TOKEN" \\
  --ipc=host \\
  lmsysorg/sglang:latest \\
  python3 -m sglang.launch_server \\
    --model-path "${modelPath}"${revision ? ` --revision "${revision}"` : ''} \\
    --host 0.0.0.0 --port 30000`,
  }

  return {
    lang: 'bash',
    code: commands[engine],
  }
}

function buildMessages(task: UseModelTask, prompts: SnippetPrompts): string {
  if (task === 'image-text-to-text') {
    return `messages = [{"role": "user", "content": [
    {"type": "image", "url": "${prompts.imageUrl}"},
    {"type": "text", "text": "${prompts.imagePrompt}"},
]}]`
  }

  return `messages = [{"role": "user", "content": "${prompts.textPrompt}"}]`
}

export function buildPipelineSnippet(task: UseModelTask, modelPath: string, prompts: SnippetPrompts, revision?: string): UseModelSnippet {
  const invocation = task === 'image-text-to-text' ? 'pipe(text=messages)' : 'pipe(messages)'

  return {
    lang: 'python',
    code: `from transformers import pipeline

pipe = pipeline("${task}", model="${modelPath}"${revision ? `, revision=${JSON.stringify(revision)}` : ''})

${buildMessages(task, prompts)}
result = ${invocation}`,
  }
}

/**
 * The lower-level snippet uses the generic Auto* class for the task. Some
 * architectures (Qwen-VL, LLaVA, …) need their own model class; this is the
 * common baseline, not a guarantee for every checkpoint.
 */
export function buildLowLevelSnippet(task: UseModelTask, modelPath: string, prompts: SnippetPrompts, revision?: string): UseModelSnippet {
  if (task === 'image-text-to-text') {
    return {
      lang: 'python',
      code: `import torch
from transformers import AutoProcessor, AutoModelForImageTextToText

model_id = "${modelPath}"
processor = AutoProcessor.from_pretrained(model_id${revision ? `, revision=${JSON.stringify(revision)}` : ''})
model = AutoModelForImageTextToText.from_pretrained(
    model_id,
${revision ? `    revision=${JSON.stringify(revision)},\n` : ''}    device_map="auto",
    torch_dtype="auto",
)

${buildMessages(task, prompts)}

inputs = processor.apply_chat_template(
    messages,
    add_generation_prompt=True,
    tokenize=True,
    return_dict=True,
    return_tensors="pt",
).to(model.device)

outputs = model.generate(**inputs, max_new_tokens=128)
generated = outputs[:, inputs["input_ids"].shape[-1]:]
print(processor.batch_decode(generated, skip_special_tokens=True)[0])`,
    }
  }

  return {
    lang: 'python',
    code: `import torch
from transformers import AutoTokenizer, AutoModelForCausalLM

model_id = "${modelPath}"
tokenizer = AutoTokenizer.from_pretrained(model_id${revision ? `, revision=${JSON.stringify(revision)}` : ''})
model = AutoModelForCausalLM.from_pretrained(
    model_id,
${revision ? `    revision=${JSON.stringify(revision)},\n` : ''}    device_map="auto",
    torch_dtype="auto",
)

${buildMessages(task, prompts)}

inputs = tokenizer.apply_chat_template(
    messages,
    add_generation_prompt=True,
    tokenize=True,
    return_dict=True,
    return_tensors="pt",
).to(model.device)

outputs = model.generate(**inputs, max_new_tokens=128)
generated = outputs[:, inputs["input_ids"].shape[-1]:]
print(tokenizer.decode(generated[0], skip_special_tokens=True))`,
  }
}
