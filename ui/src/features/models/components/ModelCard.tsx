import { Link } from '@tanstack/react-router'

import {
  buildModelBadges,
  buildModelMetaItems,
  buildModelTitle,
} from '@/features/models/models.utils.tsx'
import { BaseCard } from '@/shared/components/resource-card/BaseCard.tsx'

import type { Model } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'

interface ModelCardProps {
  model: Model
  fallbackProjectId?: string
}

export function ModelCard({
  model,
  fallbackProjectId,
}: ModelCardProps) {
  const projectId = model.project?.trim() ?? fallbackProjectId?.trim()
  const modelName = model.name?.trim()

  return (
    <BaseCard
      title={buildModelTitle(model, projectId ?? '-')}
      renderRoot={projectId && modelName
        ? (props: Record<string, unknown>) => (
            <Link
              {...props}
              to="/projects/$projectId/models/$modelId"
              params={{
                projectId,
                modelId: modelName,
              }}
            />
          )
        : undefined}
      badges={buildModelBadges(model)}
      metaItems={buildModelMetaItems(model, projectId ?? '-')}
    />
  )
}
