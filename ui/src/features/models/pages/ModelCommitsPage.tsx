import {
  Flex, Group,
  Stack,
} from '@mantine/core'
import { getRouteApi, linkOptions } from '@tanstack/react-router'

import { ModelRevisionSelect } from '@/features/models/components/ModelRevisionSelect.tsx'
import { useModelCommits } from '@/features/models/models.query'
import { CommitsTable } from '@/shared/components/CommitsTable'
import { PathBreadcrumbs } from '@/shared/components/PathBreadcrumbs.tsx'

const {
  useNavigate, useParams, useSearch,
} = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId/commits/$ref/')

export function ModelCommitsPage() {
  const navigate = useNavigate()
  const {
    ref, projectId, modelId,
  } = useParams()
  const { page } = useSearch()

  const {
    data, isPending,
  } = useModelCommits(projectId, modelId, {
    revision: ref,
    page,
  })

  const onPageChange = (page: number) => {
    void navigate({
      search: prev => ({
        ...prev,
        page,
      }),
    })
  }

  return (
    <Stack pt="sm" pb="lg">
      <Flex justify="space-between" align="center" wrap="nowrap">
        <Group gap="md" wrap="nowrap">
          <ModelRevisionSelect
            projectId={projectId}
            modelId={modelId}
            revision={ref}
          />

          <PathBreadcrumbs
            name={modelId}
            getPathLinkProps={() => linkOptions({
              to: '/projects/$projectId/models/$modelId/tree/$ref/$',
              params: {
                projectId,
                modelId,
                ref,
              },
            })}
          />
        </Group>

        {/* <SearchInput */}
        {/*  w={260} */}
        {/*  value={search.q} */}
        {/*  onChange={onSearchChange} */}
        {/*  placeholder={t('shared.commitsTable.searchPlaceholder')} */}
        {/* /> */}
      </Flex>

      <CommitsTable
        commits={data?.items ?? []}
        pagination={data?.pagination}
        page={page}
        loading={isPending}
        onPageChange={onPageChange}
        getDetailLinkProps={commit => linkOptions({
          to: '/projects/$projectId/models/$modelId/commit/$commitId',
          params: {
            projectId,
            modelId,
            commitId: commit.id as string,
          },
          search: {
            branch: ref,
          },
        })}
      />
    </Stack>
  )
}
