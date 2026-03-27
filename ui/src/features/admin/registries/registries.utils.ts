import type { Registry } from '@matrixhub/api-ts/v1alpha1/registry.pb'

export function getRegistryRowId(registry: Registry) {
  return String(registry.id ?? registry.name ?? '-')
}

export function getRegistryDisplayName(registry: Registry) {
  return registry.name ?? registry.url ?? '-'
}
