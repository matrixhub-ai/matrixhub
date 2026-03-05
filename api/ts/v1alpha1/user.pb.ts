/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as GoogleProtobufTimestamp from "../google/protobuf/timestamp.pb"
import * as MatrixhubV1alpha1Utils from "./utils.pb"

export enum UserSource {
  USER_SOURCE_UNSPECIFIED = "USER_SOURCE_UNSPECIFIED",
  USER_SOURCE_LOCAL = "USER_SOURCE_LOCAL",
}

export type User = {
  id?: string
  username?: string
  email?: string
  isAdmin?: boolean
  source?: UserSource
  createdAt?: GoogleProtobufTimestamp.Timestamp
}

export type CreateUserRequest = {
  username?: string
  password?: string
  isAdmin?: boolean
}

export type CreateUserResponse = {
}

export type ListUsersRequest = {
  page?: number
  pageSize?: number
  search?: string
}

export type ListUsersResponse = {
  users?: User[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type GetUserRequest = {
  id?: string
}

export type GetUserResponse = {
  id?: string
  username?: string
  email?: string
  isAdmin?: boolean
  source?: UserSource
  createdAt?: GoogleProtobufTimestamp.Timestamp
}

export type DeleteUserRequest = {
  id?: string
}

export type DeleteUserResponse = {
}

export type SetUserSysAdminRequest = {
  id?: string
  sysadminFlag?: boolean
}

export type SetUserSysAdminResponse = {
}

export type ResetUserPasswordRequest = {
  id?: string
  password?: string
}

export type ResetUserPasswordResponse = {
}

export class Users {
  static ListUsers(req: ListUsersRequest, initReq?: fm.InitReq): Promise<ListUsersResponse> {
    return fm.fetchReq<ListUsersRequest, ListUsersResponse>(`/api/v1alpha1/users?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static CreateUser(req: CreateUserRequest, initReq?: fm.InitReq): Promise<CreateUserResponse> {
    return fm.fetchReq<CreateUserRequest, CreateUserResponse>(`/api/v1alpha1/users`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetUser(req: GetUserRequest, initReq?: fm.InitReq): Promise<GetUserResponse> {
    return fm.fetchReq<GetUserRequest, GetUserResponse>(`/api/v1alpha1/users/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, {...initReq, method: "GET"})
  }
  static DeleteUser(req: DeleteUserRequest, initReq?: fm.InitReq): Promise<DeleteUserResponse> {
    return fm.fetchReq<DeleteUserRequest, DeleteUserResponse>(`/api/v1alpha1/users/${req["id"]}`, {...initReq, method: "DELETE"})
  }
  static ResetUserPassword(req: ResetUserPasswordRequest, initReq?: fm.InitReq): Promise<ResetUserPasswordResponse> {
    return fm.fetchReq<ResetUserPasswordRequest, ResetUserPasswordResponse>(`/api/v1alpha1/users/${req["id"]}/reset-password`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static SetUserSysAdmin(req: SetUserSysAdminRequest, initReq?: fm.InitReq): Promise<SetUserSysAdminResponse> {
    return fm.fetchReq<SetUserSysAdminRequest, SetUserSysAdminResponse>(`/apis/v1alpha1/users/${req["id"]}/sysadmin`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
}