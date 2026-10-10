// Copyright The MatrixHub Authors. Licensed under Apache-2.0.
// Client for api/openapi/artifact-security.json. The existing generated *.pb.ts
// surface remains untouched while this opt-in API is reviewed.
import { fetchReq } from '@matrixhub/api-ts/fetch.pb'

export type ScanStatus = 'unscanned' | 'pending' | 'scanning' | 'passed' | 'warning' | 'blocked' | 'failed' | 'cancelled'
export interface ScanFinding {
  scanner: string
  version: string
  rule: string
  severity: string
}
export interface ScanFile {
  file_type?: string
  checked_at?: string
  recommended_action?: 'quarantine_and_replace' | 'manual_review' | 'distribute_by_policy' | 'repair_and_rescan'
  path: string
  sha256: string
  size: number
  status: ScanStatus
  findings: ScanFinding[]
  checks: string[]
  error?: string
  reused: boolean
  ruleset?: string
}
export interface ScanReport {
  repo: string
  revision: string
  status: ScanStatus
  files: ScanFile[] | null
  error?: string
  updated_at: string
  attempt: number
  ruleset?: string
  force: boolean
}
export interface ScanPolicy {
  block_severity: 'medium' | 'high'
  on_pending: 'block' | 'allow'
  on_failure: 'block' | 'allow'
}
export interface ScanAudit {
  id: number
  repo: string
  revision: string
  attempt: number
  actor: string
  action: string
  detail: string
  at: string
}

function request<T>(project: string, model: string, action: string, revision?: string, method = 'GET', body?: unknown): Promise<T> {
  const path = `/api/security/v1alpha1/models/${encodeURIComponent(project)}/${encodeURIComponent(model)}/${action}`
    + (revision ? `?revision=${encodeURIComponent(revision)}` : '')

  return fetchReq<unknown, T>(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  }).catch((error: unknown) => {
    if (error instanceof Error) {
      throw error
    }
    if (typeof error === 'object' && error !== null && 'error' in error) {
      throw new Error(String(error.error))
    }
    throw new Error(String(error))
  })
}

export const ModelSecurity = {
  report: (project: string, model: string, revision: string) => request<ScanReport>(project, model, 'report', revision),
  audit: (project: string, model: string) => request<{ events: ScanAudit[] }>(project, model, 'audit'),
  policy: (project: string, model: string) => request<ScanPolicy>(project, model, 'policy'),
  setPolicy: (project: string, model: string, policy: ScanPolicy) => request<ScanPolicy>(project, model, 'policy', undefined, 'PUT', policy),
  rescan: (project: string, model: string, revision: string) => request<ScanReport>(project, model, 'rescan', revision, 'POST', { force: true }),
  cancel: (project: string, model: string, revision: string) => request<ScanReport>(project, model, 'cancel', revision, 'POST', {}),
}
