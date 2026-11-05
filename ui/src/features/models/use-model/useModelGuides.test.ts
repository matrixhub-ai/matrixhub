import { Category } from '@matrixhub/api-ts/v1alpha1/model.pb'
import {
  describe, expect, it,
} from 'vitest'

import {
  buildDockerSnippet,
  buildEnvSnippet,
  buildInstallSnippet,
  buildServeSnippet,
  resolveModelDownloadProvider,
  buildLowLevelSnippet,
  buildPipelineSnippet,
  buildTestRequestSnippet,
  resolveUseModelTask,
  type SnippetPrompts,
} from './useModelGuides'

const prompts: SnippetPrompts = {
  textPrompt: 'Who are you?',
  imageUrl: '<image-url>',
  imagePrompt: 'Describe this image',
}

describe('resolveUseModelTask', () => {
  it('detects image-text-to-text from TASK labels regardless of case', () => {
    expect(resolveUseModelTask({
      labels: [{
        category: Category.TASK,
        name: 'Image-Text-to-Text',
      }],
    })).toBe('image-text-to-text')
  })

  it('detects text generation from TASK labels', () => {
    expect(resolveUseModelTask({
      labels: [{
        category: Category.TASK,
        name: 'text-generation',
      }],
    })).toBe('text-generation')
  })

  it('does not guess a task for unsupported or missing TASK labels', () => {
    expect(resolveUseModelTask({
      labels: [{
        category: Category.LIBRARY,
        name: 'image-text-to-text',
      }],
    })).toBeNull()
    expect(resolveUseModelTask({})).toBeNull()
  })
})

describe('snippet builders', () => {
  it('uses the configured HF endpoint', () => {
    expect(buildEnvSnippet('https://hub.example.com').code)
      .toContain('export HF_ENDPOINT="https://hub.example.com"')
  })

  it('selects ModelScope for its proxy provider and switches every vLLM step together', () => {
    expect(resolveModelDownloadProvider('REGISTRY_TYPE_MODELSCOPE')).toBe('modelscope')
    expect(resolveModelDownloadProvider('REGISTRY_TYPE_HUGGINGFACE')).toBe('hf')
    expect(resolveModelDownloadProvider()).toBe('hf')

    const env = buildEnvSnippet('http://hub.local:9527', 'modelscope').code

    expect(env).toContain('export VLLM_USE_MODELSCOPE=True')
    expect(env).toContain('export MODELSCOPE_ENDPOINT="http://hub.local:9527"')
    expect(env).toContain('export MODELSCOPE_API_TOKEN="<your-matrixhub-token>"')
    expect(env).not.toContain('HF_ENDPOINT')
    expect(buildInstallSnippet('vllm', 'modelscope').code)
      .toBe('pip install vllm modelscope modelscope-hub')
    expect(buildServeSnippet('vllm', 'Eco-Tech/model', 'master').code)
      .toBe('vllm serve "Eco-Tech/model" --revision "master" --port 8000')

    const docker = buildDockerSnippet('vllm', 'http://hub.local:9527', 'Eco-Tech/model', 'modelscope', 'master').code

    expect(docker).toContain('--env "VLLM_USE_MODELSCOPE=True"')
    expect(docker).toContain('--env "MODELSCOPE_ENDPOINT=http://hub.local:9527"')
    expect(docker).toContain('--env "MODELSCOPE_API_TOKEN=$MODELSCOPE_API_TOKEN"')
    expect(docker).toContain('~/.cache/modelscope:/root/.cache/modelscope')
    expect(docker).toContain('pip install modelscope modelscope-hub && exec vllm serve')
    expect(docker).toContain('--revision "master"')
    expect(docker).not.toContain('HF_ENDPOINT')
  })

  it('clears the ModelScope flag when switching vLLM back to Hugging Face', () => {
    expect(buildEnvSnippet('http://hub.local', 'hf').code)
      .toContain('export VLLM_USE_MODELSCOPE=False')
    expect(buildDockerSnippet('vllm', 'http://hub.local', 'org/model', 'hf').code)
      .toContain('--env "VLLM_USE_MODELSCOPE=False"')
  })

  it('preserves non-main model revisions in the other usage examples', () => {
    expect(buildServeSnippet('sglang', 'Qwen/model', 'master').code).toContain('--revision "master"')
    expect(buildDockerSnippet('vllm', 'http://hub', 'Qwen/model', 'hf', 'master').code).toContain('--revision "master"')
    expect(buildDockerSnippet('sglang', 'http://hub', 'Qwen/model', undefined, 'master').code).toContain('--revision "master"')
    expect(buildPipelineSnippet('text-generation', 'Qwen/model', prompts, 'master').code).toContain('revision="master"')
    for (const task of ['text-generation', 'image-text-to-text'] as const) {
      const code = buildLowLevelSnippet(task, 'Qwen/model', prompts, 'master').code

      expect(code.match(/revision="master"/g)).toHaveLength(2)
    }
  })

  it('switches task and messages structure for multimodal models', () => {
    const pipeline = buildPipelineSnippet('image-text-to-text', 'org/model', prompts).code

    expect(pipeline).toContain('pipeline("image-text-to-text"')
    expect(pipeline).toContain('"type": "image", "url": "<image-url>"')
    expect(pipeline).toContain('result = pipe(text=messages)')

    const lowLevel = buildLowLevelSnippet('image-text-to-text', 'org/model', prompts).code

    expect(lowLevel).toContain('AutoModelForImageTextToText')
  })

  it('builds an OpenAI-compatible chat request against the engine port', () => {
    const text = buildTestRequestSnippet('vllm', 'org/model', 'text-generation', prompts).code

    expect(text).toContain('http://localhost:8000/v1/chat/completions')
    expect(text).toContain('"model": "org/model"')
    expect(text).toContain('"content": "Who are you?"')

    const image = buildTestRequestSnippet('sglang', 'org/model', 'image-text-to-text', prompts).code

    expect(image).toContain('http://localhost:30000/v1/chat/completions')
    expect(image).toContain('"type": "image_url"')
    expect(image).toContain('"url": "<image-url>"')
  })

  it('passes the MatrixHub endpoint and token into engine Docker containers', () => {
    const vllm = buildDockerSnippet('vllm', 'https://hub.example.com', 'org/model').code

    expect(vllm).toContain('docker run --runtime nvidia --gpus all')
    expect(vllm).toContain('vllm/vllm-openai:latest')
    expect(vllm).toContain('--env "HF_ENDPOINT=https://hub.example.com"')
    expect(vllm).toContain('--env "HF_TOKEN=$HF_TOKEN"')
    expect(vllm).toContain('--model "org/model"')

    const sglang = buildDockerSnippet('sglang', 'https://hub.example.com', 'org/model').code

    expect(sglang).toContain('lmsysorg/sglang:latest')
    expect(sglang).toContain('--env "HF_ENDPOINT=https://hub.example.com"')
    expect(sglang).toContain('--model-path "org/model"')
  })

  it('does not print in the pipeline snippet', () => {
    expect(buildPipelineSnippet('text-generation', 'org/model', prompts).code).not.toContain('print(')
  })

  it('emits plain text messages for text generation', () => {
    const pipeline = buildPipelineSnippet('text-generation', 'org/model', prompts).code

    expect(pipeline).toContain('pipeline("text-generation"')
    expect(pipeline).toContain('"content": "Who are you?"')
    expect(pipeline).toContain('result = pipe(messages)')
    expect(buildLowLevelSnippet('text-generation', 'org/model', prompts).code).toContain('AutoModelForCausalLM')
  })
})
