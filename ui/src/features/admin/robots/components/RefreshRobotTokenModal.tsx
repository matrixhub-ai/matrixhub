import {
  Alert,
  PasswordInput,
  Stack,
  Switch,
  rem,
} from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useEffectEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { ModalWrapper } from '@/shared/components/ModalWrapper.tsx'
import { useForm } from '@/shared/hooks/useForm.ts'
import { fieldError } from '@/shared/utils/form.ts'

import { refreshRobotAccountTokenMutationOptions } from '../robot.mutation.ts'
import {
  refreshRobotTokenFormDefaults,
  refreshRobotTokenSchema,
  type RefreshRobotTokenFormValues,
} from '../robot.schema.ts'

import type { RobotAccount } from '../robot.utils.ts'
import type { StandardSchemaV1Issue } from '@tanstack/react-form'

interface RefreshRobotTokenModalProps {
  robot: RobotAccount
  opened: boolean
  onClose: () => void
  onSuccess: (token: string) => void
}

function getRefreshRobotTokenFieldIssues(
  fieldName: keyof Pick<RefreshRobotTokenFormValues, 'token' | 'confirmToken'>,
  values: RefreshRobotTokenFormValues,
) {
  const result = refreshRobotTokenSchema.safeParse(values)

  if (result.success) {
    return undefined
  }

  const issues = result.error.issues.filter(issue => issue.path[0] === fieldName)

  return issues.length > 0
    ? issues as StandardSchemaV1Issue[]
    : undefined
}

export function RefreshRobotTokenModal({
  opened,
  robot,
  onClose,
  onSuccess,
}: RefreshRobotTokenModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(refreshRobotAccountTokenMutationOptions())
  const form = useForm({
    defaultValues: { ...refreshRobotTokenFormDefaults },
    validators: {
      onSubmit: refreshRobotTokenSchema,
    },
    onSubmit: async ({ value }) => {
      const response = await mutation.mutateAsync({
        id: robot.id,
        autoGenerate: value.autoGenerate,
        token: value.autoGenerate ? undefined : value.token.trim(),
      })

      onClose()
      onSuccess(response.token ?? '')
    },
  })

  const resetForm = useEffectEvent(() => {
    if (opened) {
      form.reset()
      mutation.reset()
    }
  })

  useEffect(() => {
    resetForm()
  }, [opened])

  const handleAutoGenerateChange = (
    autoGenerate: boolean,
    onChange: (value: boolean) => void,
  ) => {
    onChange(autoGenerate)

    if (autoGenerate) {
      form.setFieldValue('token', refreshRobotTokenFormDefaults.token)
      form.setFieldValue('confirmToken', refreshRobotTokenFormDefaults.confirmToken)
    }
  }

  return (
    <ModalWrapper
      size="md"
      title={t('routes.admin.robots.refreshTokenModal.title', {
        name: robot.name,
      })}
      closeOnClickOutside={false}
      opened={opened}
      onClose={onClose}
      confirmLoading={mutation.isPending || form.state.isSubmitting}
      onConfirm={form.handleSubmit}
    >
      <Stack gap="md">
        <Alert
          lh={rem(20)}
          icon={<IconInfoCircle size={20} />}
        >
          {t('routes.admin.robots.refreshTokenModal.description')}
        </Alert>

        <form.Field name="autoGenerate">
          {field => (
            <Switch
              size="sm"
              py={rem(6)}
              label={t('routes.admin.robots.refreshTokenModal.manualSwitch')}
              checked={!field.state.value}
              onChange={event => handleAutoGenerateChange(
                !event.currentTarget.checked,
                field.handleChange,
              )}
            />
          )}
        </form.Field>

        <form.Subscribe selector={state => state.values.autoGenerate}>
          {autoGenerate => !autoGenerate && (
            <>
              <form.Field
                name="token"
                validators={{
                  onChange: ({
                    value,
                    fieldApi,
                  }) => getRefreshRobotTokenFieldIssues(
                    'token',
                    {
                      ...fieldApi.form.state.values,
                      token: value,
                    },
                  ),
                }}
              >
                {field => (
                  <PasswordInput
                    required
                    autoComplete="new-password"
                    label={t('routes.admin.robots.refreshTokenModal.token')}
                    value={field.state.value ?? ''}
                    onChange={event => field.handleChange(event.currentTarget.value)}
                    onBlur={field.handleBlur}
                    error={fieldError(field)}
                  />
                )}
              </form.Field>

              <form.Field
                name="confirmToken"
                validators={{
                  onChange: ({
                    value,
                    fieldApi,
                  }) => getRefreshRobotTokenFieldIssues(
                    'confirmToken',
                    {
                      ...fieldApi.form.state.values,
                      confirmToken: value,
                    },
                  ),
                  onChangeListenTo: ['token'],
                }}
              >
                {field => (
                  <PasswordInput
                    required
                    autoComplete="new-password"
                    label={t('routes.admin.robots.refreshTokenModal.confirmToken')}
                    value={field.state.value ?? ''}
                    onChange={event => field.handleChange(event.currentTarget.value)}
                    onBlur={field.handleBlur}
                    error={fieldError(field)}
                  />
                )}
              </form.Field>
            </>
          )}
        </form.Subscribe>
      </Stack>
    </ModalWrapper>
  )
}
