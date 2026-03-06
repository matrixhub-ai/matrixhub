/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as MatrixhubV1alpha1Model from "./model.pb"
import * as MatrixhubV1alpha1Utils from "./utils.pb"
export type ListDatasetTaskLabelsRequest = {
}

export type ListDatasetTaskLabelsResponse = {
  items?: MatrixhubV1alpha1Model.Label[]
}

export type ListDatasetsRequest = {
  label?: string[]
  search?: string
  sort?: string
  project?: string
  page?: number
  pageSize?: number
}

export type ListDatasetsResponse = {
  items?: Dataset[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type GetDatasetRequest = {
  project?: string
  name?: string
}

export type CreateDatasetRequest = {
  project?: string
  name?: string
}

export type CreateDatasetResponse = {
}

export type DeleteDatasetRequest = {
  project?: string
  name?: string
}

export type DeleteDatasetResponse = {
}

export type ListDatasetRevisionsRequest = {
  project?: string
  name?: string
}

export type ListDatasetRevisionsResponse = {
  items?: MatrixhubV1alpha1Model.Revisions
}

export type ListDatasetCommitsRequest = {
  project?: string
  name?: string
  revision?: string
  diff?: boolean
  page?: number
  pageSize?: number
}

export type ListDatasetCommitsResponse = {
  items?: MatrixhubV1alpha1Model.Commit[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type GetDatasetCommitRequest = {
  project?: string
  name?: string
  id?: string
}

export type GetDatasetTreeRequest = {
  project?: string
  name?: string
  revision?: string
  path?: string
}

export type GetDatasetTreeResponse = {
  items?: MatrixhubV1alpha1Model.Files[]
}

export type GetDatasetBlobRequest = {
  project?: string
  name?: string
  revision?: string
  path?: string
}

export type GetDatasetBlobResponse = {
  lfs?: boolean
  content?: string
  url?: string
}

export type Dataset = {
  id?: number
  project?: string
  name?: string
  nickname?: string
  defaultBranch?: string
  numRows?: string
  license?: string
  labels?: MatrixhubV1alpha1Model.Label[]
  size?: string
  cloneUrls?: MatrixhubV1alpha1Model.CloneUrls
  readmeContent?: string
  downloads?: string
  createdAt?: string
  updatedAt?: string
}

export class Datasets {
  static ListDatasetTaskLabels(req: ListDatasetTaskLabelsRequest, initReq?: fm.InitReq): Promise<ListDatasetTaskLabelsResponse> {
    return fm.fetchReq<ListDatasetTaskLabelsRequest, ListDatasetTaskLabelsResponse>(`/api/v1alpha1/datasets/task-labels?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static ListDatasets(req: ListDatasetsRequest, initReq?: fm.InitReq): Promise<ListDatasetsResponse> {
    return fm.fetchReq<ListDatasetsRequest, ListDatasetsResponse>(`/api/v1alpha1/datasets?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetDataset(req: GetDatasetRequest, initReq?: fm.InitReq): Promise<Dataset> {
    return fm.fetchReq<GetDatasetRequest, Dataset>(`/api/v1alpha1/datasets/${req["project"]}/${req["name"]}?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static CreateDataset(req: CreateDatasetRequest, initReq?: fm.InitReq): Promise<CreateDatasetResponse> {
    return fm.fetchReq<CreateDatasetRequest, CreateDatasetResponse>(`/api/v1alpha1/datasets`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteDataset(req: DeleteDatasetRequest, initReq?: fm.InitReq): Promise<DeleteDatasetResponse> {
    return fm.fetchReq<DeleteDatasetRequest, DeleteDatasetResponse>(`/api/v1alpha1/datasets/${req["project"]}/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static ListDatasetRevisions(req: ListDatasetRevisionsRequest, initReq?: fm.InitReq): Promise<ListDatasetRevisionsResponse> {
    return fm.fetchReq<ListDatasetRevisionsRequest, ListDatasetRevisionsResponse>(`/api/v1alpha1/datasets/${req["project"]}/${req["name"]}/revisions?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static ListDatasetCommits(req: ListDatasetCommitsRequest, initReq?: fm.InitReq): Promise<ListDatasetCommitsResponse> {
    return fm.fetchReq<ListDatasetCommitsRequest, ListDatasetCommitsResponse>(`/api/v1alpha1/datasets/${req["project"]}/${req["name"]}/commits?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static GetDatasetCommit(req: GetDatasetCommitRequest, initReq?: fm.InitReq): Promise<MatrixhubV1alpha1Model.Commit> {
    return fm.fetchReq<GetDatasetCommitRequest, MatrixhubV1alpha1Model.Commit>(`/api/v1alpha1/datasets/${req["project"]}/${req["name"]}/commits/${req["id"]}?${fm.renderURLSearchParams(req, ["project", "name", "id"])}`, {...initReq, method: "GET"})
  }
  static GetDatasetTree(req: GetDatasetTreeRequest, initReq?: fm.InitReq): Promise<GetDatasetTreeResponse> {
    return fm.fetchReq<GetDatasetTreeRequest, GetDatasetTreeResponse>(`/api/v1alpha1/datasets/${req["project"]}/${req["name"]}/tree?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static GetDatasetBlob(req: GetDatasetBlobRequest, initReq?: fm.InitReq): Promise<GetDatasetBlobResponse> {
    return fm.fetchReq<GetDatasetBlobRequest, GetDatasetBlobResponse>(`/api/v1alpha1/datasets/${req["project"]}/${req["name"]}/blob?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
}