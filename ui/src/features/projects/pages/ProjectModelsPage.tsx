import {
  Box,
  Button,
  Center,
  Space,
  Stack,
  Text,
} from '@mantine/core'
import { Category } from '@matrixhub/api-ts/v1alpha1/model.pb'
import {
  getRouteApi,
  Link,
} from '@tanstack/react-router'
import { startTransition } from 'react'
import { useTranslation } from 'react-i18next'

import BinaryTreeIcon from '@/assets/svgs/binary-tree.svg?react'
import ClockIcon from '@/assets/svgs/clock.svg?react'
import ModelIcon from '@/assets/svgs/model.svg?react'
import PhotoUpIcon from '@/assets/svgs/photo-up.svg?react'
import PytorchIcon from '@/assets/svgs/pytorch.svg?react'
import {
  PAGE_SIZE,
  useModels,
} from '@/features/models/models.query'
import {
  buildModelBadges,
  buildModelMetaItems,
  buildModelTitle,
} from '@/features/models/models.utils'
import { Pagination } from '@/shared/components/Pagination'
import { ResourceCard } from '@/shared/components/ResourceCard'
import { ResourceCardGrid } from '@/shared/components/ResourceCardGrid'
import { SearchToolbar } from '@/shared/components/SearchToolbar'
import { SortDropdown } from '@/shared/components/SortDropdown'

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
    isError,
    isFetching,
    isPending,
  } = useModels(projectId, {
    q: query,
    sort: sortField,
    order: sortOrder,
    page,
  })

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
  const showErrorState = isError && !data
  const isRefreshing = isFetching && !showSkeletons
  const isEmpty = !showSkeletons && !showErrorState && models.length === 0

  const sortFieldOptions: SortDropdownOption[] = [
    {
      value: 'updatedAt',
      label: t('projects.detail.modelsPage.sortFieldUpdatedAt'),
      icon: <ClockIcon width={16} height={16} />,
    },
  ]

  const cardElements = models.map((model) => {
    const modelName = model.name?.trim()

    return (
      <ResourceCard
        key={`${model.project ?? projectId}/${model.name ?? '-'}`}
        title={buildModelTitle(model, projectId)}
        renderRoot={modelName
          ? (props: Record<string, unknown>) => (
              <Link
                {...props}
                to="/projects/$projectId/models/$modelId"
                params={{
                  projectId,
                  modelId: modelName,
                }}
              />
            )
          : undefined}
        badges={buildModelBadges(model, {
          taskCategory: Category.TASK,
          libraryCategory: Category.LIBRARY,
          taskIcon: (
            <PhotoUpIcon
              width={16}
              height={16}
              style={{ color: 'var(--mantine-color-blue-4)' }}
            />
          ),
          libraryIconFn: name => /pytorch/i.test(name)
            ? <PytorchIcon width={16} height={16} />
            : undefined,
          parameterCountIcon: (
            <BinaryTreeIcon
              width={16}
              height={16}
              style={{ color: 'var(--mantine-color-violet-4)' }}
            />
          ),
        })}
        metaItems={buildModelMetaItems(model, projectId)}
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

          <Button
            h={32}
            px="md"
            radius={6}
            leftSection={<ModelIcon width={16} height={16} />}
            component={Link}
            to="/models/new"
          >
            {t('projects.detail.modelsPage.create')}
          </Button>
        </SearchToolbar>

        <Space h="lg"></Space>

        <ResourceCardGrid
          loading={showSkeletons}
          skeletonCount={PAGE_SIZE}
        >
          {cardElements}
        </ResourceCardGrid>

        {isEmpty && (
          <Center py="xl">
            <Stack align="center" gap="xs">
              <Text fw={500}>{t('projects.detail.modelsPage.emptyTitle')}</Text>
              <Text size="sm" c="dimmed">
                {t('projects.detail.modelsPage.emptyDescription')}
              </Text>
            </Stack>
          </Center>
        )}

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
