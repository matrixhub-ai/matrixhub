import {
  describe, expect, it,
} from 'vitest'

import { decodeHash } from './hash'

describe('decodeHash', () => {
  it('decodes valid hashes and preserves malformed ones', () => {
    expect(decodeHash('best%20practices')).toBe('best practices')
    expect(decodeHash('%E0%A4%A')).toBe('%E0%A4%A')
  })
})
