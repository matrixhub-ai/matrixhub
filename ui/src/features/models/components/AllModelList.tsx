import {
  Box,
  Group,
  Space,
  Stack,
  Text,
} from '@mantine/core'
import { IconClock } from '@tabler/icons-react'
import { getRouteApi } from '@tanstack/react-router'
import { startTransition } from 'react'
import { useTranslation } from 'react-i18next'

import {
  splitFilterCsv, toSortParam, useModels,
} from '@/features/models/models.query'
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

  const query = {
    q: search.q ?? '',
    sort: search.sort ?? 'updatedAt',
    order: search.order ?? 'desc',
    page: search.page ?? 1,
  }

  const {
    data: {
      items = [],
      pagination,
    } = {},
    isLoading,
    isPending,
  } = useModels({
    search: query.q,
    sort: toSortParam(query.sort, query.order),
    project: search.project,
    labels: splitFilterCsv(search.task ?? search.library),
    page: query.page,
  })

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
          searchValue={query.q}
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
            fieldValue={query.sort}
            order={query.order}
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
                    order: query.order,
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
                    order: query.order === 'desc' ? 'asc' : 'desc',
                    page: 1,
                  }),
                })
              })
            }}
          />
        </SearchToolbar>

        <Space h="lg" />

        <Box miw={780} maw={1380}>
          <ResourceCardGrid
            loading={isLoading || isPending}
            skeletonCount={DEFAULT_PAGE_SIZE}
          >
            {cardElements}
          </ResourceCardGrid>

          <Pagination
            total={pagination?.total ?? 0}
            totalPages={pagination?.pages ?? 0}
            page={query.page}
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
