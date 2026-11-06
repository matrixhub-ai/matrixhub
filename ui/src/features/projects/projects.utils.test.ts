import {
  describe, expect, it,
} from 'vitest'

import {
  buildProxyDownloadCommand,
  excludeProxyProjects,
  isProxyProject,
} from './projects.utils'

describe('proxy projects', () => {
  it('recognizes a configured registry and builds its upstream download command', () => {
    expect(isProxyProject(' https://huggingface.co ')).toBe(true)
    expect(isProxyProject('')).toBe(false)
    expect(buildProxyDownloadCommand('https://matrixhub.example', 'Qwen', 'Qwen3-0.6B'))
      .toBe('export HF_ENDPOINT=https://matrixhub.example\nhf download Qwen/Qwen3-0.6B')
  })

  it('excludes projects with a registry', () => {
    const local = { name: 'local' }
    const noRegistry = {
      name: 'no-registry',
      enabledRegistry: false,
    }
    const proxy = {
      name: 'proxy',
      enabledRegistry: true,
    }

    expect(excludeProxyProjects([local, proxy, noRegistry])).toEqual([local, noRegistry])
  })

  it('includes a token placeholder for private projects', () => {
    expect(buildProxyDownloadCommand('https://matrixhub.example', 'Qwen', 'Qwen3-0.6B', true))
      .toBe('export HF_ENDPOINT=https://matrixhub.example\nexport HF_TOKEN="<your-matrixhub-token>"\nhf download Qwen/Qwen3-0.6B')
  })
})
