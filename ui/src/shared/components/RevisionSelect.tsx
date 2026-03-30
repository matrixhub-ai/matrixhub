import {
  type ComboboxItemGroup,
  Select,
} from '@mantine/core'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import type { Revision } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'

const EMPTY_REVISIONS: Revision[] = []

export interface RevisionSelectProps {
  revision?: string
  branches?: Revision[]
  tags?: Revision[]
  onRevisionChange?: (revision: string) => void
}

export function RevisionSelect({
  revision,
  branches = EMPTY_REVISIONS,
  tags = EMPTY_REVISIONS,
  onRevisionChange,
}: RevisionSelectProps) {
  const { t } = useTranslation()

  const revisionOptions: ComboboxItemGroup[] = []

  if (branches.length > 0) {
    revisionOptions.push({
      group: t('model.revisionPathHeader.group.branch'),
      items: branches?.filter(Boolean)?.map(item => ({
        label: item.name as string,
        value: `branch:${item.name}`,
      })),
    })
  }

  if (tags.length > 0) {
    revisionOptions.push({
      group: t('model.revisionPathHeader.group.tag'),
      items: tags?.filter(Boolean)?.map(item => ({
        label: item.name as string,
        value: `tag:${item.name}`,
      })),
    })
  }

  // Keep route params as raw revision names, but map to unique select option values.
  const selectedRevisionValue = useMemo(() => {
    if (!revision) {
      return null
    }

    if (branches.find(item => item.name === revision)) {
      return `branch:${revision}`
    }

    if (tags.find(item => item.name === revision)) {
      return `tag:${revision}`
    }

    return null
  }, [revision, branches, tags])

  return (
    <Select
      w={170}
      size="xs"
      value={selectedRevisionValue}
      data={revisionOptions}
      disabled={revisionOptions.length === 0}
      styles={{
        input: {
          fontSize: 'var(--mantine-font-size-sm)',
        },
        option: {
          fontSize: 'var(--mantine-font-size-sm)',
        },
      }}
      searchable
      onChange={(value, option) => {
        if (!value || !onRevisionChange) {
          return
        }

        onRevisionChange(option.label)
      }}
    />
  )
}
