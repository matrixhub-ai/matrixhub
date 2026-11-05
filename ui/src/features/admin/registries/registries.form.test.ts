import { RegistryType } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import {
  describe, expect, it,
} from 'vitest'

import {
  defaultRegistryFormValues,
  getRegistryDefaultUrl,
  getRegistryUrlCopyKeys,
  registryProviderLabelKeys,
  supportedRegistryTypes,
} from './registries.form'

describe('ModelScope registry provider', () => {
  it('offers ModelScope alongside the existing providers with its own URL and guidance', () => {
    expect(supportedRegistryTypes).toContain(RegistryType.REGISTRY_TYPE_MODELSCOPE)
    expect(getRegistryDefaultUrl(RegistryType.REGISTRY_TYPE_MODELSCOPE)).toBe('https://www.modelscope.cn')
    expect(registryProviderLabelKeys[RegistryType.REGISTRY_TYPE_MODELSCOPE]).toBe('routes.admin.registries.provider.modelScope')
    expect(getRegistryUrlCopyKeys(RegistryType.REGISTRY_TYPE_MODELSCOPE).hint).toBe('routes.admin.registries.form.urlHint.modelScope')
    expect(defaultRegistryFormValues.verifyRemoteCert).toBe(true)
  })
})
