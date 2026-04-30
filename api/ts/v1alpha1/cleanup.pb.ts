/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type PreviewCleanupRequest = {
  includeOrphanedRepos?: boolean
  includeOrphanedLfs?: boolean
}

export type CleanupPreview = {
  orphanedRepos?: OrphanedRepo[]
  orphanedLfsObjects?: OrphanedLFS[]
  totalReclaimableBytes?: string
}

export type OrphanedRepo = {
  path?: string
  type?: string
  projectName?: string
  resourceName?: string
  sizeBytes?: string
}

export type OrphanedLFS = {
  oid?: string
  sizeBytes?: string
}

export type ExecuteCleanupRequest = {
  cleanOrphanedRepos?: boolean
  cleanOrphanedLfs?: boolean
  dryRun?: boolean
}

export type CleanupResult = {
  reposDeleted?: number
  lfsObjectsDeleted?: number
  spaceReclaimedBytes?: string
  errors?: string[]
}

export type GetStorageStatsRequest = {
}

export type StorageStats = {
  totalSizeBytes?: string
  repositoriesSizeBytes?: string
  lfsSizeBytes?: string
  orphanedSizeBytes?: string
}

export class Cleanup {
  static PreviewCleanup(req: PreviewCleanupRequest, initReq?: fm.InitReq): Promise<CleanupPreview> {
    return fm.fetchReq<PreviewCleanupRequest, CleanupPreview>(`/api/v1alpha1/cleanup/preview`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ExecuteCleanup(req: ExecuteCleanupRequest, initReq?: fm.InitReq): Promise<CleanupResult> {
    return fm.fetchReq<ExecuteCleanupRequest, CleanupResult>(`/api/v1alpha1/cleanup/execute`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetStorageStats(req: GetStorageStatsRequest, initReq?: fm.InitReq): Promise<StorageStats> {
    return fm.fetchReq<GetStorageStatsRequest, StorageStats>(`/api/v1alpha1/cleanup/stats?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
}