/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as GoogleProtobufWrappers from "../google/protobuf/wrappers.pb"
import * as MatrixhubV1alpha1Utils from "./utils.pb"

export enum FileType {
  DIR = "DIR",
  FILE = "FILE",
}

export enum Category {
  TASK = "TASK",
  LIBRARY = "LIBRARY",
  LICENSE = "LICENSE",
  LANGUAGE = "LANGUAGE",
  OTHER = "OTHER",
}

export type ListModelTaskLabelsRequest = {
}

export type ListModelTaskLabelsResponse = {
  items?: Label[]
}

export type ListModelFrameLabelsRequest = {
}

export type ListModelFrameLabelsResponse = {
  items?: Label[]
}

export type ListModelsRequest = {
  labels?: string[]
  search?: string
  sort?: string
  project?: string
  page?: number
  pageSize?: number
  popular?: boolean
}

export type ListModelsResponse = {
  items?: Model[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type GetModelRequest = {
  project?: string
  name?: string
}

export type CreateModelRequest = {
  project?: string
  name?: string
}

export type CreateModelResponse = {
}

export type DeleteModelRequest = {
  project?: string
  name?: string
}

export type DeleteModelResponse = {
}

export type ListModelRevisionsRequest = {
  project?: string
  name?: string
}

export type ListModelRevisionsResponse = {
  items?: Revisions
}

export type Revisions = {
  branches?: Revision[]
  tags?: Revision[]
}

export type ListModelCommitsRequest = {
  project?: string
  name?: string
  revision?: string
  diff?: boolean
  page?: number
  pageSize?: number
}

export type ListModelCommitsResponse = {
  items?: Commit[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type GetModelCommitRequest = {
  project?: string
  name?: string
  id?: string
}

export type GetModelTreeRequest = {
  project?: string
  name?: string
  revision?: string
  path?: string
}

export type File = {
  name?: string
  type?: FileType
  path?: string
  size?: string
  lfs?: boolean
  sha256?: string
  commit?: Commit
  url?: string
}

export type GetModelTreeResponse = {
  items?: File[]
}

export type GetModelBlobRequest = {
  project?: string
  name?: string
  revision?: string
  path?: string
}

export type Model = {
  id?: number
  name?: string
  nickname?: string
  defaultBranch?: string
  createdAt?: string
  updatedAt?: string
  cloneUrls?: CloneUrls
  labels?: Label[]
  project?: string
  readmeContent?: string
  size?: string
  parameterCount?: string
  popular?: boolean
}

export type CloneUrls = {
  sshUrl?: string
  httpUrl?: string
}

export type Revision = {
  name?: string
}

export type Commit = {
  id?: string
  message?: string
  authorName?: string
  authorEmail?: string
  authorDate?: string
  committerName?: string
  committerEmail?: string
  committerDate?: string
  diff?: string
  createdAt?: string
}

export type Label = {
  id?: number
  name?: string
  category?: Category
  createdAt?: string
  updatedAt?: string
}

export type UpdateModelSettingRequest = {
  project?: string
  name?: string
  popular?: GoogleProtobufWrappers.BoolValue
}

export type UpdateModelSettingResponse = {
}

export class Models {
  static ListModelTaskLabels(req: ListModelTaskLabelsRequest, initReq?: fm.InitReq): Promise<ListModelTaskLabelsResponse> {
    return fm.fetchReq<ListModelTaskLabelsRequest, ListModelTaskLabelsResponse>(`/api/v1alpha1/models/task-labels?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static ListModelFrameLabels(req: ListModelFrameLabelsRequest, initReq?: fm.InitReq): Promise<ListModelFrameLabelsResponse> {
    return fm.fetchReq<ListModelFrameLabelsRequest, ListModelFrameLabelsResponse>(`/api/v1alpha1/models/library-labels?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static ListModels(req: ListModelsRequest, initReq?: fm.InitReq): Promise<ListModelsResponse> {
    return fm.fetchReq<ListModelsRequest, ListModelsResponse>(`/api/v1alpha1/models?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetModel(req: GetModelRequest, initReq?: fm.InitReq): Promise<Model> {
    return fm.fetchReq<GetModelRequest, Model>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static CreateModel(req: CreateModelRequest, initReq?: fm.InitReq): Promise<CreateModelResponse> {
    return fm.fetchReq<CreateModelRequest, CreateModelResponse>(`/api/v1alpha1/models`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteModel(req: DeleteModelRequest, initReq?: fm.InitReq): Promise<DeleteModelResponse> {
    return fm.fetchReq<DeleteModelRequest, DeleteModelResponse>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static ListModelRevisions(req: ListModelRevisionsRequest, initReq?: fm.InitReq): Promise<ListModelRevisionsResponse> {
    return fm.fetchReq<ListModelRevisionsRequest, ListModelRevisionsResponse>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}/revisions?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static ListModelCommits(req: ListModelCommitsRequest, initReq?: fm.InitReq): Promise<ListModelCommitsResponse> {
    return fm.fetchReq<ListModelCommitsRequest, ListModelCommitsResponse>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}/commits?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static GetModelCommit(req: GetModelCommitRequest, initReq?: fm.InitReq): Promise<Commit> {
    return fm.fetchReq<GetModelCommitRequest, Commit>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}/commits/${req["id"]}?${fm.renderURLSearchParams(req, ["project", "name", "id"])}`, {...initReq, method: "GET"})
  }
  static GetModelTree(req: GetModelTreeRequest, initReq?: fm.InitReq): Promise<GetModelTreeResponse> {
    return fm.fetchReq<GetModelTreeRequest, GetModelTreeResponse>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}/tree?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static GetModelBlob(req: GetModelBlobRequest, initReq?: fm.InitReq): Promise<File> {
    return fm.fetchReq<GetModelBlobRequest, File>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}/blob?${fm.renderURLSearchParams(req, ["project", "name"])}`, {...initReq, method: "GET"})
  }
  static UpdateModelSetting(req: UpdateModelSettingRequest, initReq?: fm.InitReq): Promise<UpdateModelSettingResponse> {
    return fm.fetchReq<UpdateModelSettingRequest, UpdateModelSettingResponse>(`/api/v1alpha1/models/${req["project"]}/${req["name"]}/setting`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
}