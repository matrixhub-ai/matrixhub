import {
  Button,
  Checkbox,
  Group,
  PasswordInput,
  Select,
  Stack,
  Text,
  TextInput,
  Textarea,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { RegistryType } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import { useStore } from '@tanstack/react-form'
import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
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
  getRegistryDefaultUrl,
  getRegistryFormValues,
  getRegistryUrlCopyKeys,
  isRegistryUrlPristine,
  isLikelyRepositoryUrl,
  REGISTRY_DESCRIPTION_MAX_LENGTH,
  registryProviderLabelKeys,
  registryDescriptionSchema,
  registryNameSchema,
  registryUrlSchema,
  sanitizeRegistryName,
  supportedRegistryTypes,
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

  const providerOptions = supportedRegistryTypes.map((type) => {
    const labelKey = registryProviderLabelKeys[type]

    return {
      value: type,
      label: labelKey ? t(labelKey) : type,
    }
  })
  const isSubmitting = useStore(form.store, state => state.isSubmitting)
  const [pendingType, setPendingType] = useState<RegistryType | null>(null)
  const [
    switchConfirmOpened,
    {
      open: openSwitchConfirm,
      close: closeSwitchConfirm,
    },
  ] = useDisclosure(false)

  const applyProviderChange = (nextType: RegistryType, nextUrl: string) => {
    form.setFieldValue('type', nextType)
    form.setFieldValue('url', nextUrl)

    /* Switching providers rewrites the URL on the user's behalf, so the field
    starts over as untouched instead of complaining about the value it just got. */
    form.setFieldMeta('url', meta => ({
      ...meta,
      isTouched: false,
      isDirty: false,
      errors: [],
      errorMap: {},
    }))
  }

  const handleProviderChange = (nextType: RegistryType) => {
    const currentValues = form.state.values

    if (nextType === currentValues.type) {
      return
    }

    /* Only a user-customized URL is worth a confirmation prompt;
    an untouched one is replaced with the new provider default silently. */
    if (isRegistryUrlPristine(currentValues.url, currentValues.type)) {
      applyProviderChange(nextType, getRegistryDefaultUrl(nextType))

      return
    }

    setPendingType(nextType)
    openSwitchConfirm()
  }

  const closeProviderSwitchConfirm = () => {
    closeSwitchConfirm()
    setPendingType(null)
  }

  const handleKeepUrlAndSwitch = () => {
    if (pendingType != null) {
      applyProviderChange(pendingType, form.state.values.url)
    }

    closeProviderSwitchConfirm()
  }

  const handleClearUrlAndSwitch = () => {
    if (pendingType != null) {
      applyProviderChange(pendingType, getRegistryDefaultUrl(pendingType))
    }

    closeProviderSwitchConfirm()
  }

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
              onChange={value => handleProviderChange(
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

        <form.Subscribe selector={state => state.values.type}>
          {(type) => {
            const urlCopyKeys = getRegistryUrlCopyKeys(type)

            return (
              <form.Field
                name="url"
                validators={{ onChange: registryUrlSchema }}
              >
                {field => (
                  <TextInput
                    label={(
                      <FieldHintLabel
                        label={t('routes.admin.registries.form.url')}
                        hint={t(urlCopyKeys.hint)}
                        tooltipProps={{ style: { whiteSpace: 'pre-line' } }}
                      />
                    )}
                    withAsterisk
                    placeholder={t(urlCopyKeys.placeholder)}
                    description={(
                      <>
                        {t(urlCopyKeys.description)}
                        {isLikelyRepositoryUrl(field.state.value) && (
                          <Text
                            component="span"
                            display="block"
                            inherit
                            c="yellow.7"
                            mt={4}
                          >
                            {t(urlCopyKeys.repositoryUrlWarning)}
                          </Text>
                        )}
                      </>
                    )}
                    inputWrapperOrder={['label', 'input', 'description', 'error']}
                    styles={{ error: { marginTop: 4 } }}
                    value={field.state.value}
                    onChange={event => field.handleChange(event.currentTarget.value)}
                    onBlur={field.handleBlur}
                    error={fieldError(field)}
                  />
                )}
              </form.Field>
            )
          }}
        </form.Subscribe>

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

      {switchConfirmOpened && (
        <ModalWrapper
          opened
          onClose={closeProviderSwitchConfirm}
          type="info"
          size="md"
          title={t('routes.admin.registries.switchProviderModal.title')}
          footer={(
            <Group justify="flex-end" gap="md" wrap="nowrap">
              <Button
                color="default"
                variant="subtle"
                fw={400}
                onClick={closeProviderSwitchConfirm}
              >
                {t('common.cancel')}
              </Button>
              <Button
                variant="outline"
                fw={400}
                onClick={handleClearUrlAndSwitch}
              >
                {t('routes.admin.registries.switchProviderModal.clearAndSwitch')}
              </Button>
              <Button
                fw={400}
                onClick={handleKeepUrlAndSwitch}
              >
                {t('routes.admin.registries.switchProviderModal.keepAndSwitch')}
              </Button>
            </Group>
          )}
        >
          <Text size="sm">
            {t('routes.admin.registries.switchProviderModal.description')}
          </Text>
        </ModalWrapper>
      )}
    </ModalWrapper>
  )
}
