/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as GoogleProtobufTimestamp from "../google/protobuf/timestamp.pb"
import * as MatrixhubV1alpha1Utils from "./utils.pb"

type Absent<T, K extends keyof T> = { [k in Exclude<keyof T, K>]?: undefined };
type OneOf<T> =
  | { [k in keyof T]?: undefined }
  | (
    keyof T extends infer K ?
      (K extends string & keyof T ? { [k in K]: T[K] } & Absent<T, K>
        : never)
    : never);

export enum RegistryType {
  REGISTRY_TYPE_UNSPECIFIED = "REGISTRY_TYPE_UNSPECIFIED",
  REGISTRY_TYPE_HUGGINGFACE = "REGISTRY_TYPE_HUGGINGFACE",
  REGISTRY_TYPE_MATRIXHUB = "REGISTRY_TYPE_MATRIXHUB",
}

export enum RegistryStatus {
  REGISTRY_STATUS_UNSPECIFIED = "REGISTRY_STATUS_UNSPECIFIED",
  REGISTRY_STATUS_HEALTHY = "REGISTRY_STATUS_HEALTHY",
  REGISTRY_STATUS_UNHEALTHY = "REGISTRY_STATUS_UNHEALTHY",
}


type BaseRegistry = {
  id?: number
  name?: string
  description?: string
  type?: RegistryType
  url?: string
  insecure?: boolean
  status?: RegistryStatus
  createdAt?: GoogleProtobufTimestamp.Timestamp
  updatedAt?: GoogleProtobufTimestamp.Timestamp
}

export type Registry = BaseRegistry
  & OneOf<{ basic: RegistryBasicCredential }>

export type RegistryBasicCredential = {
  username?: string
  password?: string
}

export type ListRegistriesRequest = {
  page?: number
  pageSize?: number
  search?: string
}

export type ListRegistriesResponse = {
  registries?: Registry[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type GetRegistryRequest = {
  id?: number
}

export type GetRegistryResponse = {
  registry?: Registry
}


type BaseCreateRegistryRequest = {
  name?: string
  description?: string
  type?: RegistryType
  url?: string
  insecure?: boolean
}

export type CreateRegistryRequest = BaseCreateRegistryRequest
  & OneOf<{ basic: RegistryBasicCredential }>

export type CreateRegistryResponse = {
  registry?: Registry
}


type BaseUpdateRegistryRequest = {
  id?: number
  name?: string
  description?: string
  url?: string
  insecure?: boolean
}

export type UpdateRegistryRequest = BaseUpdateRegistryRequest
  & OneOf<{ basic: RegistryBasicCredential }>

export type UpdateRegistryResponse = {
  registry?: Registry
}

export type DeleteRegistryRequest = {
  id?: number
}

export type DeleteRegistryResponse = {
}


type BasePingRegistryRequest = {
  type?: RegistryType
  url?: string
  insecure?: boolean
}

export type PingRegistryRequest = BasePingRegistryRequest
  & OneOf<{ basic: RegistryBasicCredential }>

export type PingRegistryResponse = {
}

export class Registries {
  static ListRegistries(req: ListRegistriesRequest, initReq?: fm.InitReq): Promise<ListRegistriesResponse> {
    return fm.fetchReq<ListRegistriesRequest, ListRegistriesResponse>(`/api/v1alpha1/registries?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetRegistry(req: GetRegistryRequest, initReq?: fm.InitReq): Promise<GetRegistryResponse> {
    return fm.fetchReq<GetRegistryRequest, GetRegistryResponse>(`/api/v1alpha1/registries/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, {...initReq, method: "GET"})
  }
  static CreateRegistry(req: CreateRegistryRequest, initReq?: fm.InitReq): Promise<CreateRegistryResponse> {
    return fm.fetchReq<CreateRegistryRequest, CreateRegistryResponse>(`/api/v1alpha1/registries`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static UpdateRegistry(req: UpdateRegistryRequest, initReq?: fm.InitReq): Promise<UpdateRegistryResponse> {
    return fm.fetchReq<UpdateRegistryRequest, UpdateRegistryResponse>(`/api/v1alpha1/registries/${req["id"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteRegistry(req: DeleteRegistryRequest, initReq?: fm.InitReq): Promise<DeleteRegistryResponse> {
    return fm.fetchReq<DeleteRegistryRequest, DeleteRegistryResponse>(`/api/v1alpha1/registries/${req["id"]}`, {...initReq, method: "DELETE"})
  }
  static PingRegistry(req: PingRegistryRequest, initReq?: fm.InitReq): Promise<PingRegistryResponse> {
    return fm.fetchReq<PingRegistryRequest, PingRegistryResponse>(`/api/v1alpha1/registries/ping`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
}