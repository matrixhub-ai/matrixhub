import {
  Box,
  Stack,
  Text,
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

/** Locale key suffix (under `model.detail.cache.tooltips.hints`) of what the viewer can do about the state. */
function tooltipHint(state: ModelCacheIndicatorState['state'], canPull: boolean) {
  switch (state) {
    case 'partial':
    case 'uncached':
      return canPull ? 'pull' : 'askToPull'
    case 'failed':
      return canPull ? 'retry' : 'askToPull'
    default:
      return undefined
  }
}

interface ModelCacheIndicatorProps {
  indicator: ModelCacheIndicatorState
  /** The viewer can pull this model on the server from the Download Model drawer of this page. */
  canPull: boolean
}

/** Read-only cache state of a proxy-project model, with what it means for workloads in a tooltip. */
export function ModelCacheIndicator({
  indicator,
  canPull,
}: ModelCacheIndicatorProps) {
  const { t } = useTranslation()
  const hint = tooltipHint(indicator.state, canPull)

  return (
    <Tooltip
      label={(
        <Stack gap={4}>
          <Text size="xs">{t(`model.detail.cache.tooltips.${indicator.state}`)}</Text>
          {hint && <Text size="xs">{t(`model.detail.cache.tooltips.hints.${hint}`)}</Text>}
        </Stack>
      )}
      multiline
      w={300}
      withArrow
      events={{
        hover: true,
        focus: true,
        touch: true,
      }}
    >
      {/* The tooltip needs a ref-able child; it is focusable so keyboard users can open it too. */}
      <Box component="span" display="inline-flex" tabIndex={0}>
        <StatusIndicator
          type={indicatorTypes[indicator.state]}
          dotSize={8}
          textProps={{ size: 'xs' }}
          label={t(`model.detail.cache.states.${indicator.state}`, {
            percent: indicator.belowOnePercent ? '<1' : indicator.percent,
          })}
        />
      </Box>
    </Tooltip>
  )
}
