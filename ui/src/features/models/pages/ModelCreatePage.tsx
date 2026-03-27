import {
  Button,
  Combobox,
  Group,
  InputBase,
  Stack,
  Text,
  TextInput,
  Title,
  Tooltip,
  useCombobox,
} from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import { useForm } from '@tanstack/react-form'
import { useMutation } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { createModelMutationOptions } from '@/features/models/models.mutation'
import { useModelProjects } from '@/features/models/models.query.ts'
import { createModelSchema } from '@/features/models/models.schema'
import { ProjectTypeBadge } from '@/shared/components/badges/ProjectTypeBadge'

// TODO: if initialProjectId provided, whether project selection should be disabled or not, confirm with PM
interface ModelCreatePageProps {
  initialProjectId?: string
}

const routeApi = getRouteApi('/(auth)/(app)/models/')

export function ModelCreatePage({ initialProjectId = '' }: ModelCreatePageProps) {
  const { t } = useTranslation()
  const navigate = routeApi.useNavigate()

  const createMutation = useMutation(createModelMutationOptions())
  const { data: projects = [] } = useModelProjects()

  const projectCombobox = useCombobox()
  const modelCreateSchema = createModelSchema(t)

  const form = useForm({
    defaultValues: {
      name: '',
      projectId: initialProjectId?.trim(),
    },
    onSubmit: async ({ value }) => {
      await createMutation.mutateAsync({
        name: value.name,
        project: value.projectId,
      })

      // FIXME: confirm which page to navigate after creation
      void navigate({})
    },
  })

  useEffect(() => {
    const projectId = form.state.values.projectId

    if (projects.length && projectId && !projects?.find(option => option.name === projectId)) {
      form.setFieldValue('projectId', projects[0]?.name ?? '')
    }
  }, [projects, form])

  return (
    <Stack
      w="100%"
      maw={640}
      pt={20}
      gap="lg"
    >
      <Title order={3}>{t('model.new')}</Title>

      <form
        onSubmit={(event) => {
          event.preventDefault()
          void form.handleSubmit()
        }}
      >
        <Stack gap="md">
          <form.Field
            name="name"
            validators={{ onChange: modelCreateSchema.shape.name }}
          >
            {field => (
              <TextInput
                label={t('model.create.modelName')}
                withAsterisk
                placeholder={t('model.create.modelNamePlaceholder')}
                description={t('model.create.modelNameHelper')}
                value={field.state.value}
                error={field.state.meta.errors[0]?.message?.toString()}
                onBlur={field.handleBlur}
                onChange={e => field.handleChange(e.currentTarget.value)}
              />
            )}
          </form.Field>

          <form.Field
            name="projectId"
            validators={{ onChange: modelCreateSchema.shape.projectId }}
          >
            {(field) => {
              const selectedProjectOption = projects.find(option => option.name === field.state.value)

              return (
                <Combobox
                  store={projectCombobox}
                  onOptionSubmit={(value) => {
                    field.handleChange(value)
                    projectCombobox.closeDropdown()
                  }}
                >
                  <Combobox.Target>
                    <InputBase
                      component="button"
                      type="button"
                      label={(
                        <Group
                          component="span"
                          gap={6}
                          align="center"
                          wrap="nowrap"
                          style={{ display: 'inline-flex' }}
                        >
                          <span>{t('model.create.project')}</span>
                          <Tooltip label={t('model.create.projectTooltip')}>
                            <IconInfoCircle size={16} />
                          </Tooltip>
                        </Group>
                      )}
                      withAsterisk
                      error={field.state.meta.errors[0]?.message?.toString()}
                      rightSection={<Combobox.Chevron />}
                      rightSectionPointerEvents="none"
                      onBlur={field.handleBlur}
                      onClick={() => projectCombobox.toggleDropdown()}
                    >
                      {selectedProjectOption
                        ? (
                            <Group gap={8} wrap="nowrap">
                              <Text size="sm">{selectedProjectOption.name}</Text>
                              <ProjectTypeBadge type={selectedProjectOption.type} />
                            </Group>
                          )
                        : <Text c="dimmed" size="sm">{t('model.create.projectPlaceholder')}</Text>}
                    </InputBase>
                  </Combobox.Target>

                  <Combobox.Dropdown>
                    <Combobox.Options>
                      {projects.map(option => (
                        <Combobox.Option value={option.name as string} key={option.name}>
                          <Group gap={8} wrap="nowrap">
                            <Text size="sm">{option.name}</Text>
                            <ProjectTypeBadge type={option.type} />
                          </Group>
                        </Combobox.Option>
                      ))}
                    </Combobox.Options>
                  </Combobox.Dropdown>
                </Combobox>
              )
            }}
          </form.Field>

          <form.Subscribe selector={s => [s.canSubmit, s.isSubmitting, s.isPristine]}>
            {([canSubmit, isSubmitting, isPristine]) => (
              <Button
                type="submit"
                disabled={!canSubmit || isPristine}
                loading={isSubmitting}
              >
                {t('model.create.submit')}
              </Button>
            )}
          </form.Subscribe>

          <Text size="sm" c="dimmed">{t('model.create.uploadTip')}</Text>
        </Stack>
      </form>
    </Stack>
  )
}

export default ModelCreatePage
