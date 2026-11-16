import {
  describe, expect, it,
} from 'vitest'

import { buildProxyDownloadCommand, isProxyProject } from './projects.utils'

describe('proxy projects', () => {
  it('recognizes a configured registry and builds its upstream download command', () => {
    expect(isProxyProject(' https://huggingface.co ')).toBe(true)
    expect(isProxyProject('')).toBe(false)
    expect(buildProxyDownloadCommand('https://matrixhub.example', 'Qwen', 'Qwen3-0.6B'))
      .toBe('export HF_ENDPOINT=https://matrixhub.example\nhf download Qwen/Qwen3-0.6B')
  })

  it('uses ModelScope for a ModelScope registry, including a private token', () => {
    expect(buildProxyDownloadCommand('http://hub', 'Qwen', 'model', false, 'REGISTRY_TYPE_MODELSCOPE'))
      .toBe('export MODELSCOPE_ENDPOINT=http://hub\nms-hub download Qwen/model')
    expect(buildProxyDownloadCommand('http://hub', 'local', 'model', true, 'REGISTRY_TYPE_MODELSCOPE'))
      .toBe('export MODELSCOPE_ENDPOINT=http://hub\nexport MODELSCOPE_API_TOKEN="<your-matrixhub-token>"\nms-hub download local/model')
  })

  it('includes a token placeholder for private projects', () => {
    expect(buildProxyDownloadCommand('https://matrixhub.example', 'Qwen', 'Qwen3-0.6B', true))
      .toBe('export HF_ENDPOINT=https://matrixhub.example\nexport HF_TOKEN="<your-matrixhub-token>"\nhf download Qwen/Qwen3-0.6B')
  })
})
