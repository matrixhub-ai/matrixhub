/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type User = {
  id?: string
  username?: string
  email?: string
}

export type CreateUserRequest = {
  username?: string
  password?: string
  email?: string
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
  pagination?: Pagination
}

export type Pagination = {
  total?: number
  page?: number
  pageSize?: number
}

export type GetUserRequest = {
  id?: string
}

export type GetUserResponse = {
  id?: string
  username?: string
  email?: string
}

export type DeleteUserRequest = {
  id?: string
}

export type DeleteUserResponse = {
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
}