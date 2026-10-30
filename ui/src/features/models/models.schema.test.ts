import {
  describe, expect, it,
} from 'vitest'

import {
  createModelSchema,
  createUpstreamModelNameSchema,
} from './models.schema'

import type { TFunction } from 'i18next'

const t = ((key: string) => key) as unknown as TFunction

describe('createUpstreamModelNameSchema', () => {
  const schema = createUpstreamModelNameSchema(t)

  // Hugging Face repositories may start with '_', and the cache API accepts them.
  it('accepts upstream model names, also with a leading underscore', () => {
    for (const name of ['Qwen3-0.6B', 'gpt2', '_tiny-model', ' Qwen3-0.6B ']) {
      expect(schema.safeParse(name).success).toBe(true)
    }
  })

  it('rejects names that the API rejects', () => {
    for (const name of ['', '   ', '-model', '.model', 'Qwen/Qwen3-0.6B', 'my model']) {
      expect(schema.safeParse(name).success).toBe(false)
    }
  })
})

describe('createModelSchema', () => {
  it('keeps rejecting a leading underscore for models created in MatrixHub', () => {
    expect(createModelSchema(t).shape.name.safeParse('_tiny-model').success).toBe(false)
  })
})
