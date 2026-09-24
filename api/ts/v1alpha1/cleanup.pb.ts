/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as GoogleProtobufDuration from "../google/protobuf/duration.pb"
export type ExecuteCleanupRequest = {
  cleanOrphanedRepos?: boolean
  cleanOrphanedLfs?: boolean
  dryRun?: boolean
  grace?: GoogleProtobufDuration.Duration
  maxDeletes?: number
  budget?: GoogleProtobufDuration.Duration
}

export type CleanupResult = {
  spaceReclaimedBytes?: string
  errors?: string[]
  orphanedRepos?: string[]
  gc?: GCResult
}

export type GCResult = {
  dryRun?: boolean
  repositories?: number
  deletedGitObjects?: number
  deletedGitBytes?: string
  gitReclaimedBytes?: string
  failed?: {[key: string]: string}
  liveObjects?: number
  unlinked?: string[]
  pruneSkippedInGrace?: number
  sweptShards?: number
  sweptXorbs?: number
  xetReclaimedBytes?: string
  sweepSkippedInGrace?: number
  dangling?: string[]
  unreadableShards?: string[]
  sweepDone?: boolean | null
  remainingShards?: number
  remainingXorbs?: number
}

export type GetStorageStatsRequest = {
}

export type StorageStats = {
  totalSizeBytes?: string
  git?: GitStorageUsage
  xet?: XetStorageUsage
}

export type StorageObjectUsage = {
  count?: string
  bytes?: string
}

export type GitStorageUsage = {
  objects?: StorageObjectUsage
  other?: StorageObjectUsage
}

export type XetStorageUsage = {
  xorbs?: StorageObjectUsage
  shards?: StorageObjectUsage
  fileIndex?: StorageObjectUsage
  chunkIndex?: StorageObjectUsage
  sha256Index?: StorageObjectUsage
}

export class Cleanup {
  static ExecuteCleanup(req: ExecuteCleanupRequest, initReq?: fm.InitReq): Promise<CleanupResult> {
    return fm.fetchReq<ExecuteCleanupRequest, CleanupResult>(`/api/v1alpha1/cleanup/execute`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetStorageStats(req: GetStorageStatsRequest, initReq?: fm.InitReq): Promise<StorageStats> {
    return fm.fetchReq<GetStorageStatsRequest, StorageStats>(`/api/v1alpha1/cleanup/stats?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
}