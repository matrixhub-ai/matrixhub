import { useSuspenseQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'

import { modelRevisionsQueryOptions } from '@/features/models/models.query.ts'
import { RevisionSelect } from '@/shared/components/RevisionSelect.tsx'

interface ModelRevisionSelectProps {
  projectId: string
  modelId: string
  revision: string
}

export function ModelRevisionSelect({
  projectId,
  modelId,
  revision,
}: ModelRevisionSelectProps) {
  const navigate = useNavigate()
  const { data: revisions } = useSuspenseQuery(modelRevisionsQueryOptions(projectId, modelId))

  return (
    <RevisionSelect
      revision={revision}
      branches={revisions?.items?.branches}
      tags={revisions?.items?.tags}
      onRevisionChange={(nextRevision) => {
        void navigate({
          to: '.',
          params: prev => ({
            ...prev,
            ref: nextRevision,
          }),
          search: prev => ({
            ...prev,
            ...(prev.page ? { page: 1 } : {}),
          }),
        })
      }}
    />
  )
}
