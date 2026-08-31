import {
  Alert,
  Anchor,
  Box,
  Button,
  Center,
  Group,
  Paper,
  Space,
  Stack,
  Text,
} from '@mantine/core'
import { IconClock, IconCube } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, Link } from '@tanstack/react-router'
import { startTransition } from 'react'
import { useTranslation } from 'react-i18next'

import { catalogModelsQueryOptions } from '@/features/models/models.query'
import { Pagination } from '@/shared/components/Pagination'
import { ModelCard } from '@/shared/components/resource-card/ModelCard.tsx'
import { ResourceCardGrid } from '@/shared/components/ResourceCardGrid'
import { SearchToolbar } from '@/shared/components/SearchToolbar'
import {
  SortDropdown,
  type SortDropdownOption,
} from '@/shared/components/SortDropdown'
import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts'

const {
  useNavigate, useSearch,
} = getRouteApi('/(auth)/(app)/models/')

export function AllModelList() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const search = useSearch()

  const {
    data: {
      items = [],
      pagination,
    } = {},
    isLoading,
    isPending,
    isLoadingError,
    refetch,
  } = useQuery(catalogModelsQueryOptions(search))

  const hasActiveFilters = Boolean(search.q || search.task || search.library || search.project)
  const modelCount = pagination?.total ?? items.length
  const showCreatePrompt = !isLoading && !isPending && !isLoadingError
    && !hasActiveFilters && modelCount === 0

  const sortFieldOptions: SortDropdownOption[] = [
    {
      value: 'updatedAt',
      label: t('projects.detail.modelsPage.sortFieldUpdatedAt'),
      icon: <IconClock size={16} />,
    },
  ]

  const cardElements = items.map((model) => {
    const projectId = model.project?.trim() ?? '-'
    const modelName = model.name?.trim() ?? '-'

    return <ModelCard key={`${projectId}/${modelName}`} model={model} />
  })

  return (
    <Box>
      <Group>
        <Text fz="md" fw={600} lh="20px" mb="sm">
          {t('model.list.allModels') }
        </Text>
      </Group>

      <Stack gap={0}>
        <SearchToolbar
          searchPlaceholder={t('model.list.placeholder.modelName')}
          searchValue={search.q}
          onSearchChange={(nextQuery) => {
            void navigate({
              replace: true,
              search: prev => ({
                ...prev,
                q: nextQuery,
                page: 1,
              }),
            })
          }}
        >
          <SortDropdown
            fieldOptions={sortFieldOptions}
            fieldValue={search.sort}
            order={search.order}
            onFieldChange={(nextField) => {
              if (sortFieldOptions.find(o => o.value === nextField)?.disabled) {
                return
              }

              startTransition(() => {
                void navigate({
                  replace: true,
                  search: prev => ({
                    ...prev,
                    sort: nextField === 'updatedAt' ? nextField : prev.sort,
                    order: search.order,
                    page: 1,
                  }),
                })
              })
            }}
            onToggleOrder={() => {
              startTransition(() => {
                void navigate({
                  replace: true,
                  search: prev => ({
                    ...prev,
                    order: search.order === 'desc' ? 'asc' : 'desc',
                    page: 1,
                  }),
                })
              })
            }}
          />
        </SearchToolbar>

        <Space h="lg" />

        <Box miw={780} maw={1380}>
          {isLoadingError
            ? (
                <Alert color="red" title={t('model.list.loadFailed')}>
                  <Button
                    mt="sm"
                    size="xs"
                    variant="light"
                    onClick={() => void refetch()}
                  >
                    {t('model.list.retry')}
                  </Button>
                </Alert>
              )
            : showCreatePrompt
              ? (
                  <Paper withBorder radius="md">
                    <Center py="xl">
                      <Stack align="center" gap="xs">
                        <IconCube
                          size={48}
                          stroke={1.25}
                          color="var(--mantine-primary-color-filled)"
                        />
                        <Text fw={600}>{t('model.list.emptyTitle')}</Text>
                        <Text size="sm" c="dimmed">
                          {t('model.list.emptyDescription')}
                        </Text>
                        <Button component={Link} to="/models/new" mt="xs">
                          {t('model.list.create')}
                        </Button>
                        <Anchor
                          href={t('common.docs', { doc: '/docs/operations/model-repo/upload-download/' })}
                          target="_blank"
                          rel="noopener noreferrer"
                          size="sm"
                        >
                          {t('model.list.uploadGuide')}
                        </Anchor>
                      </Stack>
                    </Center>
                  </Paper>
                )
              : (
                  <ResourceCardGrid
                    loading={isLoading || isPending}
                    skeletonCount={DEFAULT_PAGE_SIZE}
                  >
                    {cardElements}
                  </ResourceCardGrid>
                )}

          <Pagination
            total={pagination?.total ?? 0}
            totalPages={pagination?.pages ?? 0}
            page={search.page}
            onPageChange={(nextPage) => {
              void navigate({
                search: prev => ({
                  ...prev,
                  page: nextPage,
                }),
              })
            }}
          />
        </Box>
      </Stack>
    </Box>
  )
}
