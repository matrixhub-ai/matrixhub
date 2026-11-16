import {
  describe, expect, it,
} from 'vitest'

import {
  buildDownloadSnippet,
  buildModelScopeDownloadSnippet,
  buildModelScopeEnvSnippet,
  buildModelScopePythonSnippet,
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

  it('builds ModelScope downloads against the configured MatrixHub endpoint', () => {
    expect(buildModelScopeEnvSnippet('http://hub.local:9527').code)
      .toBe('export MODELSCOPE_ENDPOINT="http://hub.local:9527"')
    expect(buildModelScopeDownloadSnippet('team/model').code)
      .toBe('ms-hub download "team/model" --local-dir "./model"')
    expect(buildModelScopePythonSnippet('team/model', 'http://hub.local:9527').code)
      .toContain('endpoint="http://hub.local:9527"')
  })

  it('keeps the upstream default branch in both download commands', () => {
    expect(buildDownloadSnippet('team/model', 'master').code).toContain('--revision "master"')
    expect(buildModelScopeDownloadSnippet('team/model', 'release/v1').code).toContain('--revision "release/v1"')
  })

  it('derives the local path placeholder from the model name', () => {
    expect(buildLocalPathPlaceholder('Qwen/Qwen3-0.6B')).toBe('/path/to/Qwen3-0.6B')
    expect(buildLocalPathPlaceholder('')).toBe('/path/to/model')
  })
})
