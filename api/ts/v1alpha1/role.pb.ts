/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"

export enum ProjectRoleType {
  ROLE_TYPE_PROJECT_ADMIN = "ROLE_TYPE_PROJECT_ADMIN",
  ROLE_TYPE_PROJECT_EDITOR = "ROLE_TYPE_PROJECT_EDITOR",
  ROLE_TYPE_PROJECT_VIEWER = "ROLE_TYPE_PROJECT_VIEWER",
}

export type ListAllPermissionsRequest = {
}

export type ListAllPermissionsResponse = {
  systemCategories?: RoleCategory[]
  projectCategories?: RoleCategory[]
}

export type RoleCategory = {
  name?: string
  permissions?: Permission[]
}

export type Permission = {
  name?: string
  permission?: string
}

export class Roles {
  static ListAllPermissions(req: ListAllPermissionsRequest, initReq?: fm.InitReq): Promise<ListAllPermissionsResponse> {
    return fm.fetchReq<ListAllPermissionsRequest, ListAllPermissionsResponse>(`/api/v1alpha1/roles/permissions?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
}