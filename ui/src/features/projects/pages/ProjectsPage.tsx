import {
  Button,
  Group,
  Paper,
  Stack,
  Title,
} from '@mantine/core'
import { IconApiApp as ProjectIcon } from '@tabler/icons-react'
import {
  getRouteApi,
  useRouterState,
} from '@tanstack/react-router'
import {
  useCallback,
  useMemo,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'

import { ProjectsTable } from '../components/ProjectsTable'

import type { MRT_RowSelectionState } from 'mantine-react-table'

const projectsRouteApi = getRouteApi('/(auth)/(app)/projects/')

export function ProjectsPage() {
  const { t } = useTranslation()
  const navigate = projectsRouteApi.useNavigate()
  const search = projectsRouteApi.useSearch()
  const {
    projects,
    pagination,
  } = projectsRouteApi.useLoaderData()
  const loading = useRouterState({
    select: state => state.isLoading,
  })
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({})

  const handleSearchChange = useCallback((value: string) => {
    if (value === (search.query ?? '')) {
      return
    }

    setRowSelection({})
    void navigate({
      replace: true,
      search: prev => ({
        ...prev,
        page: 1,
        query: value,
      }),
    })
  }, [navigate, search.query])

  const handleCreate = () => {
    // TODO: open create project modal
  }

  const handleDelete = () => {
    // TODO: open delete project modal
  }

  const selectedProjects = useMemo(
    () => projects.filter(project => !!project.name && !!rowSelection[project.name]),
    [projects, rowSelection],
  )

  const handleBatchDelete = () => {
    if (selectedProjects.length === 0) {
      return
    }

    // TODO: open batch delete project modal with selectedProjects
  }

  const handleRefresh = useCallback(() => {
    setRowSelection({})
    // TODO： refresh projects list, currently we rely on router's loading state which is not ideal
  }, [])

  const handlePageChange = useCallback((page: number) => {
    setRowSelection({})
    void navigate({
      search: prev => ({
        ...prev,
        page,
      }),
    })
  }, [navigate])

  return (
    <Stack gap="lg">
      <Group gap="sm">
        <ProjectIcon size={24} />
        <Title order={2}>{t('routes.projects.title')}</Title>
      </Group>

      <Paper>
        <Stack gap="lg">

          <ProjectsTable
            records={projects}
            pagination={pagination}
            loading={loading}
            page={search.page ?? 1}
            searchValue={search.query ?? ''}
            onSearchChange={handleSearchChange}
            onRefresh={handleRefresh}
            onDelete={handleDelete}
            onBatchDelete={handleBatchDelete}
            rowSelection={rowSelection}
            onRowSelectionChange={setRowSelection}
            onPageChange={handlePageChange}
            selectedCount={selectedProjects.length}
            toolbarExtra={(
              <Button onClick={handleCreate}>
                {t('routes.projects.create')}
              </Button>
            )}
          />
        </Stack>
      </Paper>
    </Stack>
  )
}
