/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as MatrixhubV1alpha1Utils from "./utils.pb"

export enum RobotAccountProjectScope {
  ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED = "ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED",
  ROBOT_ACCOUNT_PROJECT_SCOPE_ALL = "ROBOT_ACCOUNT_PROJECT_SCOPE_ALL",
}

export enum RobotAccountStatus {
  ROBOT_ACCOUNT_STATUS_DISABLED = "ROBOT_ACCOUNT_STATUS_DISABLED",
  ROBOT_ACCOUNT_STATUS_ENABLED = "ROBOT_ACCOUNT_STATUS_ENABLED",
}

export enum RobotAccountExpireStatus {
  ROBOT_ACCOUNT_EXPIRE_STATUS_EXPIRED = "ROBOT_ACCOUNT_EXPIRE_STATUS_EXPIRED",
  ROBOT_ACCOUNT_EXPIRE_STATUS_VALID = "ROBOT_ACCOUNT_EXPIRE_STATUS_VALID",
  ROBOT_ACCOUNT_EXPIRE_STATUS_NEVER = "ROBOT_ACCOUNT_EXPIRE_STATUS_NEVER",
}

export type CreateRobotAccountRequest = {
  name?: string
  description?: string
  expireDays?: number
  systemPermissions?: string[]
  projectPermissions?: string[]
  projects?: string[]
  projectScope?: RobotAccountProjectScope
}

export type CreateRobotAccountResponse = {
  token?: string
}

export type ListRobotAccountsRequest = {
  search?: string
  page?: number
  pageSize?: number
}

export type ListRobotAccountsResponse = {
  items?: RobotAccount[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type RobotAccount = {
  id?: number
  name?: string
  description?: string
  status?: RobotAccountStatus
  systemPermissions?: string[]
  projectPermissions?: string[]
  projects?: string[]
  createdAt?: string
  expireStatus?: RobotAccountExpireStatus
  remainPeriod?: string
  expireDays?: number
  projectScope?: RobotAccountProjectScope
}

export type GetRobotAccountRequest = {
  id?: number
}

export type GetRobotAccountResponse = {
  id?: number
  name?: string
  description?: string
  status?: RobotAccountStatus
  systemPermissions?: string[]
  projectPermissions?: string[]
  projects?: string[]
  createdAt?: string
  expireStatus?: RobotAccountExpireStatus
  remainPeriod?: string
  expireDays?: number
  projectScope?: RobotAccountProjectScope
}

export type DeleteRobotAccountRequest = {
  id?: number
}

export type DeleteRobotAccountResponse = {
}

export type UpdateRobotAccountRequest = {
  id?: number
  description?: string
  status?: RobotAccountStatus
  systemPermissions?: string[]
  projectPermissions?: string[]
  projects?: string[]
  projectScope?: RobotAccountProjectScope
  expireDays?: number
}

export type UpdateRobotAccountResponse = {
}

export type RefreshRobotAccountTokenRequest = {
  id?: number
  autoGenerate?: boolean
  token?: string
}

export type RefreshRobotAccountTokenResponse = {
  token?: string
}

export class Robots {
  static CreateRobotAccount(req: CreateRobotAccountRequest, initReq?: fm.InitReq): Promise<CreateRobotAccountResponse> {
    return fm.fetchReq<CreateRobotAccountRequest, CreateRobotAccountResponse>(`/api/v1alpha1/robot-accounts`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ListRobotAccounts(req: ListRobotAccountsRequest, initReq?: fm.InitReq): Promise<ListRobotAccountsResponse> {
    return fm.fetchReq<ListRobotAccountsRequest, ListRobotAccountsResponse>(`/api/v1alpha1/robot-accounts?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetRobotAccount(req: GetRobotAccountRequest, initReq?: fm.InitReq): Promise<GetRobotAccountResponse> {
    return fm.fetchReq<GetRobotAccountRequest, GetRobotAccountResponse>(`/api/v1alpha1/robot-accounts/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, {...initReq, method: "GET"})
  }
  static DeleteRobotAccount(req: DeleteRobotAccountRequest, initReq?: fm.InitReq): Promise<DeleteRobotAccountResponse> {
    return fm.fetchReq<DeleteRobotAccountRequest, DeleteRobotAccountResponse>(`/api/v1alpha1/robot-accounts/${req["id"]}`, {...initReq, method: "DELETE"})
  }
  static UpdateRobotAccount(req: UpdateRobotAccountRequest, initReq?: fm.InitReq): Promise<UpdateRobotAccountResponse> {
    return fm.fetchReq<UpdateRobotAccountRequest, UpdateRobotAccountResponse>(`/api/v1alpha1/robot-accounts/${req["id"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static RefreshRobotAccountToken(req: RefreshRobotAccountTokenRequest, initReq?: fm.InitReq): Promise<RefreshRobotAccountTokenResponse> {
    return fm.fetchReq<RefreshRobotAccountTokenRequest, RefreshRobotAccountTokenResponse>(`/api/v1alpha1/robot-accounts/${req["id"]}`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
}