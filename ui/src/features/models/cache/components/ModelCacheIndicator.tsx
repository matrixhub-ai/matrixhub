import {
  Box,
  Tooltip,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

import {
  StatusIndicator,
  type StatusIndicatorType,
} from '@/shared/components/StatusIndicator'

import type { ModelCacheIndicatorState } from '../cache.utils'

const indicatorTypes: Record<ModelCacheIndicatorState['state'], StatusIndicatorType> = {
  cached: 'success',
  partial: 'warning',
  uncached: 'neutral',
  running: 'info',
  failed: 'danger',
}

/** Read-only cache state of a proxy-project model, with what it means for workloads in a tooltip. */
export function ModelCacheIndicator({ indicator }: { indicator: ModelCacheIndicatorState }) {
  const { t } = useTranslation()

  return (
    <Tooltip
      label={t(`model.detail.cache.tooltips.${indicator.state}`)}
      multiline
      w={300}
      withArrow
    >
      {/* The tooltip needs a ref-able child; inline-flex keeps it on the line of the meta row. */}
      <Box component="span" display="inline-flex">
        <StatusIndicator
          type={indicatorTypes[indicator.state]}
          dotSize={8}
          textProps={{ size: 'xs' }}
          label={t(`model.detail.cache.states.${indicator.state}`, {
            done: indicator.doneFiles,
            total: indicator.totalFiles,
            percent: indicator.percent,
          })}
        />
      </Box>
    </Tooltip>
  )
}
