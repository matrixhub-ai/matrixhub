import { Box } from '@mantine/core'
import { getRouteApi } from '@tanstack/react-router'
import Markdown from 'react-markdown'

import { EmptyStatePrompt } from '@/shared/components/EmptyStatePrompt.tsx'

const { useLoaderData } = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId')

export function ModelReadmePage() {
  const model = useLoaderData()

  return (
    <Box pt={20}>
      {
        model.readmeContent
          ? (
              <Markdown>
                { model?.readmeContent }
              </Markdown>
            )
          : <EmptyStatePrompt />
      }
    </Box>
  )
}
