import {
  Box,
  Button,
  Space,
  Stack,
} from '@mantine/core'
import {
  IconClock,
  IconCube,
} from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, Link } from '@tanstack/react-router'
import { startTransition } from 'react'
import { useTranslation } from 'react-i18next'

import { projectModelsQueryOptions } from '@/features/models/models.query.ts'
import { Pagination } from '@/shared/components/Pagination'
import { ModelCard } from '@/shared/components/resource-card/ModelCard.tsx'
import { ResourceCardGrid } from '@/shared/components/ResourceCardGrid'
import { SearchToolbar } from '@/shared/components/SearchToolbar'
import { SortDropdown } from '@/shared/components/SortDropdown'
import { DEFAULT_PAGE_SIZE } from '@/utils/constants.ts'

import type { SortDropdownOption } from '@/shared/components/SortDropdown'

const projectModelsRouteApi = getRouteApi('/(auth)/(app)/projects/$projectId/models/')

export function ProjectModelsPage() {
  const { projectId } = projectModelsRouteApi.useParams()
  const navigate = projectModelsRouteApi.useNavigate()
  const {
    q: query,
    sort: sortField,
    order: sortOrder,
    page,
  } = projectModelsRouteApi.useSearch()
  const { t } = useTranslation()

  const {
    data,
    isFetching,
    isPending,
  } = useQuery(projectModelsQueryOptions(projectId, projectModelsRouteApi.useSearch()))

  const models = data?.items ?? []
  const pagination = data?.pagination
  const total = pagination?.total ?? 0
  const totalPages = pagination?.pages
    ?? (
      pagination?.total && pagination?.pageSize
        ? Math.ceil(pagination.total / pagination.pageSize)
        : 0
    )
  const showSkeletons = isPending && !data
  const isRefreshing = isFetching && !showSkeletons

  const sortFieldOptions: SortDropdownOption[] = [
    {
      value: 'updatedAt',
      label: t('projects.detail.modelsPage.sortFieldUpdatedAt'),
      icon: <IconClock size={16} />,
    },
  ]

  const cardElements = models.map((model) => {
    const modelName = model.name?.trim() ?? '-'

    return (
      <ModelCard
        key={`${model.project?.trim() ?? projectId}/${modelName}`}
        model={model}
        fallbackProjectId={projectId}
      />
    )
  })

  return (
    <Box pt={20}>
      <Stack gap={0}>
        <SearchToolbar
          searchPlaceholder={t('projects.detail.modelsPage.searchPlaceholder')}
          searchValue={query}
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
            fieldValue={sortField}
            order={sortOrder}
            refreshing={isRefreshing}
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
                    order: sortOrder,
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
                    order: sortOrder === 'desc' ? 'asc' : 'desc',
                    page: 1,
                  }),
                })
              })
            }}
          />

          <Link to="/models/new" search={{ projectId }}>
            <Button
              h={32}
              px="md"
              radius={6}
              leftSection={<IconCube size={16} />}
            >
              {t('projects.detail.modelsPage.create')}
            </Button>
          </Link>
        </SearchToolbar>

        <Space h="lg"></Space>

        <ResourceCardGrid
          loading={showSkeletons}
          skeletonCount={DEFAULT_PAGE_SIZE}
        >
          {cardElements}
        </ResourceCardGrid>

        <Pagination
          total={total}
          totalPages={totalPages}
          page={page}
          paginationProps={{ withControls: false }}
          onPageChange={(nextPage) => {
            void navigate({
              search: prev => ({
                ...prev,
                page: nextPage,
              }),
            })
          }}
        />
      </Stack>
    </Box>
  )
}
