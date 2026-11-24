import {
  ActionIcon,
  Alert,
  Anchor,
  Button,
  Group,
  rem,
  Stack,
  Text,
  TextInput,
  Title,
  Tooltip,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  IconCube,
  IconExternalLink,
  IconInfoCircle,
  IconRefresh,
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

import classes from './ModelCreatePage.module.css'

interface ModelCreatePageProps {
  initialProjectId?: string
}

const fieldStyles = {
  label: { display: 'block' },
  wrapper: {
    '--input-height-sm': rem(32),
    '--input-padding': rem(11),
    marginBottom: 0,
  },
  section: { insetInlineEnd: 0 },
  description: {
    marginTop: rem(4),
    lineHeight: rem(18),
  },
  error: { marginTop: rem(4) },
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

  const {
    data: projects,
    isFetching: isFetchingProjects,
    refetch: refetchProjects,
  } = useWritableModelProjects()
  const fixedProjectUnavailable = !!initialProjectId && projects !== undefined
    && !projects.some(project => project.name === initialProjectId)

  const form = useForm({
    defaultValues: {
      name: '',
      projectId: initialProjectId,
    },
    onSubmit: async ({ value }) => {
      if (!projects?.some(project => project.name === value.projectId)) {
        return
      }

      await mutation.mutateAsync({
        name: value.name.trim(),
        project: value.projectId,
      })

      handleNavigateBack()
    },
  })

  const handleProjectCreated = async (projectName: string) => {
    await refetchProjects()
    form.setFieldValue('projectId', projectName)
  }

  useEffect(() => {
    if (initialProjectId || !projects) {
      return
    }

    const selected = form.state.values.projectId
    const fallback = projects[0]?.name ?? ''

    if (selected !== fallback && !projects.some(project => project.name === selected)) {
      form.setFieldValue('projectId', fallback)
    }
  }, [projects, initialProjectId, form])

  return (
    <>
      <Stack
        component="form"
        aria-labelledby="model-create-title"
        maw={622}
        mx="auto"
        my={32}
        gap="lg"
        onSubmit={(event) => {
          event.preventDefault()
          void form.handleSubmit()
        }}
      >
        <Group gap={6}>
          <IconCube size={20} color="var(--app-color-text-label)" aria-hidden />
          <Title id="model-create-title" order={1} size="md" lh={rem(24)}>{t('model.new')}</Title>
        </Group>

        <div className={classes.fields}>
          <form.Field name="projectId" validators={{ onChange: projectIdValidator }}>
            {field => (
              <Stack gap={4}>
                <ProjectSelect
                  data={projects}
                  value={field.state.value}
                  onChange={field.handleChange}
                  inputProps={{
                    disabled: !!initialProjectId || !projects?.length,
                    onBlur: field.handleBlur,
                    error: fixedProjectUnavailable ? t('model.create.projectUnavailable') : fieldError(field),
                    rightSectionWidth: 40,
                    styles: fieldStyles,
                  }}
                />
                {!initialProjectId && (
                  <>
                    {projects?.length === 0 && (
                      <Text size="xs" c="dimmed">{t('model.create.emptyProjectDescription')}</Text>
                    )}
                    <Group justify="space-between" gap="xs">
                      <Anchor
                        component="button"
                        type="button"
                        aria-haspopup="dialog"
                        size="xs"
                        lh={rem(18)}
                        c="var(--mantine-primary-color-filled)"
                        onClick={createProjectHandlers.open}
                      >
                        <Group component="span" gap={4} wrap="nowrap">
                          {t('model.create.createProject')}
                          <IconExternalLink size={12} aria-hidden />
                        </Group>
                      </Anchor>
                      <Tooltip label={t('model.create.refreshProjects')}>
                        <ActionIcon
                          type="button"
                          variant="subtle"
                          color="gray"
                          size="xs"
                          loading={isFetchingProjects}
                          aria-label={t('model.create.refreshProjects')}
                          onClick={() => void refetchProjects()}
                        >
                          <IconRefresh size={16} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
                  </>
                )}
              </Stack>
            )}
          </form.Field>

          <form.Field name="name" validators={{ onChange: nameValidator }}>
            {field => (
              <TextInput
                label={t('model.create.modelName')}
                withAsterisk
                aria-required
                description={t('model.create.modelNameHelper')}
                inputWrapperOrder={['label', 'input', 'description', 'error']}
                styles={fieldStyles}
                value={field.state.value}
                onBlur={field.handleBlur}
                onChange={e => field.handleChange(e.currentTarget.value)}
                error={fieldError(field)}
              />
            )}
          </form.Field>
        </div>

        <Stack gap={6}>
          <form.Subscribe selector={s => s.values}>
            {values => (
              <Text size="sm" lh={rem(20)} fw={600} c="var(--app-color-text-label)" className={classes.identifier}>
                {t('model.create.identifier', {
                  project: values.projectId || '—',
                  name: values.name.trim() || '—',
                })}
              </Text>
            )}
          </form.Subscribe>
          <Alert
            icon={<IconInfoCircle size={20} />}
            variant="light"
            px={12}
            py={12}
            styles={{ icon: { marginTop: 0 } }}
          >
            <Text size="sm" lh={rem(20)} c="var(--mantine-color-text)">{t('model.create.uploadTip')}</Text>
          </Alert>
        </Stack>

        <form.Subscribe selector={s => [s.canSubmit, s.isSubmitting]}>
          {([canSubmit, isSubmitting]) => (
            <Group gap={12}>
              <Button
                type="submit"
                fw={400}
                disabled={!canSubmit || !projects?.length || fixedProjectUnavailable}
                loading={isSubmitting}
              >
                {t('common.confirm')}
              </Button>
              <Button
                type="button"
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

      <CreateProjectModal
        opened={createProjectOpened}
        onClose={createProjectHandlers.close}
        onCreated={handleProjectCreated}
      />
    </>
  )
}

export default ModelCreatePage
