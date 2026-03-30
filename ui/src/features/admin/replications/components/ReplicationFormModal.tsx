import {
  Box,
  Checkbox,
  Flex,
  Group,
  Input,
  MultiSelect,
  NumberInput,
  Radio,
  RadioGroup,
  Select,
  Stack,
  Text,
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
import { useMemo, type ReactNode } from 'react'
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
const INLINE_FIELD_LABEL_WIDTH = 80

const inlineTextInputStyles = {
  input: {
    border: 0,
    borderRadius: 0,
    paddingInline: 'var(--mantine-spacing-md)',
    backgroundColor: 'transparent',
  },
} as const

const inlineMultiSelectStyles = {
  input: {
    border: 0,
    borderRadius: 0,
    paddingInline: 'var(--mantine-spacing-md)',
    backgroundColor: 'transparent',
  },
  section: {
    color: 'var(--mantine-color-dimmed)',
  },
} as const

export interface ReplicationFormModalProps {
  mode: 'create' | 'edit'
  opened: boolean
  syncPolicy?: SyncPolicyItem
  onClose: () => void
}

interface InlineFieldShellProps {
  label: string
  error?: ReactNode
  children: ReactNode
}

function InlineFieldShell({
  label,
  error,
  children,
}: InlineFieldShellProps) {
  return (
    <Box>
      <Flex
        style={{
          border: '1px solid var(--mantine-color-default-border)',
          borderRadius: 'var(--mantine-radius-sm)',
          overflow: 'hidden',
        }}
      >
        <Flex
          align="center"
          gap={6}
          px="md"
          bg="var(--mantine-color-gray-light)"
          fs="xs"
          p="0"
          style={{
            textAlign: 'center',
            minWidth: INLINE_FIELD_LABEL_WIDTH,
            flexShrink: 0,
            borderRight: '1px solid var(--mantine-color-default-border)',
          }}
        >
          <Text size="sm" c="dimmed">
            {label}
          </Text>
        </Flex>

        <Box
          style={{
            flex: 1,
            minWidth: 0,
          }}
        >
          {children}
        </Box>
      </Flex>

      {error && (
        <Input.Error mt={4}>
          {error}
        </Input.Error>
      )}
    </Box>
  )
}

// The backend accepts multiple resource types. Legacy ALL/empty values are
// normalized to both supported concrete types when editing existing data.
function normalizeResourceTypesForForm(
  resourceTypes?: readonly ResourceType[],
): ReplicationResourceTypeValue[] {
  const selected = (resourceTypes ?? []).filter(
    (resourceType): resourceType is ReplicationResourceTypeValue =>
      replicationResourceTypeValues.includes(resourceType as ReplicationResourceTypeValue),
  )

  return selected.length > 0
    ? selected
    : [...replicationResourceTypeValues]
}

function buildResourceTypesForApi(
  resourceTypes: ReplicationResourceTypeValue[],
): ResourceType[] {
  return resourceTypes
}

function buildPolicyPayload(values: ReplicationFormValues) {
  const bandwidth = convertBandwidthToApiValue(values.bandwidth ?? '', values.bandwidthUnit)
  const resourceTypes = buildResourceTypesForApi(values.resourceTypes)
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
    resourceTypes: normalizeResourceTypesForForm(rawResourceTypes),
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
      label: value === ResourceType.RESOURCE_TYPE_MODEL
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
      disabled: value === TriggerType.TRIGGER_TYPE_SCHEDULED,
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
        name: syncPolicy.name,
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

        <Input.Wrapper
          label={t('routes.admin.replications.form.resourceFilter')}
          labelElement="div"
        >
          <Stack gap="xs">
            <InlineFieldShell label={t('routes.admin.replications.form.resourceName')}>
              <form.Field name="resourceName">
                {field => (
                  <TextInput
                    aria-label={t('routes.admin.replications.form.resourceName')}
                    variant="unstyled"
                    placeholder={t('routes.admin.replications.form.resourceNamePlaceholder')}
                    value={field.state.value}
                    onChange={event => field.handleChange(event.currentTarget.value)}
                    onBlur={field.handleBlur}
                    styles={inlineTextInputStyles}
                  />
                )}
              </form.Field>
            </InlineFieldShell>

            <form.Field name="resourceTypes">
              {field => (
                <InlineFieldShell
                  label={t('routes.admin.replications.form.resourceTypes.label')}
                  error={fieldError(field)}
                >
                  <MultiSelect
                    aria-label={t('routes.admin.replications.form.resourceTypes.label')}
                    variant="unstyled"
                    data={resourceTypeOptions}
                    value={field.state.value}
                    onChange={value => field.handleChange(value as ReplicationFormValues['resourceTypes'])}
                    onBlur={field.handleBlur}
                    styles={inlineMultiSelectStyles}
                  />
                </InlineFieldShell>
              )}
            </form.Field>
          </Stack>
        </Input.Wrapper>

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

        <Input.Wrapper
          label={t('routes.admin.replications.form.target')}
          labelElement="div"
        >
          <form.Field name="targetProjectName">
            {field => (
              <InlineFieldShell
                label={t('routes.admin.replications.form.targetProjectName')}
                error={fieldError(field)}
              >
                <TextInput
                  aria-label={t('routes.admin.replications.form.targetProjectName')}
                  variant="unstyled"
                  placeholder={t('routes.admin.replications.form.targetProjectNamePlaceholder')}
                  value={field.state.value}
                  onChange={event => field.handleChange(event.currentTarget.value)}
                  onBlur={field.handleBlur}
                  styles={inlineTextInputStyles}
                />
              </InlineFieldShell>
            )}
          </form.Field>
        </Input.Wrapper>

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
