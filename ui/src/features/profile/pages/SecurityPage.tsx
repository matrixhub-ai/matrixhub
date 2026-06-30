import {
  Button,
  PasswordInput,
  Stack,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { resetPasswordMutationOptions } from '@/features/profile/profile.mutation'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

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
    ...resetPasswordMutationOptions(),
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
      resetPassword({
        oldPassword: value.oldPassword,
        newPassword: value.newPassword,
      })
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
            {field => (
              <PasswordInput
                label={t('profile.oldPassword')}
                required
                value={field.state.value}
                onChange={e => field.handleChange(e.currentTarget.value)}
                onBlur={field.handleBlur}
                error={fieldError(field)}
              />
            )}
          </Field>
          <Field
            name="newPassword"
            validators={{ onChange: fieldSchemas.newPassword }}
          >
            {field => (
              <PasswordInput
                label={t('profile.newPassword')}
                required
                value={field.state.value}
                onChange={e => field.handleChange(e.currentTarget.value)}
                onBlur={field.handleBlur}
                error={fieldError(field)}
              />
            )}
          </Field>
          <Field
            name="confirmNewPassword"
            validators={{ onChange: fieldSchemas.confirmNewPassword }}
          >
            {field => (
              <PasswordInput
                label={t('profile.confirmNewPassword')}
                required
                value={field.state.value}
                onChange={e => field.handleChange(e.currentTarget.value)}
                onBlur={field.handleBlur}
                error={fieldError(field)}
              />
            )}
          </Field>
        </Stack>
      </ModalWrapper>
    </Stack>
  )
}
