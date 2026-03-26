import type { SyncPolicyItem } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'

const BANDWIDTH_UNIT_BASE = 1024

export function getReplicationRowId(item: SyncPolicyItem) {
  return String(item.id ?? item.name ?? '-')
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
