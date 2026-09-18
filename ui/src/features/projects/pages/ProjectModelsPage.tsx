import {
  Box,
  Button,
  Space,
  Stack,
} from '@mantine/core'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import {
  IconClock,
  IconCube,
  IconDownload,
} from '@tabler/icons-react'
import {
  useQuery,
  useSuspenseQuery,
} from '@tanstack/react-query'
import { getRouteApi, Link } from '@tanstack/react-router'
import {
  startTransition,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'

import { useProjectRole } from '@/features/auth/useProjectRole'
import { projectModelsQueryOptions } from '@/features/models/models.query.ts'
import {
  ProxyProjectDownloadDrawer,
  ProxyProjectDownloadGuide,
} from '@/features/projects/components/ProxyProjectDownload'
import { projectDetailQueryOptions } from '@/features/projects/projects.query'
import { isProxyProject } from '@/features/projects/projects.utils'
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
  const [downloadDrawerOpened, setDownloadDrawerOpened] = useState(false)
  const { data: project } = useSuspenseQuery(projectDetailQueryOptions(projectId))

  const projectRole = useProjectRole(projectId)
  const canCreateModel = projectRole === ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN
    || projectRole === ProjectRoleType.ROLE_TYPE_PROJECT_EDITOR

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
  const isProxy = isProxyProject(project.registryUrl)
  const showProxyDownloadGuide = isProxy && !isPending && project.modelCount === 0

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

          {isProxy
            ? (
                <Button
                  radius={6}
                  leftSection={<IconDownload size={16} />}
                  onClick={() => setDownloadDrawerOpened(true)}
                >
                  {t('projects.detail.proxyDownload.title')}
                </Button>
              )
            : canCreateModel && (
              <Link to="/models/new" search={{ projectId }}>
                <Button
                  radius={6}
                  leftSection={<IconCube size={16} />}
                >
                  {t('projects.detail.modelsPage.create')}
                </Button>
              </Link>
            )}
        </SearchToolbar>

        {showProxyDownloadGuide
          ? <ProxyProjectDownloadGuide remoteOrganization={project.organization} organization={project.name} />
          : (
              <>
                <Space h="lg" />

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
                  onPageChange={(nextPage) => {
                    void navigate({
                      search: prev => ({
                        ...prev,
                        page: nextPage,
                      }),
                    })
                  }}
                />
              </>
            )}
      </Stack>

      {isProxy && (
        <ProxyProjectDownloadDrawer
          opened={downloadDrawerOpened}
          remoteOrganization={project.organization}
          organization={project.name}
          onClose={() => setDownloadDrawerOpened(false)}
        />
      )}
    </Box>
  )
}
