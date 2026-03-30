import {
  Alert,
  Box,
  Center,
  Flex,
  Group,
  Loader,
  Stack,
} from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import { getRouteApi, linkOptions } from '@tanstack/react-router'

import { FileViewer, useFileContent } from '@/features/file-viewer'
import { ModelRevisionSelect } from '@/features/models/components/ModelRevisionSelect.tsx'
import { useModelBlob } from '@/features/models/models.query.ts'
import { PathBreadcrumbs } from '@/shared/components/PathBreadcrumbs.tsx'

import { ModelBlobDemo } from './ModelBlobDemo'

const { useParams } = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId/blob/$ref/$')

export function ModelBlobPage() {
  const {
    projectId, modelId, ref: revision, _splat: path,
  } = useParams()

  const {
    data: file, isLoading: isFileLoading, error: fileError,
  } = useModelBlob(projectId, modelId, {
    revision,
    path,
  })
  // hook calls needs to be unconditional, but it returns empty state when file is undefined
  const {
    content, isLoading: isContentLoading, error: contentError,
  } = useFileContent(file)

  let body: React.ReactNode

  if (fileError) {
    // If backend API is not ready, we fall back to the demo component
    body = (
      <Stack gap="md">
        <Alert variant="light" color="blue" title="API Not Ready" icon={<IconInfoCircle />}>
          The ModelBlob query failed (likely because the backend API is not ready). Showing Demo data instead.
        </Alert>
        <ModelBlobDemo />
      </Stack>
    )
  } else if (isFileLoading) {
    body = (
      <Center p="xl"><Loader /></Center>
    )
  } else if (!file) {
    body = (
      <Alert variant="light" color="yellow">No file found.</Alert>
    )
  } else {
    body = (
      <Stack gap="md">
        <FileViewer
          file={file}
          content={content}
          loading={isContentLoading}
          error={contentError}
        />
      </Stack>
    )
  }

  return (
    <Box pt="sm" pb="xl">
      <Flex
        justify="space-between"
        align="center"
        mb="md"
        wrap="nowrap"
      >
        <Group gap="md" wrap="nowrap">
          <ModelRevisionSelect
            projectId={projectId}
            modelId={modelId}
            revision={revision}
          />

          <PathBreadcrumbs
            name={modelId}
            treePath={path}
            copyValue={path}
            getPathLinkProps={nextPath => linkOptions({
              to: '/projects/$projectId/models/$modelId/tree/$ref/$',
              params: {
                projectId,
                modelId,
                ref: revision,
                _splat: nextPath,
              },
            })}
          />
        </Group>
      </Flex>

      {body}
    </Box>
  )
}
