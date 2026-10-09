import {
  describe, expect, it,
} from 'vitest'

import { createProjectSchema } from './projects.schema'

const base = {
  name: 'demo-project',
  isPublic: false,
  enabledProxy: false,
}

function issuePaths(input: unknown) {
  const result = createProjectSchema.safeParse(input)

  return result.success
    ? []
    : result.error.issues.map(issue => issue.path.join('.'))
}

describe('createProjectSchema', () => {
  it('accepts a non-proxy project without registry or organization', () => {
    expect(issuePaths(base)).toEqual([])
  })

  it('requires registry and organization once proxy is enabled', () => {
    expect(issuePaths({
      ...base,
      enabledProxy: true,
    }))
      .toEqual(['registryId', 'organization'])
  })

  it('accepts a proxy project with both proxy fields filled', () => {
    expect(issuePaths({
      ...base,
      enabledProxy: true,
      registryId: 1,
      organization: 'Qwen',
    })).toEqual([])
  })

  it('rejects a blank organization on a proxy project', () => {
    expect(issuePaths({
      ...base,
      enabledProxy: true,
      registryId: 1,
      organization: '   ',
    })).toEqual(['organization'])
  })
})
