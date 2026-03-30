import {
  Select,
  Stack,
  TextInput,
} from '@mantine/core'
import { useForm, useStore } from '@tanstack/react-form'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import i18n from '@/i18n'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { fieldError } from '@/shared/utils/form'

import { useProjectRoleOptions } from '../member.utils'
import { updateMemberRoleMutationOptions } from '../members.mutation'

import type { ProjectMember } from '@matrixhub/api-ts/v1alpha1/project.pb'
import type { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'

const requiredString = z.string().min(1, i18n.t('common.validation.required'))

interface EditRoleModalProps {
  opened: boolean
  onClose: () => void
  projectId: string
  member: ProjectMember | null
  currentRole?: ProjectRoleType
}

export function EditRoleModal({
  opened,
  onClose,
  projectId,
  member,
  currentRole,
}: EditRoleModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(updateMemberRoleMutationOptions())

  const form = useForm({
    defaultValues: {
      role: member?.role ?? '' as string,
    },
    onSubmit: async ({ value }) => {
      if (!member?.memberId || !member?.memberType) {
        return
      }

      await mutation.mutateAsync({
        name: projectId,
        memberType: member.memberType,
        memberId: member.memberId,
        role: value.role as ProjectRoleType,
      })
      handleClose()
    },
  })

  const roleOptions = useProjectRoleOptions(currentRole)

  const handleClose = () => {
    form.reset()
    onClose()
  }

  const canSubmit = useStore(form.store, s => s.canSubmit && !s.isSubmitting)
  const isSubmitting = useStore(form.store, s => s.isSubmitting)

  const handleConfirm = () => {
    void form.handleSubmit()
  }

  return (
    <ModalWrapper
      opened={opened}
      onClose={handleClose}
      title={t('projects.detail.membersPage.editRoleModal.title')}
      confirmLoading={isSubmitting}
      onConfirm={canSubmit ? handleConfirm : undefined}
    >
      <Stack gap="md">
        <TextInput
          label={t('projects.detail.membersPage.editRoleModal.user')}
          value={member?.memberName ?? ''}
          disabled
        />
        <form.Field
          name="role"
          validators={{ onChange: requiredString }}
        >
          {field => (
            <Select
              label={t('projects.detail.membersPage.editRoleModal.roleType')}
              withAsterisk
              allowDeselect={false}
              data={roleOptions}
              value={field.state.value || null}
              onChange={value => field.handleChange(value ?? '')}
              error={fieldError(field)}
            />
          )}
        </form.Field>
      </Stack>
    </ModalWrapper>
  )
}
