import {
  Button,
  PasswordInput,
  Stack,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { CurrentUser } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { useForm } from '@/shared/hooks/useForm'

interface FormValues {
  oldPassword: string
  newPassword: string
  confirmNewPassword: string
}

const initialForm: FormValues = {
  oldPassword: '',
  newPassword: '',
  confirmNewPassword: '',
}

export function SecurityPage() {
  const { t } = useTranslation()
  const [opened, {
    open, close,
  }] = useDisclosure(false)

  const fieldSchemas = {
    oldPassword: z.string().min(1, { error: t('common.validation.fieldRequired', { field: t('profile.oldPassword') }) }),
    newPassword: z.string()
      .min(8, { error: t('common.validation.minLength', {
        field: t('profile.newPassword'),
        min: 8,
      }) })
      .max(20, { error: t('common.validation.maxLength', {
        field: t('profile.newPassword'),
        max: 20,
      }) })
      .regex(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).+$/, { message: t('profile.passwordComplexity') }),

    confirmNewPassword: z.string().min(1, { error: t('common.validation.fieldRequired', { field: t('profile.confirmNewPassword') }) }),
  }

  const formSchema = z.object(fieldSchemas).refine(
    data => data.newPassword === data.confirmNewPassword,
    {
      error: t('profile.passwordMismatch'),
      path: ['confirmNewPassword'],
    },
  )

  const {
    mutate: resetPassword, isPending,
  } = useMutation({
    mutationFn: (value: FormValues) =>
      CurrentUser.ResetPassword({
        oldPassword: value.oldPassword,
        newPassword: value.newPassword,
      }),
    meta: {
      successMessage: t('profile.passwordChanged'),
    },
    onSuccess: () => {
      handleClose()
    },
  })

  const {
    reset, handleSubmit, Field,
  } = useForm({
    defaultValues: initialForm,
    validators: {
      onSubmit: formSchema,
    },
    onSubmit: ({ value }) => {
      resetPassword(value)
    },
  })

  const handleClose = () => {
    close()
    reset()
  }

  return (
    <Stack
      gap="md"
      align="flex-start"
    >
      <PasswordInput
        label={t('profile.password')}
        w={200}
        readOnly
        disabled
        value="********"
      />

      <Button onClick={open}>
        {t('profile.changePassword')}
      </Button>

      <ModalWrapper
        title={t('profile.editPassword')}
        size="sm"
        opened={opened}
        onClose={handleClose}
        onConfirm={() => handleSubmit()}
        confirmLoading={isPending}
      >
        <Stack gap="md">
          <Field
            name="oldPassword"
            validators={{ onChange: fieldSchemas.oldPassword }}
          >
            {({
              state, handleChange,
            }) => (
              <PasswordInput
                label={t('profile.oldPassword')}
                required
                value={state.value}
                onChange={e => handleChange(e.currentTarget.value)}
                error={state.meta.errors[0]?.message}
              />
            )}
          </Field>
          <Field
            name="newPassword"
            validators={{ onChange: fieldSchemas.newPassword }}
          >
            {({
              state, handleChange,
            }) => (
              <PasswordInput
                label={t('profile.newPassword')}
                required
                value={state.value}
                onChange={e => handleChange(e.currentTarget.value)}
                error={state.meta.errors[0]?.message}
              />
            )}
          </Field>
          <Field
            name="confirmNewPassword"
            validators={{ onChange: fieldSchemas.confirmNewPassword }}
          >
            {({
              state, handleChange,
            }) => (
              <PasswordInput
                label={t('profile.confirmNewPassword')}
                required
                value={state.value}
                onChange={e => handleChange(e.currentTarget.value)}
                error={state.meta.errors[0]?.message}
              />
            )}
          </Field>
        </Stack>
      </ModalWrapper>
    </Stack>
  )
}
