import {
  Alert,
  Button, Group,
  rem,
  Stack,
  Text,
  TextInput,
} from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import { useMutation } from '@tanstack/react-query'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { createModelMutationOptions } from '@/features/models/models.mutation'
import { useModelProjects } from '@/features/models/models.query.ts'
import { createModelSchema } from '@/features/models/models.schema'
import { ProjectSelect } from '@/shared/components/ProjectSelect'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form.ts'

interface ModelCreatePageProps {
  initialProjectId?: string
}

export function ModelCreatePage({ initialProjectId = '' }: ModelCreatePageProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const router = useRouter()

  const mutation = useMutation(createModelMutationOptions())
  const modelCreateSchema = createModelSchema(t)
  const {
    name: nameValidator, projectId: projectIdValidator,
  } = modelCreateSchema.shape

  const handleNavigateBack = () => {
    if (initialProjectId) {
      return navigate({
        to: '/projects/$projectId/models',
        params: {
          projectId: initialProjectId,
        },
      })
    }

    if (router.history.length > 1) {
      return router.history.back()
    }

    return navigate({ to: '/models' })
  }

  const form = useForm({
    defaultValues: {
      name: '',
      projectId: initialProjectId?.trim(),
    },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync({
        name: value.name,
        project: value.projectId,
      })

      handleNavigateBack()
    },
  })

  const { data: projects = [] } = useModelProjects()

  useEffect(() => {
    const projectId = form.state.values.projectId

    if (projects.length && (!projectId || !projects?.find(option => option.name === projectId))) {
      form.setFieldValue('projectId', projects[0]?.name ?? '')
    }
  }, [projects, form])

  return (
    <Stack
      w={350}
      mx="auto"
      py="lg"
      gap="lg"
    >
      <Text fw={600} lh={rem(24)} size="md">
        {t('model.new')}
      </Text>

      <form
        onSubmit={(event) => {
          event.preventDefault()
          void form.handleSubmit()
        }}
      >
        <Stack gap="md">
          <form.Field
            name="name"
            validators={{ onChange: nameValidator }}
          >
            {field => (
              <TextInput
                label={t('model.create.modelName')}
                withAsterisk
                placeholder={t('model.create.modelNamePlaceholder')}
                description={t('model.create.modelNameHelper')}
                value={field.state.value}
                onBlur={field.handleBlur}
                onChange={e => field.handleChange(e.currentTarget.value)}
                error={fieldError(field)}
              />
            )}
          </form.Field>

          <form.Field
            name="projectId"
            validators={{ onChange: projectIdValidator }}
          >
            {field => (
              <ProjectSelect
                data={projects}
                value={field.state.value}
                onChange={field.handleChange}
                inputProps={{
                  disabled: !!initialProjectId,
                  onBlur: field.handleBlur,
                  error: fieldError(field),
                }}
              />
            )}
          </form.Field>

          <Alert
            icon={<IconInfoCircle size={20} />}
            variant="light"
            c="cyan.6"
          >
            <Text size="sm" lh={rem(20)} c="gray.9">
              {t('model.create.uploadTip')}
            </Text>
          </Alert>

          <form.Subscribe selector={s => [s.canSubmit, s.isSubmitting]}>
            {([canSubmit, isSubmitting]) => (
              <Group justify="flex-start" gap="md">
                <Button
                  type="submit"
                  disabled={!canSubmit}
                  loading={isSubmitting}
                >
                  {t('common.confirm')}
                </Button>
                <Button
                  color="default"
                  variant="subtle"
                  fw={400}
                  onClick={handleNavigateBack}
                >
                  {t('common.cancel')}
                </Button>
              </Group>
            )}
          </form.Subscribe>
        </Stack>
      </form>
    </Stack>
  )
}

export default ModelCreatePage
