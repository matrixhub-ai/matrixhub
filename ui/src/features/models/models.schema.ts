import { z } from 'zod'

import type { TFunction } from 'i18next'

const MODEL_NAME_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.-]*$/

export function createModelSchema(t: TFunction) {
  return z.object({
    name: z.string()
      .trim()
      .min(1, t('common.validation.fieldRequired', { field: t('model.create.modelName') }))
      .regex(MODEL_NAME_PATTERN, t('model.create.modelNameInvalid')),
    projectId: z.string()
      .trim()
      .min(1, t('common.validation.fieldRequired', { field: t('shared.projectSelect.project') })),
  })
}
