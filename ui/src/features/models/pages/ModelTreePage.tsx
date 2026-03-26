import {
  Box, Button, Flex, Group, Text,
} from '@mantine/core'
import { FileType } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'
import {
  IconClockHour4, IconFile, IconFolderCode,
} from '@tabler/icons-react'
import { getRouteApi, Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { ModelRevisionSelect } from '@/features/models/components/ModelRevisionSelect.tsx'
import { useModelCommits, useModelTree } from '@/features/models/models.query.ts'
import { PathBreadcrumbs } from '@/shared/components/PathBreadcrumbs.tsx'

const { useParams } = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId/tree/$ref/$')

export function ModelTreePage() {
  const { t } = useTranslation()

  const {
    projectId, modelId, ref, _splat: treePath,
  } = useParams()

  const { data: { items } = {} } = useModelTree(projectId, modelId, {
    revision: ref,
    path: treePath,
  })
  const { data } = useModelCommits(projectId, modelId, {
    revision: ref,
    page: 1,
  })

  return (
    <Box pt="sm" pb="xl">
      <Flex justify="space-between">
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
      </Flex>

      { treePath }

      { items?.map(item => (
        <Box key={item.path}>
          <Flex>
            {
              item.type === FileType.DIR
                ? (
                    <>
                      <IconFolderCode size={16} />
                      <Text>{item.name}</Text>
                    </>
                  )
                : null
            }
            {
              item.type === FileType.FILE
                ? (
                    <>
                      <IconFile size={16} />
                      <Text>{item.name}</Text>
                    </>
                  )
                : null
            }
          </Flex>
        </Box>
      ))}
    </Box>
  )
}

export default ModelTreePage
