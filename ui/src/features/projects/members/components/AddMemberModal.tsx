import { Select, Stack } from '@mantine/core'
import { MemberType } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { Users } from '@matrixhub/api-ts/v1alpha1/user.pb'
import { useForm, useStore } from '@tanstack/react-form'
import {
  useMutation,
  useQuery,
} from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import i18n from '@/i18n'
import { ModalWrapper } from '@/shared/components/ModalWrapper'
import { fieldError } from '@/shared/utils/form'

import { useProjectRoleOptions } from '../member.utils'
import { addMemberMutationOptions } from '../members.mutation'

import type { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'

interface AddMemberModalProps {
  opened: boolean
  onClose: () => void
  projectId: string
}

const requiredString = z.string().min(1, i18n.t('common.validation.required'))

const defaultValues = {
  memberType: MemberType.MEMBER_TYPE_USER as string,
  memberId: '' as string,
  role: '' as string,
}

export function AddMemberModal({
  opened,
  onClose,
  projectId,
}: AddMemberModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(addMemberMutationOptions())

  const form = useForm({
    defaultValues,
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync({
        name: projectId,
        memberType: value.memberType as MemberType,
        memberId: Number(value.memberId),
        role: value.role as ProjectRoleType,
      })
      handleClose()
    },
  })

  const memberTypeOptions = [
    {
      value: MemberType.MEMBER_TYPE_USER,
      label: t('projects.detail.membersPage.memberType.user'),
    },
    {
      value: MemberType.MEMBER_TYPE_GROUP,
      disabled: true, // Group member type is not supported yet
      label: t('projects.detail.membersPage.memberType.group'),
    },
  ]

  const roleOptions = useProjectRoleOptions()

  const memberType = useStore(form.store, s => s.values.memberType)

  const { data: usersData } = useQuery({
    queryKey: ['users', 'list'],
    queryFn: () => Users.ListUsers({
      page: 1,
      pageSize: -1,
    }),
    enabled: opened && memberType === MemberType.MEMBER_TYPE_USER,
  })

  const userOptions = (usersData?.users ?? []).map(u => ({
    value: String(u.id ?? ''),
    label: u.username ?? '',
  }))

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
      title={t('projects.detail.membersPage.addMemberModal.title')}
      confirmLoading={isSubmitting}
      onConfirm={canSubmit ? handleConfirm : undefined}
    >
      <Stack gap="md">
        <form.Field
          name="memberType"
          validators={{ onChange: requiredString }}
        >
          {field => (
            <Select
              label={t('projects.detail.membersPage.addMemberModal.memberType')}
              withAsterisk
              data={memberTypeOptions}
              value={field.state.value}
              onChange={(value) => {
                field.handleChange(value ?? '')
                form.setFieldValue('memberId', '')
              }}
              error={fieldError(field)}
            />
          )}
        </form.Field>
        <form.Field
          name="memberId"
          validators={{ onChange: requiredString }}
        >
          {field => (
            <Select
              label={t('projects.detail.membersPage.addMemberModal.user')}
              withAsterisk
              data={userOptions}
              value={field.state.value || null}
              onChange={value => field.handleChange(value ?? '')}
              searchable
              disabled={memberType === MemberType.MEMBER_TYPE_GROUP}
              error={fieldError(field)}
            />
          )}
        </form.Field>
        <form.Field
          name="role"
          validators={{ onChange: requiredString }}
        >
          {field => (
            <Select
              label={t('projects.detail.membersPage.addMemberModal.roleType')}
              withAsterisk
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
