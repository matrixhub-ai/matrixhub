import {
  Checkbox,
  Group,
  Input,
  Select,
  Switch,
  TextInput,
} from '@mantine/core'
import { Registries } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useEffect, useEffectEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { useCurrentUser } from '@/features/auth/auth.query'
import { FieldHintLabel } from '@/shared/components/FieldHintLabel.tsx'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import { createProjectMutationOptions } from '../projects.mutation'
import {
  organizationSchema, projectNameSchema, registryIdSchema,
} from '../projects.schema'

export interface CreateProjectModalProps {
  opened: boolean
  onClose: () => void
}

export function CreateProjectModal({
  opened,
  onClose,
}: CreateProjectModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(createProjectMutationOptions())
  const { data: currentUser } = useCurrentUser()

  const form = useForm({
    defaultValues: {
      name: '',
      isPublic: false,
      enabledProxy: false,
      registryId: undefined as number | undefined,
      organization: undefined as string | undefined,
    },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync(value)
      onClose()
    },
  })

  // Reset form when modal opens
  const resetForm = useEffectEvent(() => {
    if (opened) {
      form.reset()
      mutation.reset()
    }
  })

  useEffect(() => {
    resetForm()
  }, [opened])

  // Fetch registries for the dropdown when proxy is enabled
  const registriesQuery = useQuery({
    queryKey: ['registries', 'list'],
    queryFn: () => Registries.ListRegistries({ pageSize: -1 }),
    enabled: opened,
  })

  const registryOptions = (registriesQuery.data?.registries ?? []).map(r => ({
    value: String(r.id),
    label: r.name ?? r.url ?? '',
  }))

  const handleSubmit = () => {
    void form.handleSubmit()
  }

  return (
    <ModalWrapper
      opened={opened}
      onClose={onClose}
      closeOnClickOutside={false}
      title={t('projects.createModal.title')}
      confirmLoading={mutation.isPending}
      onConfirm={handleSubmit}
    >
      <form.Field
        name="name"
        validators={{
          onChange: projectNameSchema,
        }}
      >
        {field => (
          <TextInput
            required
            label={t('projects.createModal.nameLabel')}
            value={field.state.value}
            onChange={e => field.handleChange(e.currentTarget.value)}
            onBlur={field.handleBlur}
            error={fieldError(field)}
          />
        )}
      </form.Field>

      <Input.Wrapper label={t('projects.createModal.typeLabel')}>
        <form.Field name="isPublic">
          {field => (
            <Checkbox
              mt={4}
              label={t('projects.createModal.public')}
              checked={field.state.value}
              onChange={e => field.handleChange(e.currentTarget.checked)}
            />
          )}
        </form.Field>
      </Input.Wrapper>

      {
        currentUser?.isAdmin && (
          <Input.Wrapper
            label={(
              <FieldHintLabel
                label={t('projects.createModal.proxyLabel')}
                hint={t('projects.createModal.proxyHint')}
              />
            )}
          >
            <form.Field name="enabledProxy">
              {field => (
                <Switch
                  mt={4}
                  label={field.state.value
                    ? t('projects.createModal.proxyEnabled')
                    : t('projects.createModal.proxyDisabled')}
                  checked={field.state.value}
                  onChange={(e) => {
                    field.handleChange(e.currentTarget.checked)
                    if (!e.currentTarget.checked) {
                      form.deleteField('organization')
                      form.deleteField('registryId')
                    }
                  }}
                />
              )}
            </form.Field>
            <form.Subscribe selector={s => s.values.enabledProxy}>
              {enabledProxy => enabledProxy && (
                <Group gap="xs" grow align="flex-start" mt="xs">
                  <form.Field
                    name="registryId"
                    validators={{
                      onChange: registryIdSchema,
                    }}
                  >
                    {field => (
                      <Select
                        data={registryOptions}
                        value={field.state.value != null ? String(field.state.value) : null}
                        onChange={val => field.handleChange(Number(val))}
                        onBlur={field.handleBlur}
                        error={fieldError(field)}
                      />
                    )}
                  </form.Field>

                  <form.Field
                    name="organization"
                    validators={{
                      onChange: organizationSchema,
                    }}
                  >
                    {field => (
                      <TextInput
                        placeholder={t('projects.createModal.organizationPlaceholder')}
                        value={field.state.value ?? ''}
                        onChange={e => field.handleChange(e.currentTarget.value)}
                        error={fieldError(field)}
                      />
                    )}
                  </form.Field>
                </Group>
              )}
            </form.Subscribe>
          </Input.Wrapper>
        )
      }

    </ModalWrapper>
  )
}
