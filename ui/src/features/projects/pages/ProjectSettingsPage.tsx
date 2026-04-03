import {
  Button, Checkbox, Group, InputWrapper, rem, Stack, TextInput,
} from '@mantine/core'
import { ProjectType } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { useMutation, useSuspenseQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useForm } from '@/shared/hooks/useForm'

import { updateProjectMutationOptions } from '../projects.mutation'
import { projectDetailQueryOptions } from '../projects.query'

const projectRouteApi = getRouteApi('/(auth)/(app)/projects/$projectId')

export function ProjectSettingsPage() {
  const { t } = useTranslation()
  const { projectId } = projectRouteApi.useParams()
  const { data: project } = useSuspenseQuery(projectDetailQueryOptions(projectId))
  const mutation = useMutation(updateProjectMutationOptions())

  const isProxy = !!project.registryUrl

  const projectSettingForm = useForm({
    defaultValues: {
      type: project.type ?? ProjectType.PROJECT_TYPE_PRIVATE,
    },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync({
        name: projectId,
        type: value.type,
      })
    },
  })

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        projectSettingForm.handleSubmit()
      }}
    >
      <Stack pt="lg" gap="md" align="flex-start">
        <Stack gap="xs">
          <projectSettingForm.Field name="type">
            {field => (
              <InputWrapper label={t('projects.detail.settingsPage.projectType')}>
                <Checkbox
                  checked={field.state.value === ProjectType.PROJECT_TYPE_PUBLIC}
                  onChange={(event) => {
                    field.setValue(event.currentTarget.checked
                      ? ProjectType.PROJECT_TYPE_PUBLIC
                      : ProjectType.PROJECT_TYPE_PRIVATE)
                  }}
                  label={t('projects.type.public')}
                />
              </InputWrapper>
            )}
          </projectSettingForm.Field>
        </Stack>

        {isProxy && (
          <>
            <TextInput
              label={t('projects.detail.settingsPage.proxyAddress')}
              value={project.registryUrl ?? ''}
              w={rem('260px')}
              disabled
            />
            <TextInput
              label={t('projects.detail.settingsPage.proxyOrganization')}
              value={project.organization ?? ''}
              w={rem('260px')}
              disabled
            />
          </>
        )}

        <Group gap="md">
          <projectSettingForm.Subscribe selector={s => [s.isSubmitting]}>
            {([isSubmitting]) => (
              <>
                <Button
                  type="submit"
                  loading={isSubmitting}
                >
                  {t('projects.detail.settingsPage.save')}
                </Button>
                <Button
                  variant="default"
                  onClick={() => projectSettingForm.reset()}
                >
                  {t('projects.detail.settingsPage.cancel')}
                </Button>
              </>
            )}
          </projectSettingForm.Subscribe>
        </Group>
      </Stack>
    </form>
  )
}
