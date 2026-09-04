import {
  ActionIcon,
  Checkbox,
  Group,
  Input,
  Select,
  Stack,
  Switch,
  Text,
  TextInput,
  Tooltip,
} from '@mantine/core'
import {
  IconExternalLink, IconInfoCircle, IconPlus, IconRefresh,
} from '@tabler/icons-react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useEffect, useEffectEvent } from 'react'
import { Trans, useTranslation } from 'react-i18next'

import { allRegistriesQueryOptions } from '@/features/admin/registries/registries.query'
import { useCurrentUser } from '@/features/auth/auth.query'
import AnchorLink from '@/shared/components/AnchorLink'
import { FieldHintLabel } from '@/shared/components/FieldHintLabel.tsx'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import { createProjectMutationOptions } from '../projects.mutation'
import {
  organizationSchema, projectNameSchema, registryIdSchema,
} from '../projects.schema'

import type { MantineSize } from '@mantine/core'
import type { ReactNode } from 'react'

export interface CreateProjectModalProps {
  opened: boolean
  onClose: () => void
  onCreated?: (projectName: string) => void | Promise<void>
}

/**
 * Link to registry creation. Opened in a new tab so the half-filled project
 * form stays untouched, with the usual icon marking the tab switch.
 */
function RegistryCreateLink({
  children,
  size,
}: {
  children?: ReactNode
  size?: MantineSize
}) {
  return (
    <AnchorLink
      to="/admin/registries"
      search={{ create: true }}
      target="_blank"
      size={size}
      inherit={!size}
    >
      <Group component="span" gap={2} align="center" wrap="nowrap" display="inline-flex">
        {children}
        <IconExternalLink size={14} />
      </Group>
    </AnchorLink>
  )
}

export function CreateProjectModal({
  opened,
  onClose,
  onCreated,
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
      await onCreated?.(value.name)
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
    ...allRegistriesQueryOptions(),
    enabled: opened,
  })

  const registryOptions = (registriesQuery.data ?? []).map(r => ({
    value: String(r.id),
    label: r.name ?? r.url ?? '',
  }))

  // Both states are decided only once the request has succeeded, so loading
  // and error states show neither the guidance nor the shortcut, and the row
  // does not shift as the request settles.
  const showEmptyRegistryHint
    = registriesQuery.isSuccess && registryOptions.length === 0
  const showCreateRegistryShortcut
    = registriesQuery.isSuccess && registryOptions.length > 0

  // Opened in a new tab so the half-filled project form stays untouched here.
  const registryLink = <RegistryCreateLink />

  // Sits next to whichever registry action is currently shown: the create
  // shortcut when registries exist, the guidance line when none do.
  const refreshRegistriesButton = (
    <Tooltip label={t('projects.createModal.refreshRegistries')}>
      <ActionIcon
        variant="subtle"
        color="gray"
        aria-label={t('projects.createModal.refreshRegistries')}
        loading={registriesQuery.isFetching}
        onClick={() => void registriesQuery.refetch()}
      >
        <IconRefresh size={16} />
      </ActionIcon>
    </Tooltip>
  )

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
            <Group justify="space-between" align="center" wrap="nowrap" mt={4}>
              <form.Field name="enabledProxy">
                {field => (
                  <Switch
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
                {enabledProxy => enabledProxy && !showEmptyRegistryHint && (
                  <Group gap="xs" align="center" wrap="nowrap">
                    {showCreateRegistryShortcut && (
                      <RegistryCreateLink size="sm">
                        <IconPlus size={14} />
                        {t('projects.createModal.createRegistry')}
                      </RegistryCreateLink>
                    )}
                    {refreshRegistriesButton}
                  </Group>
                )}
              </form.Subscribe>
            </Group>
            <form.Subscribe selector={s => s.values.enabledProxy}>
              {enabledProxy => enabledProxy && (
                <Stack gap="xs" mt="xs">
                  {showEmptyRegistryHint && (
                    <Group gap={4} align="center" wrap="nowrap">
                      <IconInfoCircle
                        size={16}
                        color="var(--mantine-primary-color-filled)"
                      />
                      <Text size="sm" c="dimmed">
                        <Trans
                          i18nKey="projects.createModal.emptyRegistryHint"
                          components={[registryLink]}
                        />
                      </Text>
                      {refreshRegistriesButton}
                    </Group>
                  )}

                  <Group gap="xs" align="flex-start" wrap="nowrap">
                    <form.Field
                      name="registryId"
                      validators={{
                        onChange: registryIdSchema,
                      }}
                    >
                      {field => (
                        <Select
                          flex={1}
                          data={registryOptions}
                          placeholder={t('projects.createModal.registryPlaceholder')}
                          nothingFoundMessage={t('projects.createModal.registryEmptyOption')}
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
                          flex={1}
                          placeholder={t('projects.createModal.organizationPlaceholder')}
                          value={field.state.value ?? ''}
                          onChange={e => field.handleChange(e.currentTarget.value)}
                          error={fieldError(field)}
                        />
                      )}
                    </form.Field>
                  </Group>
                </Stack>
              )}
            </form.Subscribe>
          </Input.Wrapper>
        )
      }

    </ModalWrapper>
  )
}
