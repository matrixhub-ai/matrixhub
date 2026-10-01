/**
 * Scan API client for the security subsystem (issue #1066).
 *
 * These endpoints live under /api/scan/v1alpha1 and authenticate with the
 * browser session (same-origin cookies) or an access token.
 */

export type ScanStatus = 'unscanned' | 'scanning' | 'pass' | 'warning' | 'blocked' | 'failed'

export interface ScanFileSummary {
  path: string
  digest: string
  fileType?: string
  size: number
  severity: string
  rules?: string[]
}

export interface ScanReport {
  repoType: string
  project: string
  name: string
  revision: string
  status: ScanStatus
  verdict?: string
  taskId?: number
  files?: ScanFileSummary[]
  scannerVersions?: Record<string, string>
  counts?: Record<string, number>
  scannedAt?: string
  error?: string
}

export interface ScanStatusResp {
  revision: string
  status: ScanStatus
  hfStatus: string
}

export interface ScanAuditEvent {
  id: number
  repoType: string
  project: string
  name: string
  revision: string
  actor: string
  action: string
  decision: string
  reason: string
  taskId: number
  createdAt: string
}

async function getJSON<T>(url: string): Promise<T> {
  const resp = await fetch(url, { credentials: 'same-origin' })

  if (!resp.ok) {
    throw new Error(`scan api ${resp.status}: ${await resp.text()}`)
  }

  return resp.json() as Promise<T>
}

async function postJSON(url: string): Promise<Response> {
  return fetch(url, {
    method: 'POST',
    credentials: 'same-origin',
  })
}

export function fetchScanStatus(repoId: string, revision: string) {
  return getJSON<ScanStatusResp>(`/api/scan/v1alpha1/status/models/${repoId}/revision/${encodeURIComponent(revision)}`)
}

export function fetchScanReport(repoId: string, revision: string) {
  return getJSON<ScanReport>(`/api/scan/v1alpha1/reports/models/${repoId}/revision/${encodeURIComponent(revision)}`)
}

export function postRescan(repoId: string, revision: string) {
  return postJSON(`/api/scan/v1alpha1/reports/models/${repoId}/revision/${encodeURIComponent(revision)}/rescan`)
}

export function fetchScanAudit(name: string, limit = 20) {
  return getJSON<{ events: ScanAuditEvent[] }>(`/api/scan/v1alpha1/audit?name=${encodeURIComponent(name)}&limit=${limit}`)
}
