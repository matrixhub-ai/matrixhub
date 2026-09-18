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
})
