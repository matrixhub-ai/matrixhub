import {
  ResourceType,
  SyncPolicyType,
  TriggerType,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { z } from 'zod'

import i18n from '@/i18n'

export const DEFAULT_REPLICATIONS_PAGE = 1
export const DEFAULT_REPLICATIONS_PAGE_SIZE = 10

export const replicationResourceTypeValues = [
  ResourceType.RESOURCE_TYPE_ALL,
  ResourceType.RESOURCE_TYPE_MODEL,
  ResourceType.RESOURCE_TYPE_DATASET,
] as const
export type ReplicationResourceTypeValue = typeof replicationResourceTypeValues[number]

export const replicationTriggerTypeValues = [
  TriggerType.TRIGGER_TYPE_MANUAL,
  TriggerType.TRIGGER_TYPE_SCHEDULED,
] as const

export const replicationBandwidthUnitValues = ['Kbps', 'Mbps'] as const
export type ReplicationBandwidthUnit = typeof replicationBandwidthUnitValues[number]

export const replicationsSearchSchema = z.object({
  page: z.coerce.number().int().positive().optional().catch(DEFAULT_REPLICATIONS_PAGE),
  query: z.string().trim().optional().catch(undefined),
})

export type ReplicationsSearch = z.infer<typeof replicationsSearchSchema>

function t(key: string) {
  return i18n.getFixedT(i18n.resolvedLanguage ?? i18n.language)(key)
}

function replicationFormBaseSchema() {
  return z.object({
    name: z
      .string()
      .trim()
      .min(2)
      .regex(/^[a-z0-9]/u, t('routes.admin.replications.validation.nameStartChar'))
      .regex(/^[a-z0-9._-]+$/u, t('routes.admin.replications.validation.nameInvalidChars')),
    description: z.string().trim().max(50),
    policyType: z.union([
      z.literal(SyncPolicyType.SYNC_POLICY_TYPE_PULL_BASE),
      z.literal(SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE),
    ]),
    triggerType: z.enum(replicationTriggerTypeValues),
    bandwidth: z.string().trim().optional(),
    bandwidthUnit: z.enum(replicationBandwidthUnitValues),
    isOverwrite: z.boolean(),
    resourceName: z.string().trim().optional(),
    resourceType: z.enum(replicationResourceTypeValues),
    sourceRegistryId: z.number(),
    targetProjectName: z.string().trim().optional(),
    targetRegistryId: z.number(),
  })
}

export function createReplicationFormSchema() {
  return replicationFormBaseSchema().superRefine((data, ctx) => {
    if (!data.sourceRegistryId || data.sourceRegistryId < 1) {
      ctx.addIssue({
        code: 'custom',
        message: t('routes.admin.replications.validation.sourceRegistryRequired'),
        path: ['sourceRegistryId'],
      })
    }

    if (
      data.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE
      && (!data.targetRegistryId || data.targetRegistryId < 1)
    ) {
      ctx.addIssue({
        code: 'custom',
        message: t('routes.admin.replications.validation.targetRegistryRequired'),
        path: ['targetRegistryId'],
      })
    }
  })
}

export type ReplicationFormValues = z.infer<ReturnType<typeof createReplicationFormSchema>>
