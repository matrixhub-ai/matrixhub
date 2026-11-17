import { Tooltip } from '@mantine/core'
import { useTranslation } from 'react-i18next'

import LfsIcon from '@/assets/svgs/lfs.svg?react'
import XetIcon from '@/assets/svgs/xet.svg?react'
import { BaseBadge } from '@/shared/components/badges/BaseBadge'

interface FileStorageBadgeProps {
  lfs?: boolean
  xetHash?: string
}

export function FileStorageBadge({
  lfs,
  xetHash,
}: FileStorageBadgeProps) {
  const { t } = useTranslation()

  if (!lfs) {
    return null
  }

  return (
    <Tooltip
      withArrow
      label={xetHash ? t('common.fileTree.xetStored') : t('common.fileTree.lfsStored')}
    >
      <BaseBadge
        h={18}
        flex="0 0 auto"
        icon={xetHash ? <XetIcon width={12} height={12} /> : <LfsIcon width={12} height={12} />}
        label={xetHash ? 'xet' : 'LFS'}
        styles={{ root: { paddingInline: 6 } }}
      />
    </Tooltip>
  )
}
