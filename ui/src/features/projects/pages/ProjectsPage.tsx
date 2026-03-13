import {
  Button,
  Group,
  Paper,
  Stack,
  Text,
  Title,
} from '@mantine/core'
import { useDebouncedValue } from '@mantine/hooks'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import ProjectIcon from '@/assets/svgs/project.svg?react'

import { ProjectsTable } from '../components/ProjectsTable'
import { useProjects } from '../hooks/useProjects'

export function ProjectsPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const [debouncedQuery] = useDebouncedValue(query, 300)
  const [page, setPage] = useState(1)

  const handleCreate = () => {
    // TODO: open create project modal
  }

  const handleDelete = () => {
    // TODO: open delete project modal
  }

  const {
    projects, pagination, loading, refresh,
  } = useProjects({
    name: debouncedQuery || undefined,
    page,
  })

  return (
    <Stack gap="lg">
      <Group gap="sm">
        <ProjectIcon width={24} />
        <Title order={2}>{t('routes.projects.title')}</Title>
      </Group>

      <Paper
        withBorder
        radius="lg"
        p="xl"
        shadow="xs"
      >
        <Stack gap="lg">
          <Text
            c="dimmed"
            size="sm"
          >
            {t('routes.projects.description')}
          </Text>

          <ProjectsTable
            records={projects}
            pagination={pagination}
            loading={loading}
            page={page}
            searchValue={query}
            onSearchChange={(value) => {
              setQuery(value)
              setPage(1)
            }}
            onRefresh={refresh}
            onDelete={handleDelete}
            onPageChange={nextPage => setPage(nextPage)}
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
