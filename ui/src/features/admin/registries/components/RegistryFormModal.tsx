import {
  Button,
  Checkbox,
  Group,
  PasswordInput,
  Select,
  Stack,
  TextInput,
  Textarea,
} from '@mantine/core'
import { RegistryType } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import { useStore } from '@tanstack/react-form'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { FieldHintLabel } from '@/shared/components/FieldHintLabel'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import {
  buildCreateRegistryRequest,
  buildPingRegistryRequest,
  buildUpdateRegistryRequest,
  createRegistryFormSchema,
  editRegistryFormSchema,
  getRegistryFormValues,
  REGISTRY_DESCRIPTION_MAX_LENGTH,
  registryDescriptionSchema,
  registryNameSchema,
  registryUrlSchema,
  sanitizeRegistryName,
} from '../registries.form'
import {
  createRegistryMutationOptions,
  pingRegistryMutationOptions,
  updateRegistryMutationOptions,
} from '../registries.mutation'

import type { Registry } from '@matrixhub/api-ts/v1alpha1/registry.pb'

export interface RegistryFormModalProps {
  mode: 'create' | 'edit'
  opened: boolean
  registry?: Registry
  onClose: () => void
}

export function RegistryFormModal({
  mode,
  opened,
  registry,
  onClose,
}: RegistryFormModalProps) {
  const { t } = useTranslation()
  const submitMutation = useMutation(
    mode === 'create'
      ? createRegistryMutationOptions()
      : updateRegistryMutationOptions(),
  )
  const pingMutation = useMutation(pingRegistryMutationOptions())
  const form = useForm({
    defaultValues: getRegistryFormValues(registry),
    validators: {
      onSubmit: mode === 'create'
        ? createRegistryFormSchema
        : editRegistryFormSchema,
    },
    onSubmit: async ({ value }) => {
      if (mode === 'create') {
        await submitMutation.mutateAsync(buildCreateRegistryRequest(value))
      } else if (registry?.id != null) {
        await submitMutation.mutateAsync(buildUpdateRegistryRequest(registry.id, value))
      }

      onClose()
    },
  })

  const providerOptions = [
    {
      value: RegistryType.REGISTRY_TYPE_HUGGINGFACE,
      label: t('routes.admin.registries.provider.huggingFace'),
    },
  ]
  const isSubmitting = useStore(form.store, state => state.isSubmitting)

  const handleTestConnection = async () => {
    await pingMutation.mutateAsync(buildPingRegistryRequest(form.state.values))
  }

  return (
    <ModalWrapper
      opened={opened}
      onClose={onClose}
      size="sm"
      closeOnClickOutside={false}
      title={mode === 'create'
        ? t('routes.admin.registries.createModal.title')
        : t('routes.admin.registries.editModal.title')}
      footer={(
        <Group justify="flex-end" gap="md">
          <Button
            variant="white"
            color="gray"
            onClick={onClose}
          >
            {t('common.cancel')}
          </Button>
          <form.Subscribe
            selector={state => [
              state.values.url.trim().length > 0,
              state.fieldMeta.url?.isValid ?? false,
            ]}
          >
            {([hasRegistryUrl, isRegistryUrlValid]) => (
              <Button
                variant="outline"
                loading={pingMutation.isPending}
                disabled={
                  !hasRegistryUrl
                  || !isRegistryUrlValid
                  || submitMutation.isPending
                  || isSubmitting
                }
                onClick={() => {
                  void handleTestConnection()
                }}
              >
                {t('routes.admin.registries.form.testConnection')}
              </Button>
            )}
          </form.Subscribe>
          <Button
            loading={submitMutation.isPending || isSubmitting}
            disabled={pingMutation.isPending}
            onClick={() => {
              void form.handleSubmit()
            }}
          >
            {t('common.confirm')}
          </Button>
        </Group>
      )}
    >
      <Stack gap="md">
        <form.Field name="type">
          {field => (
            <Select
              label={t('routes.admin.registries.form.provider')}
              withAsterisk
              data={providerOptions}
              value={field.state.value}
              onChange={value => field.handleChange(
                (value as RegistryType | null) ?? RegistryType.REGISTRY_TYPE_HUGGINGFACE,
              )}
              onBlur={field.handleBlur}
              disabled={mode === 'edit'}
              allowDeselect={false}
            />
          )}
        </form.Field>

        <form.Field
          name="name"
          validators={{ onChange: registryNameSchema }}
        >
          {field => (
            <TextInput
              label={t('routes.admin.registries.form.name')}
              withAsterisk
              value={field.state.value}
              onChange={event => field.handleChange(
                sanitizeRegistryName(event.currentTarget.value),
              )}
              onBlur={field.handleBlur}
              error={fieldError(field)}
            />
          )}
        </form.Field>

        <form.Field
          name="description"
          validators={{ onChange: registryDescriptionSchema }}
        >
          {field => (
            <Textarea
              label={t('routes.admin.registries.form.description')}
              autosize
              minRows={3}
              maxLength={REGISTRY_DESCRIPTION_MAX_LENGTH}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
              error={fieldError(field)}
              description={`${field.state.value.length}/${REGISTRY_DESCRIPTION_MAX_LENGTH}`}
            />
          )}
        </form.Field>

        <form.Field
          name="url"
          validators={{ onChange: registryUrlSchema }}
        >
          {field => (
            <TextInput
              label={t('routes.admin.registries.form.url')}
              withAsterisk
              placeholder={t('routes.admin.registries.form.urlPlaceholder')}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
              error={fieldError(field)}
            />
          )}
        </form.Field>

        <form.Field name="username">
          {field => (
            <TextInput
              label={(
                <FieldHintLabel
                  label={t('routes.admin.registries.form.username')}
                  hint={t('routes.admin.registries.form.usernameHint')}
                />
              )}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
            />
          )}
        </form.Field>

        <form.Field name="password">
          {field => (
            <PasswordInput
              label={t('routes.admin.registries.form.password')}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
              error={fieldError(field)}
            />
          )}
        </form.Field>

        <form.Field name="verifyRemoteCert">
          {field => (
            <Checkbox
              label={(
                <FieldHintLabel
                  label={t('routes.admin.registries.form.verifyRemoteCert')}
                  hint={t('routes.admin.registries.form.verifyRemoteCertHint')}
                />
              )}
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
