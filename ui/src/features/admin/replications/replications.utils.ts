import cronstrue from 'cronstrue'
import 'cronstrue/locales/zh_CN'

import type { ReplicationBandwidthUnit } from './replications.schema'
import type { SyncPolicyItem } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'

const BANDWIDTH_UNIT_BASE = 1024
const CRON_FIELD_COUNT = 5

export function getReplicationRowId(item: SyncPolicyItem) {
  return String(item.id ?? item.name ?? '-')
}

export function getReplicationDisplayName(item: SyncPolicyItem) {
  return item.name ?? `#${item.id ?? '-'}`
}

/**
 * Format a bandwidth value from Kbps to a human-readable string.
 * By current product convention, the unit step is 1024, so 1024 Kbps = 1 Mbps.
 */
export function formatReplicationBandwidth(bandwidth: string | undefined): string {
  if (!bandwidth || bandwidth === '-1' || bandwidth === '0') {
    return ''
  }

  const kbps = Number(bandwidth)

  if (!Number.isFinite(kbps) || kbps <= 0) {
    return ''
  }

  if (kbps >= BANDWIDTH_UNIT_BASE) {
    const mbps = kbps / BANDWIDTH_UNIT_BASE

    return Number.isInteger(mbps) ? `${mbps} Mbps` : `${mbps.toFixed(1)} Mbps`
  }

  return `${kbps} Kbps`
}

export function convertBandwidthToApiValue(bandwidth: string, unit: ReplicationBandwidthUnit): string {
  if (!bandwidth || bandwidth === '-1') {
    return '-1'
  }

  const num = Number(bandwidth)

  if (!Number.isFinite(num) || num <= 0) {
    return '-1'
  }

  return unit === 'Mbps' ? String(num * BANDWIDTH_UNIT_BASE) : bandwidth
}

export function getDefaultBandwidthUnit(bandwidth?: string): ReplicationBandwidthUnit {
  if (!bandwidth || bandwidth === '-1' || bandwidth === '0') {
    return 'Kbps'
  }

  const kbps = Number(bandwidth)

  if (Number.isFinite(kbps) && kbps >= BANDWIDTH_UNIT_BASE && kbps % BANDWIDTH_UNIT_BASE === 0) {
    return 'Mbps'
  }

  return 'Kbps'
}

export function getDefaultBandwidthValue(bandwidth?: string): string {
  if (!bandwidth || bandwidth === '0') {
    return '-1'
  }

  const kbps = Number(bandwidth)

  if (Number.isFinite(kbps) && kbps >= BANDWIDTH_UNIT_BASE && kbps % BANDWIDTH_UNIT_BASE === 0) {
    return String(kbps / BANDWIDTH_UNIT_BASE)
  }

  return bandwidth
}

export function normalizeFiveFieldCronExpression(expression: string) {
  return expression.trim().replace(/\s+/g, ' ')
}

export function isFiveFieldCronExpression(expression: string) {
  const fields = normalizeFiveFieldCronExpression(expression).split(' ')

  return fields.length === CRON_FIELD_COUNT && fields.every(field => field.length > 0)
}

function getCronstrueLocale(language?: string) {
  return language?.toLowerCase().startsWith('zh') ? 'zh_CN' : 'en'
}

export function getCronExpressionDescription(expression: string, language?: string) {
  const normalizedExpression = normalizeFiveFieldCronExpression(expression)

  if (!isFiveFieldCronExpression(normalizedExpression)) {
    return undefined
  }

  try {
    return cronstrue.toString(normalizedExpression, {
      locale: getCronstrueLocale(language),
      throwExceptionOnParseError: true,
      use24HourTimeFormat: true,
      verbose: true,
    })
  } catch {
    return undefined
  }
}
