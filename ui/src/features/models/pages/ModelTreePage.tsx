import {
  Box, Button, Flex, Group, TextInput,
} from '@mantine/core'
import { IconClockHour4, IconSearch } from '@tabler/icons-react'
import {
  getRouteApi, Link, linkOptions,
} from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ModelRevisionSelect } from '@/features/models/components/ModelRevisionSelect.tsx'
import { useModelCommits, useModelTree } from '@/features/models/models.query.ts'
import { PathBreadcrumbs } from '@/shared/components/PathBreadcrumbs.tsx'
import { RepoFileTree } from '@/shared/components/repo-file-tree'

import type { File as RepoFile } from '@matrixhub/api-ts/v1alpha1/model.pb'

const { useParams } = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId/tree/$ref/$')

export function ModelTreePage() {
  const { t } = useTranslation()

  const {
    projectId, modelId, ref, _splat: treePath,
  } = useParams()

  const {
    data: { items } = {}, isLoading,
  } = useModelTree(projectId, modelId, {
    revision: ref,
    path: treePath,
  })
  const { data } = useModelCommits(projectId, modelId, {
    revision: ref,
    page: 1,
    pageSize: 1,
  })

  const files = useMemo(() => items ?? [], [items])

  const [searchValue, setSearchValue] = useState('')

  const filteredFiles = useMemo(() => {
    if (!searchValue.trim()) {
      return files
    }
    const q = searchValue.toLowerCase()

    return files.filter(f => f.name?.toLowerCase().includes(q))
  }, [files, searchValue])

  const latestCommit = files.length > 0 ? findLatestCommit(files) : undefined

  return (
    <Box pt="sm" pb="xl">
      <Flex
        justify="space-between"
        align="center"
        mb="md"
      >
        <Group gap="md" wrap="nowrap">
          <ModelRevisionSelect
            projectId={projectId}
            modelId={modelId}
            revision={ref}
          />

          <PathBreadcrumbs
            name={modelId}
            treePath={treePath}
            getPathLinkProps={nextPath => ({
              to: '.',
              params: {
                projectId,
                modelId,
                ref,
                _splat: nextPath,
              },
            })}
          />
        </Group>

        <Group gap="md" wrap="nowrap">
          <TextInput
            placeholder={t('common.fileTree.searchPlaceholder')}
            value={searchValue}
            onChange={e => setSearchValue(e.currentTarget.value)}
            leftSection={<IconSearch size={16} />}
            size="xs"
            w={260}
            radius="lg"
          />

          <Link
            to="/projects/$projectId/models/$modelId/commits/$ref"
            params={{
              projectId,
              modelId,
              ref,
            }}
          >
            <Button
              size="xs"
              color="cyan"
              variant="light"
              radius="xl"
              leftSection={<IconClockHour4 size={16} />}
            >
              { t('model.detail.history', { count: data?.pagination?.total ?? 0 }) }
            </Button>
          </Link>
        </Group>
      </Flex>

      <RepoFileTree
        files={filteredFiles}
        latestCommit={latestCommit}
        buildTreeLink={path => linkOptions({
          to: '/projects/$projectId/models/$modelId/tree/$ref/$',
          params: {
            projectId,
            modelId,
            ref,
            _splat: path,
          },
        })}
        buildBlobLink={path => linkOptions({
          to: '/projects/$projectId/models/$modelId/blob/$ref/$',
          params: {
            projectId,
            modelId,
            ref,
            _splat: path,
          },
        })}
        buildCommitLink={commitId => linkOptions({
          to: '/projects/$projectId/models/$modelId/commit/$commitId',
          params: {
            projectId,
            modelId,
            commitId,
          },
        })}
        isLoading={isLoading}
      />
    </Box>
  )
}

function findLatestCommit(files: RepoFile[]): RepoFile['commit'] | undefined {
  let latest: RepoFile['commit'] | undefined

  for (const file of files) {
    if (!file.commit?.createdAt) {
      continue
    }
    if (!latest?.createdAt || Number(file.commit.createdAt) > Number(latest.createdAt)) {
      latest = file.commit
    }
  }

  return latest
}

export default ModelTreePage
