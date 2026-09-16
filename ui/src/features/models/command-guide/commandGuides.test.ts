import {
  describe, expect, it,
} from 'vitest'

import {
  buildDownloadSnippet,
  buildHfEnvSnippet,
  buildLocalPathPlaceholder,
  buildUploadSnippet,
} from './commandGuides'

describe('command guide snippets', () => {
  it('only asks for a token when uploading', () => {
    expect(buildHfEnvSnippet('download', 'https://hub.example.com').code)
      .toBe('export HF_ENDPOINT="https://hub.example.com"')
    expect(buildHfEnvSnippet('upload', 'https://hub.example.com').code)
      .toContain('export HF_TOKEN=')
  })

  it('targets the model path in hf commands', () => {
    expect(buildDownloadSnippet('org/model').code).toBe('hf download "org/model"')

    const upload = buildUploadSnippet('org/model').code

    expect(upload).toContain('hf upload')
    expect(upload).toContain('"org/model"')
    expect(upload).toContain('"/path/to/model"')
  })

  it('derives the local path placeholder from the model name', () => {
    expect(buildLocalPathPlaceholder('Qwen/Qwen3-0.6B')).toBe('/path/to/Qwen3-0.6B')
    expect(buildLocalPathPlaceholder('')).toBe('/path/to/model')
  })
})
