import {
  describe, expect, it,
} from 'vitest'

import {
  buildProxyDownloadCommand,
  getProxyDownloadStepKeys,
  isProxyProject,
  resolveProxyDownloadMethod,
} from './projects.utils'

describe('proxy projects', () => {
  it('recognizes a configured registry and builds its upstream download command', () => {
    expect(isProxyProject(' https://huggingface.co ')).toBe(true)
    expect(isProxyProject('')).toBe(false)
    expect(buildProxyDownloadCommand('https://matrixhub.example', 'Qwen', 'Qwen3-0.6B'))
      .toBe('export HF_ENDPOINT=https://matrixhub.example\nhf download Qwen/Qwen3-0.6B')
  })

  it('includes a token placeholder for private projects', () => {
    expect(buildProxyDownloadCommand('https://matrixhub.example', 'Qwen', 'Qwen3-0.6B', true))
      .toBe('export HF_ENDPOINT=https://matrixhub.example\nexport HF_TOKEN="<your-matrixhub-token>"\nhf download Qwen/Qwen3-0.6B')
  })

  it('offers the download method choice only to users who can pull on the server', () => {
    expect(getProxyDownloadStepKeys(true)).toEqual(['modelName', 'method', 'completion'])
    expect(getProxyDownloadStepKeys(false)).toEqual(['modelName', 'command', 'completion'])
  })

  it('falls back to the CLI download when server pull is not allowed', () => {
    expect(resolveProxyDownloadMethod(true, 'server')).toBe('server')
    expect(resolveProxyDownloadMethod(true, 'cli')).toBe('cli')
    expect(resolveProxyDownloadMethod(false, 'server')).toBe('cli')
  })
})
