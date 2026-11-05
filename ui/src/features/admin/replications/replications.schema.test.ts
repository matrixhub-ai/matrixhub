import { RegistryType } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import {
  ResourceType, SyncPolicyType, TriggerType,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import {
  describe, expect, it,
} from 'vitest'

import { createReplicationFormSchema, type ReplicationFormValues } from './replications.schema'

const values: ReplicationFormValues = {
  name: 'qa-rule',
  description: '',
  policyType: SyncPolicyType.SYNC_POLICY_TYPE_PULL_BASE,
  triggerType: TriggerType.TRIGGER_TYPE_MANUAL,
  cronExpression: '',
  bandwidth: '-1',
  bandwidthUnit: 'Kbps',
  isOverwrite: true,
  resourceName: 'Qwen/model',
  resourceTypes: [ResourceType.RESOURCE_TYPE_MODEL, ResourceType.RESOURCE_TYPE_DATASET],
  sourceRegistryId: 2,
  targetProjectName: 'local',
  targetRegistryId: 0,
}

describe('registry resource capabilities', () => {
  it('rejects ModelScope dataset pulls before creating jobs', () => {
    const schema = createReplicationFormSchema([{
      id: 2,
      type: RegistryType.REGISTRY_TYPE_MODELSCOPE,
    }])
    const result = schema.safeParse(values)

    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues.some(issue => issue.path[0] === 'resourceTypes')).toBe(true)
    }
    expect(schema.safeParse({
      ...values,
      resourceTypes: [ResourceType.RESOURCE_TYPE_MODEL],
    }).success).toBe(true)
  })

  it('keeps both concrete resource types for HF and ordinary MatrixHub registries', () => {
    for (const type of [RegistryType.REGISTRY_TYPE_HUGGINGFACE, RegistryType.REGISTRY_TYPE_MATRIXHUB]) {
      expect(createReplicationFormSchema([{
        id: 2,
        type,
      }]).safeParse(values).success).toBe(true)
    }
  })
})
