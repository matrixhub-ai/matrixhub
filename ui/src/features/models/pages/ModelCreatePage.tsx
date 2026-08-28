import {
  Alert,
  Box,
  Button,
  Group,
  Paper,
  rem,
  Stack,
  Text,
  TextInput,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  IconFolderPlus,
  IconInfoCircle,
  IconPlus,
} from '@tabler/icons-react'
import { useMutation } from '@tanstack/react-query'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { createModelMutationOptions } from '@/features/models/models.mutation'
import { useWritableModelProjects } from '@/features/models/models.query.ts'
import { createModelSchema } from '@/features/models/models.schema'
import { CreateProjectModal } from '@/features/projects/components/CreateProjectModal'
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
  const [createProjectOpened, createProjectHandlers] = useDisclosure(false)

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

  const {
    data: projects = [],
    isLoadingError: isProjectsLoadingError,
    isPending: isProjectsPending,
    refetch: refetchProjects,
  } = useWritableModelProjects()
  const showEmptyProjectPrompt = !initialProjectId
    && !isProjectsPending
    && !isProjectsLoadingError
    && projects.length === 0

  const handleProjectCreated = async (projectName: string) => {
    await refetchProjects()
    form.setFieldValue('projectId', projectName)
  }

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
            {(field) => {
              const error = fieldError(field)

              if (showEmptyProjectPrompt) {
                return (
                  <Paper withBorder radius="md" px="lg" py="xl">
                    <Stack align="center" gap="xs">
                      <IconFolderPlus
                        size={44}
                        stroke={1.25}
                        color="var(--mantine-primary-color-filled)"
                      />
                      <Text fw={600}>{t('model.create.emptyProjectTitle')}</Text>
                      <Text size="sm" c="dimmed" ta="center">
                        {t('model.create.emptyProjectDescription')}
                      </Text>
                      <Button
                        type="button"
                        mt="xs"
                        onClick={createProjectHandlers.open}
                      >
                        {t('model.create.createProject')}
                      </Button>
                    </Stack>
                  </Paper>
                )
              }

              return (
                <Group align="flex-start" gap="sm" wrap="nowrap">
                  <Box flex="0 0 100%">
                    <ProjectSelect
                      data={projects}
                      value={field.state.value}
                      onChange={field.handleChange}
                      inputProps={{
                        disabled: !!initialProjectId || isProjectsPending,
                        onBlur: field.handleBlur,
                        error,
                        styles: {
                          input: error
                            ? {
                                borderColor: 'var(--mantine-color-red-4)',
                                boxShadow: '0 0 0 2px var(--mantine-color-red-0)',
                              }
                            : undefined,
                          error: {
                            color: 'var(--mantine-color-red-6)',
                            fontWeight: 500,
                            lineHeight: rem(16),
                            marginTop: rem(5),
                          },
                        },
                      }}
                    />
                  </Box>
                  {!initialProjectId && (
                    <Button
                      type="button"
                      variant="subtle"
                      mt={rem(29)}
                      h={36}
                      flex="0 0 auto"
                      px="xs"
                      leftSection={<IconPlus size={16} />}
                      onClick={createProjectHandlers.open}
                    >
                      {t('model.create.createProject')}
                    </Button>
                  )}
                </Group>
              )
            }}
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

      <CreateProjectModal
        opened={createProjectOpened}
        onClose={createProjectHandlers.close}
        onCreated={handleProjectCreated}
      />
    </Stack>
  )
}

export default ModelCreatePage
