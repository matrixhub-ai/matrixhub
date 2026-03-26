import { Box } from '@mantine/core'
import { useSuspenseQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import Markdown from 'react-markdown'

import { modelQueryOptions } from '@/features/models/models.query.ts'

const { useParams } = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId')

export function ModelReadmePage() {
  const {
    projectId, modelId,
  } = useParams()

  const { data: model } = useSuspenseQuery(modelQueryOptions(projectId, modelId))

  return (
    <Box pt={20}>
      <Markdown>
        { model?.readmeContent }
      </Markdown>
    </Box>
  )
}
