import {
  ActionIcon,
  Flex,
  Group,
  Stack,
} from '@mantine/core'
import { IconArrowBackUp } from '@tabler/icons-react'
import { useSuspenseQuery } from '@tanstack/react-query'
import { getRouteApi, useNavigate } from '@tanstack/react-router'

import { modelQueryOptions, useModelCommit } from '@/features/models/models.query'
import { CommitDetail } from '@/shared/components/CommitDetail.tsx'
import { PathBreadcrumbs } from '@/shared/components/PathBreadcrumbs.tsx'

import './ModelCommitDetailPage.css'

const {
  useParams,
  useSearch,
} = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId/commit/$commitId/')

export function ModelCommitDetailPage() {
  const navigate = useNavigate()
  const {
    projectId,
    modelId,
    commitId,
  } = useParams()
  const { branch } = useSearch()

  const {
    data: commit,
  } = useModelCommit(projectId, modelId, commitId)

  const { data: model } = useSuspenseQuery(modelQueryOptions(projectId, modelId))

  const fallbackRef = branch ?? model.defaultBranch ?? ''

  const smartBack = () => {
    void navigate({
      to: '/projects/$projectId/models/$modelId/commits/$ref',
      params: {
        projectId,
        modelId,
        ref: fallbackRef,
      },
    })
  }

  return (
    <Stack pt="sm" gap="sm">
      <Flex justify="space-between" align="center" wrap="nowrap">
        <Group gap="sm" wrap="nowrap">
          <ActionIcon
            variant="filled"
            onClick={smartBack}
          >
            <IconArrowBackUp size={20} />
          </ActionIcon>

          <PathBreadcrumbs
            name={modelId}
            getPathLinkProps={() => ({
              to: '/projects/$projectId/models/$modelId/tree/$ref/$',
              params: {
                projectId,
                modelId,
                ref: fallbackRef,
              },
            })}
          />
        </Group>
      </Flex>

      <CommitDetail commit={commit} />
    </Stack>
  )
}
