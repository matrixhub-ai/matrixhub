import {
  Accordion,
  Box,
  Button,
  Checkbox,
  Group,
  LoadingOverlay,
  MultiSelect,
  NumberInput,
  Radio,
  SimpleGrid,
  Stack,
  Text,
  TextInput,
  Textarea,
  rem,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import {
  type CreateRobotAccountRequest,
  type GetRobotAccountResponse,
  RobotAccountProjectScope,
  type UpdateRobotAccountRequest,
} from '@matrixhub/api-ts/v1alpha1/robot.pb'
import { IconArrowLeft } from '@tabler/icons-react'
import { useMutation } from '@tanstack/react-query'
import {
  Link,
  useNavigate,
} from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import { RobotTokenInfoModal } from './RobotTokenInfoModal.tsx'
import {
  createRobotAccountMutationOptions,
  updateRobotAccountMutationOptions,
} from '../robot.mutation'
import {
  useRobotPermissions,
  useRobotProjects,
} from '../robot.query'
import {
  ROBOT_DESCRIPTION_MAX_LENGTH,
  type RobotAccountFormValues,
  createRobotAccountFormSchema,
} from '../robot.schema'
import {
  getPlatformPermissionCategories,
  getProjectPermissionCategories,
  type RobotPermissionCategory,
} from '../robots.permissions'

interface RobotAccountFormProps {
  mode: 'create' | 'edit'
  defaultValues: RobotAccountFormValues
  robot?: GetRobotAccountResponse
}

function permissionSelectionState(
  selectedPermissions: string[],
  category: RobotPermissionCategory,
) {
  const permissionValues = category.permissions.map(item => item.value)
  const checkedCount = permissionValues.filter(value => selectedPermissions.includes(value)).length

  return {
    checked: checkedCount === permissionValues.length && checkedCount > 0,
    indeterminate: checkedCount > 0 && checkedCount < permissionValues.length,
  }
}

function nextPermissionsForCategory(
  selectedPermissions: string[],
  category: RobotPermissionCategory,
  checked: boolean,
) {
  const permissionValues = category.permissions.map(item => item.value)

  if (checked) {
    return Array.from(new Set([...selectedPermissions, ...permissionValues]))
  }

  return selectedPermissions.filter(value => !permissionValues.includes(value))
}

export function RobotAccountForm({
  mode,
  defaultValues,
  robot,
}: RobotAccountFormProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const createMutation = useMutation(createRobotAccountMutationOptions())
  const updateMutation = useMutation(updateRobotAccountMutationOptions())
  const {
    data: projects,
  } = useRobotProjects()
  const {
    data: permissionsResponse,
  } = useRobotPermissions()
  const [createdRobotName, setCreatedRobotName] = useState('')
  const [createdToken, setCreatedToken] = useState('')
  const [tokenOpened, tokenHandlers] = useDisclosure(false)

  const projectOptions = useMemo(() => {
    const uniqueNames = Array.from(new Set(
      (projects ?? [])
        .map(project => project.name?.trim() ?? '')
        .filter(Boolean),
    ))

    return uniqueNames.map(name => ({
      value: name,
      label: name,
    }))
  }, [projects])

  const platformPermissionCategories = useMemo(
    () => getPlatformPermissionCategories(permissionsResponse?.systemCategories),
    [permissionsResponse?.systemCategories],
  )
  const projectPermissionCategories = useMemo(
    () => getProjectPermissionCategories(permissionsResponse?.projectCategories),
    [permissionsResponse?.projectCategories],
  )

  const robotAccountFormSchema = createRobotAccountFormSchema(t)
  const getRobotAccountFieldIssue = (
    fieldName: keyof RobotAccountFormValues,
    values: RobotAccountFormValues,
  ) => {
    const result = robotAccountFormSchema.safeParse(values)

    if (result.success) {
      return undefined
    }

    return result.error.issues.find(issue => issue.path[0] === fieldName)?.message
  }

  const form = useForm({
    defaultValues,
    validators: {
      onSubmit: robotAccountFormSchema,
    },
    onSubmit: async ({ value }) => {
      if (mode === 'create') {
        const response = await createMutation.mutateAsync({
          name: value.name.trim(),
          description: value.description.trim(),
          expireDays: value.expiryMode === 'days' ? value.expireDays : undefined,
          platformPermissions: value.platformPermissions,
          projectPermissions: value.projectPermissions,
          projects: value.projectScope === RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED ? value.projects : [],
          projectScope: value.projectScope,
        } satisfies CreateRobotAccountRequest)

        setCreatedRobotName(value.name.trim())
        setCreatedToken(response.token ?? '')
        tokenHandlers.open()

        return
      }

      await updateMutation.mutateAsync({
        id: robot?.id,
        description: value.description.trim(),
        expireDays: value.expiryMode === 'days' ? value.expireDays : undefined,
        platformPermissions: value.platformPermissions,
        projectPermissions: value.projectPermissions,
        projects: value.projectScope === RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED ? value.projects : [],
        projectScope: value.projectScope,
      } satisfies UpdateRobotAccountRequest)

      await navigate({
        to: '/admin/robots',
      })
    },
  })

  const isMutating = createMutation.isPending || updateMutation.isPending

  const handleBack = async () => {
    await navigate({
      to: '/admin/robots',
    })
  }

  const handleTokenClose = async () => {
    tokenHandlers.close()
    await navigate({
      to: '/admin/robots',
    })
  }

  return (
    <>
      <Stack gap="lg" py="md">
        <Button
          component={Link}
          to="/admin/robots"
          justify="flex-start"
          leftSection={<IconArrowLeft size={32} />}
          variant="transparent"
          color="text/title"
          fw={600}
          px={0}
          fz={rem(18)}
        >
          {t(mode === 'create'
            ? 'routes.admin.robots.createRobot'
            : 'routes.admin.robots.editRobot')}
        </Button>

        <Box
          component="form"
          pos="relative"
          onSubmit={(event) => {
            event.preventDefault()
            void form.handleSubmit()
          }}
        >
          <LoadingOverlay visible={isMutating} />

          <Stack gap="xl">
            <Stack gap="md" w={350}>
              <Text fw={700}>
                {t('routes.admin.robots.sections.basic')}
              </Text>

              <form.Field
                name="name"
                validators={{
                  onChange: robotAccountFormSchema.shape.name,
                }}
              >
                {field => (
                  <TextInput
                    label={t('routes.admin.robots.fields.name')}
                    required
                    w={350}
                    disabled={mode === 'edit'}
                    value={field.state.value}
                    onChange={event => field.handleChange(event.currentTarget.value)}
                    onBlur={field.handleBlur}
                    error={fieldError(field)}
                  />
                )}
              </form.Field>

              <form.Field
                name="description"
                validators={{
                  onChange: robotAccountFormSchema.shape.description,
                }}
              >
                {field => (
                  <Textarea
                    label={t('routes.admin.robots.fields.description')}
                    autosize
                    minRows={3}
                    maxLength={ROBOT_DESCRIPTION_MAX_LENGTH}
                    value={field.state.value}
                    onChange={event => field.handleChange(event.currentTarget.value)}
                    onBlur={field.handleBlur}
                    error={fieldError(field)}
                    description={t('routes.admin.robots.fields.descriptionCount', {
                      count: field.state.value.length,
                      max: ROBOT_DESCRIPTION_MAX_LENGTH,
                    })}
                  />
                )}
              </form.Field>

              <form.Field name="expiryMode">
                {field => (
                  <Radio.Group
                    label={t('routes.admin.robots.fields.expiry')}
                    value={field.state.value}
                    onChange={value => field.handleChange(value as RobotAccountFormValues['expiryMode'])}
                  >
                    <Group mt="xs">
                      <Radio
                        value="never"
                        label={t('routes.admin.robots.expiry.never')}
                      />
                      <Radio
                        value="days"
                        label={t('routes.admin.robots.expiry.days')}
                      />
                    </Group>
                  </Radio.Group>
                )}
              </form.Field>

              <form.Subscribe selector={state => state.values.expiryMode}>
                {expiryMode => expiryMode === 'days' && (
                  <form.Field
                    name="expireDays"
                    validators={{
                      onChange: ({
                        value,
                        fieldApi,
                      }) => getRobotAccountFieldIssue(
                        'expireDays',
                        {
                          ...fieldApi.form.state.values,
                          expireDays: value,
                        },
                      ),
                      onChangeListenTo: ['expiryMode'],
                    }}
                  >
                    {field => (
                      <NumberInput
                        required
                        hideControls
                        min={1}
                        w={160}
                        rightSection={t('routes.admin.robots.fields.dayUnit')}
                        rightSectionWidth={rem(38)}
                        styles={{
                          input: {
                            paddingRight: rem(38),
                          },
                          section: {
                            backgroundColor: 'var(--mantine-color-gray-1)',
                            borderLeft: '1px solid var(--mantine-color-gray-3)',
                            color: 'var(--mantine-color-gray-6)',
                            fontSize: rem(14),
                          },
                        }}
                        value={field.state.value}
                        onChange={value => field.handleChange(typeof value === 'number' ? value : undefined)}
                        onBlur={field.handleBlur}
                        error={fieldError(field)}
                      />
                    )}
                  </form.Field>
                )}
              </form.Subscribe>
            </Stack>

            {renderPermissionSection(
              t('routes.admin.robots.sections.platformPermissions'),
              platformPermissionCategories,
              'platformPermissions',
            )}

            <Stack gap="md">
              <Text fw={700}>
                {t('routes.admin.robots.sections.projectPermissions')}
              </Text>

              <form.Field name="projectScope">
                {field => (
                  <Radio.Group
                    label={t('routes.admin.robots.fields.projectScope')}
                    required
                    value={field.state.value}
                    onChange={value => field.handleChange(value as RobotAccountFormValues['projectScope'])}
                  >
                    <Group mt="xs">
                      <Radio
                        value={RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_ALL}
                        label={t('routes.admin.robots.projectScopeForm.all')}
                      />
                      <Radio
                        value={RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED}
                        label={t('routes.admin.robots.projectScopeForm.selected')}
                      />
                    </Group>
                  </Radio.Group>
                )}
              </form.Field>

              <form.Subscribe selector={state => state.values.projectScope}>
                {projectScope => projectScope === RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED && (
                  <form.Field name="projects">
                    {field => (
                      <MultiSelect
                        label={t('routes.admin.robots.fields.projects')}
                        data={projectOptions}
                        searchable
                        placeholder={t('routes.admin.robots.fields.projectsPlaceholder')}
                        nothingFoundMessage={t('common.noResults')}
                        value={field.state.value}
                        onChange={field.handleChange}
                        onBlur={field.handleBlur}
                        error={fieldError(field)}
                        w={rem(266)}
                      />
                    )}
                  </form.Field>
                )}
              </form.Subscribe>

              {renderPermissionSection(
                t('routes.admin.robots.sections.projectPermissionMatrix'),
                projectPermissionCategories,
                'projectPermissions',
              )}
            </Stack>

            <Group justify="flex-end">
              <Button
                variant="default"
                onClick={() => {
                  void handleBack()
                }}
              >
                {t('common.cancel')}
              </Button>

              <form.Subscribe selector={state => ({
                isPristine: state.isPristine,
                isSubmitting: state.isSubmitting,
                values: state.values,
              })}
              >
                {({
                  isPristine,
                  isSubmitting,
                  values,
                }) => (
                  <Button
                    type="submit"
                    loading={isSubmitting}
                    disabled={
                      !robotAccountFormSchema.safeParse(values).success
                      || (mode === 'edit' && isPristine)
                    }
                  >
                    {t('common.confirm')}
                  </Button>
                )}
              </form.Subscribe>
            </Group>
          </Stack>
        </Box>
      </Stack>

      <RobotTokenInfoModal
        opened={tokenOpened}
        title={t('routes.admin.robots.tokenInfoModal.createSuccessTitle')}
        name={createdRobotName}
        token={createdToken}
        onClose={() => {
          void handleTokenClose()
        }}
      />
    </>
  )

  function renderPermissionSection(
    title: string,
    categories: RobotPermissionCategory[],
    fieldName: 'platformPermissions' | 'projectPermissions',
  ) {
    return (
      <Stack gap="md">
        <Text fw={700}>
          {title}
        </Text>

        <form.Field name={fieldName}>
          {field => (
            <Stack gap="sm">
              {categories.map((category) => {
                const selectionState = permissionSelectionState(field.state.value, category)

                return (
                  <Accordion
                    key={category.key}
                    variant="unstyled"
                    chevronPosition="left"
                  >
                    <Accordion.Item value={category.key} bd="none">
                      <Accordion.Control
                        px="sm"
                        py="xs"
                        bg="gray.0"
                        styles={{
                          chevron: {
                            width: rem(16),
                            height: rem(16),
                          },
                          label: {
                            padding: 0,
                            display: 'inline-flex',
                          },
                        }}
                      >
                        <Box
                          display="inline-block"
                          onClick={event => event.stopPropagation()}
                        >
                          <Checkbox
                            label={category.name}
                            checked={selectionState.checked}
                            indeterminate={selectionState.indeterminate}
                            onChange={(event) => {
                              field.handleChange(
                                nextPermissionsForCategory(
                                  field.state.value,
                                  category,
                                  event.currentTarget.checked,
                                ),
                              )
                            }}
                            styles={{
                              label: {
                                fontWeight: 600,
                                color: 'var(--mantine-color-text-title)',
                              },
                            }}
                          />
                        </Box>
                      </Accordion.Control>

                      <Accordion.Panel px="sm" py="sm">
                        <SimpleGrid
                          cols={{
                            base: 2,
                            sm: 3,
                            lg: 4,
                          }}
                        >
                          {category.permissions.map(permission => (
                            <Checkbox
                              key={permission.value}
                              label={permission.name}
                              checked={field.state.value.includes(permission.value)}
                              onChange={(event) => {
                                field.handleChange(
                                  event.currentTarget.checked
                                    ? Array.from(new Set([...field.state.value, permission.value]))
                                    : field.state.value.filter((value: string) => value !== permission.value),
                                )
                              }}
                            />
                          ))}
                        </SimpleGrid>
                      </Accordion.Panel>
                    </Accordion.Item>
                  </Accordion>
                )
              })}

              {fieldError(field) && (
                <Text c="red" size="sm">
                  {fieldError(field)}
                </Text>
              )}
            </Stack>
          )}
        </form.Field>
      </Stack>
    )
  }
}
