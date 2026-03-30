import {
  Checkbox,
  Group,
  NumberInput,
  Radio,
  RadioGroup,
  Select,
  Stack,
  TextInput,
  Textarea,
} from '@mantine/core'
import {
  ResourceType,
  SyncPolicyType,
  TriggerType,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { useForm, useStore } from '@tanstack/react-form'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { registriesQueryOptions } from '@/features/admin/registries/registries.query'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { fieldError } from '@/shared/utils/form'

import {
  createReplicationMutationOptions,
  updateReplicationMutationOptions,
} from '../replications.mutation'
import {
  createReplicationFormSchema,
  type ReplicationResourceTypeValue,
  type ReplicationFormValues,
  replicationResourceTypeValues,
  replicationTriggerTypeValues,
} from '../replications.schema'
import {
  convertBandwidthToApiValue,
  getDefaultBandwidthUnit,
  getDefaultBandwidthValue,
} from '../replications.utils'

import type {
  CreateSyncPolicyRequest,
  SyncPolicyItem,
  UpdateSyncPolicyRequest,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'

const BANDWIDTH_MIN = -1
const BANDWIDTH_MAX = 1048576
const DESCRIPTION_MAX_LENGTH = 50

export interface ReplicationFormModalProps {
  mode: 'create' | 'edit'
  opened: boolean
  syncPolicy?: SyncPolicyItem
  onClose: () => void
}

// The backend accepts `resourceTypes: ResourceType[]`, but the current UI
// intentionally supports only a single selection: All, Model, or Dataset.
function normalizeResourceTypesForForm(
  resourceTypes?: readonly ResourceType[],
): ReplicationResourceTypeValue {
  const selected = (resourceTypes ?? []).filter(resourceType =>
    resourceType === ResourceType.RESOURCE_TYPE_MODEL
    || resourceType === ResourceType.RESOURCE_TYPE_DATASET,
  )

  if (
    selected.length !== 1
    || (resourceTypes ?? []).includes(ResourceType.RESOURCE_TYPE_ALL)
  ) {
    return ResourceType.RESOURCE_TYPE_ALL
  }

  return selected[0]
}

function buildResourceTypesForApi(
  resourceType: ReplicationResourceTypeValue,
): ResourceType[] {
  if (resourceType === ResourceType.RESOURCE_TYPE_ALL) {
    return []
  }

  return [resourceType]
}

function buildPolicyPayload(values: ReplicationFormValues) {
  const bandwidth = convertBandwidthToApiValue(values.bandwidth ?? '', values.bandwidthUnit)
  const resourceTypes = buildResourceTypesForApi(values.resourceType)
  const base = {
    description: values.description,
    triggerType: values.triggerType,
    bandwidth,
    isOverwrite: values.isOverwrite,
  }

  if (values.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE) {
    return {
      ...base,
      pushBasePolicy: {
        resourceName: values.resourceName ?? '',
        resourceTypes,
        targetRegistryId: values.targetRegistryId,
        targetProjectName: values.targetProjectName ?? '',
      },
    }
  }

  return {
    ...base,
    pullBasePolicy: {
      sourceRegistryId: values.sourceRegistryId,
      resourceName: values.resourceName ?? '',
      resourceTypes,
      targetProjectName: values.targetProjectName ?? '',
    },
  }
}

function getFormDefaults(syncPolicy?: SyncPolicyItem): ReplicationFormValues {
  const isPush = syncPolicy?.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE
  const rawResourceTypes = isPush
    ? syncPolicy?.pushBasePolicy?.resourceTypes
    : syncPolicy?.pullBasePolicy?.resourceTypes

  return {
    name: syncPolicy?.name ?? '',
    description: syncPolicy?.description ?? '',
    policyType: isPush
      ? SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE
      : SyncPolicyType.SYNC_POLICY_TYPE_PULL_BASE,
    triggerType: syncPolicy?.triggerType === TriggerType.TRIGGER_TYPE_SCHEDULED
      ? TriggerType.TRIGGER_TYPE_SCHEDULED
      : TriggerType.TRIGGER_TYPE_MANUAL,
    bandwidth: getDefaultBandwidthValue(syncPolicy?.bandwidth),
    bandwidthUnit: getDefaultBandwidthUnit(syncPolicy?.bandwidth),
    isOverwrite: syncPolicy?.isOverwrite ?? true,
    resourceName: isPush
      ? (syncPolicy?.pushBasePolicy?.resourceName ?? '')
      : (syncPolicy?.pullBasePolicy?.resourceName ?? ''),
    resourceType: normalizeResourceTypesForForm(rawResourceTypes),
    sourceRegistryId: syncPolicy?.pullBasePolicy?.sourceRegistryId ?? 0,
    targetProjectName: isPush
      ? (syncPolicy?.pushBasePolicy?.targetProjectName ?? '')
      : (syncPolicy?.pullBasePolicy?.targetProjectName ?? ''),
    targetRegistryId: syncPolicy?.pushBasePolicy?.targetRegistryId ?? 0,
  }
}

export function ReplicationFormModal({
  mode,
  opened,
  syncPolicy,
  onClose,
}: ReplicationFormModalProps) {
  const { t } = useTranslation()
  const createMutation = useMutation(createReplicationMutationOptions())
  const updateMutation = useMutation(updateReplicationMutationOptions())
  const registriesQuery = useQuery({
    ...registriesQueryOptions({
      page: 1,
      query: '',
    }),
    enabled: opened,
  })

  const registryOptions = useMemo(
    () => (registriesQuery.data?.registries ?? []).map(registry => ({
      value: String(registry.id),
      label: registry.name || registry.url || `#${registry.id}`,
    })),
    [registriesQuery.data],
  )

  const resourceTypeOptions = useMemo(() => replicationResourceTypeValues.map(
    value => ({
      value,
      label: value === ResourceType.RESOURCE_TYPE_ALL
        ? t('routes.admin.replications.form.resourceTypes.all')
        : value === ResourceType.RESOURCE_TYPE_MODEL
          ? t('routes.admin.replications.form.resourceTypes.model')
          : t('routes.admin.replications.form.resourceTypes.dataset'),
    }),
  ), [t])

  const triggerTypeOptions = useMemo(() => replicationTriggerTypeValues.map(
    value => ({
      value,
      label: value === TriggerType.TRIGGER_TYPE_SCHEDULED
        ? t('routes.admin.replications.trigger.scheduled')
        : t('routes.admin.replications.trigger.manual'),
    }),
  ), [t])

  const form = useForm({
    defaultValues: getFormDefaults(syncPolicy),
    validators: {
      onSubmit: createReplicationFormSchema(),
    },
    onSubmit: async ({ value }) => {
      if (mode === 'create') {
        await createMutation.mutateAsync({
          name: value.name,
          policyType: value.policyType,
          ...buildPolicyPayload(value),
        } satisfies CreateSyncPolicyRequest)
        onClose()

        return
      }

      if (syncPolicy?.id == null) {
        return
      }

      await updateMutation.mutateAsync({
        syncPolicyId: syncPolicy.id,
        ...buildPolicyPayload(value),
      } satisfies UpdateSyncPolicyRequest)
      onClose()
    },
  })

  const isSubmitting = useStore(form.store, state => state.isSubmitting)
  const isPushMode = useStore(
    form.store,
    state => state.values.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE,
  )

  const handleSubmit = () => {
    void form.handleSubmit()
  }

  return (
    <ModalWrapper
      opened={opened}
      onClose={onClose}
      size={390}
      closeOnClickOutside={false}
      title={mode === 'create'
        ? t('routes.admin.replications.createModal.title')
        : t('routes.admin.replications.editModal.title')}
      confirmLoading={isSubmitting}
      onConfirm={handleSubmit}
    >
      <Stack gap="md">
        {mode === 'create' && (
          <form.Field name="name">
            {field => (
              <TextInput
                label={t('routes.admin.replications.form.name')}
                withAsterisk
                placeholder={t('routes.admin.replications.form.namePlaceholder')}
                value={field.state.value}
                onChange={event => field.handleChange(event.currentTarget.value)}
                onBlur={field.handleBlur}
                error={fieldError(field)}
              />
            )}
          </form.Field>
        )}

        {mode === 'edit' && syncPolicy && (
          <TextInput
            label={t('routes.admin.replications.form.name')}
            disabled
            value={syncPolicy.name ?? '-'}
          />
        )}

        <form.Field name="description">
          {field => (
            <Textarea
              label={t('routes.admin.replications.form.description')}
              autosize
              minRows={2}
              maxLength={DESCRIPTION_MAX_LENGTH}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
              description={`${field.state.value?.length ?? 0}/${DESCRIPTION_MAX_LENGTH}`}
              styles={{ description: { textAlign: 'right' } }}
            />
          )}
        </form.Field>

        <form.Field name="policyType">
          {field => (
            <RadioGroup
              label={t('routes.admin.replications.form.syncRule')}
              withAsterisk
              value={String(field.state.value)}
              onChange={value => field.handleChange(value as ReplicationFormValues['policyType'])}
              onBlur={field.handleBlur}
            >
              <Group mt="xs">
                <Radio
                  value={String(SyncPolicyType.SYNC_POLICY_TYPE_PULL_BASE)}
                  label={t('routes.admin.replications.form.pull')}
                  disabled={mode === 'edit'}
                />
                <Radio
                  value={String(SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE)}
                  label={t('routes.admin.replications.form.push')}
                  disabled
                />
              </Group>
            </RadioGroup>
          )}
        </form.Field>

        <form.Field name="sourceRegistryId">
          {field => (
            <Select
              label={t('routes.admin.replications.form.sourceRegistry')}
              withAsterisk
              data={registryOptions}
              value={field.state.value ? String(field.state.value) : null}
              onChange={value => field.handleChange(value ? Number(value) : 0)}
              onBlur={field.handleBlur}
              error={fieldError(field)}
              searchable
              allowDeselect={false}
              disabled={registriesQuery.isLoading}
            />
          )}
        </form.Field>

        <form.Field name="resourceName">
          {field => (
            <TextInput
              label={t('routes.admin.replications.form.resourceFilter')}
              placeholder={t('routes.admin.replications.form.resourceName')}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
            />
          )}
        </form.Field>

        <form.Field name="resourceType">
          {field => (
            <Select
              label={t('routes.admin.replications.form.resourceTypes.label')}
              placeholder={t('routes.admin.replications.form.resourceTypes.label')}
              data={resourceTypeOptions}
              value={field.state.value}
              onChange={value => field.handleChange(value as ReplicationFormValues['resourceType'])}
              onBlur={field.handleBlur}
              error={fieldError(field)}
              allowDeselect={false}
            />
          )}
        </form.Field>

        {isPushMode && (
          <form.Field name="targetRegistryId">
            {field => (
              <Select
                label={t('routes.admin.replications.form.targetRegistry')}
                data={registryOptions}
                value={field.state.value ? String(field.state.value) : null}
                onChange={value => field.handleChange(value ? Number(value) : 0)}
                onBlur={field.handleBlur}
                error={fieldError(field)}
                searchable
                allowDeselect={false}
                disabled={registriesQuery.isLoading}
              />
            )}
          </form.Field>
        )}

        <form.Field name="targetProjectName">
          {field => (
            <TextInput
              label={t('routes.admin.replications.form.target')}
              placeholder={t('routes.admin.replications.form.targetProjectNamePlaceholder')}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
              error={fieldError(field)}
            />
          )}
        </form.Field>

        <form.Field name="triggerType">
          {field => (
            <Select
              label={t('routes.admin.replications.form.triggerType')}
              data={triggerTypeOptions}
              value={field.state.value}
              onChange={value => field.handleChange(
                (value ?? TriggerType.TRIGGER_TYPE_MANUAL) as ReplicationFormValues['triggerType'],
              )}
              onBlur={field.handleBlur}
              allowDeselect={false}
            />
          )}
        </form.Field>

        <Group gap="xs" align="flex-end">
          <form.Field name="bandwidth">
            {field => (
              <NumberInput
                style={{ flex: 1 }}
                label={t('routes.admin.replications.form.bandwidth')}
                min={BANDWIDTH_MIN}
                max={BANDWIDTH_MAX}
                value={field.state.value ? Number(field.state.value) : BANDWIDTH_MIN}
                onChange={value => field.handleChange(
                  value !== '' && value != null ? String(value) : '-1',
                )}
                onBlur={field.handleBlur}
                error={fieldError(field)}
                allowNegative
              />
            )}
          </form.Field>

          <form.Field name="bandwidthUnit">
            {field => (
              <Select
                style={{ width: 90 }}
                data={['Kbps', 'Mbps']}
                value={field.state.value}
                onChange={value => field.handleChange(
                  (value ?? 'Kbps') as ReplicationFormValues['bandwidthUnit'],
                )}
                onBlur={field.handleBlur}
                allowDeselect={false}
              />
            )}
          </form.Field>
        </Group>

        <form.Field name="isOverwrite">
          {field => (
            <Checkbox
              label={t('routes.admin.replications.form.isOverwrite')}
              checked={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.checked)}
              onBlur={field.handleBlur}
            />
          )}
        </form.Field>
      </Stack>
    </ModalWrapper>
  )
}
