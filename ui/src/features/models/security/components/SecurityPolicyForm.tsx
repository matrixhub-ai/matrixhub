import {
  Alert, Button, Select, Stack,
} from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { type ScanPolicy } from '@/features/models/security/security.api'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import { securityPolicyMutation } from '../security.mutation'
import { securityPolicySchema } from '../security.schema'

export function SecurityPolicyForm({
  project, model, policy, writable,
}: { project: string
  model: string
  policy: ScanPolicy
  writable: boolean }) {
  const { t } = useTranslation()
  const mutation = useMutation(securityPolicyMutation(project, model))
  const form = useForm({
    defaultValues: policy,
    validators: { onChange: securityPolicySchema },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync(value)
    },
  })
  const actionOptions = [{
    value: 'block',
    label: t('security.block'),
  }, {
    value: 'allow',
    label: t('security.allow'),
  }]

  return (
    <Stack maw={480} gap="md">
      <Alert color="yellow">{t('security.policyHint')}</Alert>
      <form.Field name="block_severity">
        {field => (
          <Select
            label={t('security.threshold')}
            data={[{
              value: 'medium',
              label: t('security.medium'),
            }, {
              value: 'high',
              label: t('security.high'),
            }]}
            value={field.state.value}
            onChange={value => field.handleChange(value === 'high' ? 'high' : 'medium')}
            onBlur={field.handleBlur}
            error={fieldError(field)}
            disabled={!writable}
          />
        )}
      </form.Field>
      <form.Field name="on_pending">{field => <Select label={t('security.onPending')} data={actionOptions} value={field.state.value} onChange={value => field.handleChange(value === 'allow' ? 'allow' : 'block')} onBlur={field.handleBlur} error={fieldError(field)} disabled={!writable} />}</form.Field>
      <form.Field name="on_failure">{field => <Select label={t('security.onFailure')} data={actionOptions} value={field.state.value} onChange={value => field.handleChange(value === 'allow' ? 'allow' : 'block')} onBlur={field.handleBlur} error={fieldError(field)} disabled={!writable} />}</form.Field>
      {writable && <form.Subscribe selector={state => [state.canSubmit, state.isSubmitting]}>{([canSubmit, submitting]) => <Button disabled={!canSubmit} loading={submitting} onClick={() => void form.handleSubmit()}>{t('security.savePolicy')}</Button>}</form.Subscribe>}
    </Stack>
  )
}
