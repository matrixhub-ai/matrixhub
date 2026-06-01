/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as GoogleProtobufTimestamp from "../google/protobuf/timestamp.pb"
import * as GoogleProtobufWrappers from "../google/protobuf/wrappers.pb"
import * as MatrixhubV1alpha1Role from "./role.pb"
import * as MatrixhubV1alpha1Utils from "./utils.pb"

export enum ProjectType {
  PROJECT_TYPE_UNSPECIFIED = "PROJECT_TYPE_UNSPECIFIED",
  PROJECT_TYPE_PRIVATE = "PROJECT_TYPE_PRIVATE",
  PROJECT_TYPE_PUBLIC = "PROJECT_TYPE_PUBLIC",
}

export enum ProjectPermissionFilter {
  PERMISSION_FILTER_UNSPECIFIED = "PERMISSION_FILTER_UNSPECIFIED",
  PERMISSION_FILTER_MANAGED_ONLY = "PERMISSION_FILTER_MANAGED_ONLY",
  PERMISSION_FILTER_CAN_WRITE = "PERMISSION_FILTER_CAN_WRITE",
  PERMISSION_FILTER_CAN_READ = "PERMISSION_FILTER_CAN_READ",
}

export enum MemberType {
  MEMBER_TYPE_USER = "MEMBER_TYPE_USER",
  MEMBER_TYPE_GROUP = "MEMBER_TYPE_GROUP",
}

export type CreateProjectRequest = {
  name?: string
  type?: ProjectType
  registryId?: GoogleProtobufWrappers.Int32Value
  organization?: string
}

export type CreateProjectResponse = {
}

export type ListProjectsRequest = {
  name?: string
  type?: ProjectType
  permissionFilter?: ProjectPermissionFilter
  page?: number
  pageSize?: number
}

export type ListProjectsResponse = {
  projects?: Project[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type GetProjectRequest = {
  name?: string
}

export type GetProjectResponse = {
  name?: string
  type?: ProjectType
  registryUrl?: string
  organization?: string
  modelCount?: number
  datasetCount?: number
  updatedAt?: GoogleProtobufTimestamp.Timestamp
}

export type DeleteProjectRequest = {
  name?: string
}

export type DeleteProjectResponse = {
}

export type UpdateProjectRequest = {
  name?: string
  type?: ProjectType
}

export type UpdateProjectResponse = {
}

export type Project = {
  name?: string
  type?: ProjectType
  enabledRegistry?: boolean
  modelCount?: number
  datasetCount?: number
  updatedAt?: GoogleProtobufTimestamp.Timestamp
}

export type ListProjectMembersRequest = {
  name?: string
  memberName?: string
  page?: number
  pageSize?: number
}

export type ListProjectMembersResponse = {
  members?: ProjectMember[]
  pagination?: MatrixhubV1alpha1Utils.Pagination
}

export type ProjectMember = {
  memberId?: number
  memberName?: string
  memberType?: MemberType
  role?: MatrixhubV1alpha1Role.ProjectRoleType
}

export type AddProjectMemberWithRoleRequest = {
  name?: string
  memberType?: MemberType
  memberId?: number
  role?: MatrixhubV1alpha1Role.ProjectRoleType
}

export type AddProjectMemberWithRoleResponse = {
}

export type RemoveProjectMembersRequest = {
  name?: string
  members?: MemberToRemove[]
}

export type MemberToRemove = {
  memberType?: MemberType
  memberId?: number
}

export type RemoveProjectMembersResponse = {
}

export type UpdateProjectMemberRoleRequest = {
  name?: string
  memberType?: MemberType
  memberId?: number
  role?: MatrixhubV1alpha1Role.ProjectRoleType
}

export type UpdateProjectMemberRoleResponse = {
}

export class Projects {
  static CreateProject(req: CreateProjectRequest, initReq?: fm.InitReq): Promise<CreateProjectResponse> {
    return fm.fetchReq<CreateProjectRequest, CreateProjectResponse>(`/api/v1alpha1/projects`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ListProjects(req: ListProjectsRequest, initReq?: fm.InitReq): Promise<ListProjectsResponse> {
    return fm.fetchReq<ListProjectsRequest, ListProjectsResponse>(`/api/v1alpha1/projects?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetProject(req: GetProjectRequest, initReq?: fm.InitReq): Promise<GetProjectResponse> {
    return fm.fetchReq<GetProjectRequest, GetProjectResponse>(`/api/v1alpha1/projects/${req["name"]}?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
  static UpdateProject(req: UpdateProjectRequest, initReq?: fm.InitReq): Promise<UpdateProjectResponse> {
    return fm.fetchReq<UpdateProjectRequest, UpdateProjectResponse>(`/api/v1alpha1/projects/${req["name"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteProject(req: DeleteProjectRequest, initReq?: fm.InitReq): Promise<DeleteProjectResponse> {
    return fm.fetchReq<DeleteProjectRequest, DeleteProjectResponse>(`/api/v1alpha1/projects/${req["name"]}`, {...initReq, method: "DELETE"})
  }
  static ListProjectMembers(req: ListProjectMembersRequest, initReq?: fm.InitReq): Promise<ListProjectMembersResponse> {
    return fm.fetchReq<ListProjectMembersRequest, ListProjectMembersResponse>(`/api/v1alpha1/projects/${req["name"]}/members?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
  static AddProjectMemberWithRole(req: AddProjectMemberWithRoleRequest, initReq?: fm.InitReq): Promise<AddProjectMemberWithRoleResponse> {
    return fm.fetchReq<AddProjectMemberWithRoleRequest, AddProjectMemberWithRoleResponse>(`/api/v1alpha1/projects/${req["name"]}/members`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static RemoveProjectMembers(req: RemoveProjectMembersRequest, initReq?: fm.InitReq): Promise<RemoveProjectMembersResponse> {
    return fm.fetchReq<RemoveProjectMembersRequest, RemoveProjectMembersResponse>(`/api/v1alpha1/projects/${req["name"]}/members`, {...initReq, method: "DELETE", body: JSON.stringify(req, fm.replacer)})
  }
  static UpdateProjectMemberRole(req: UpdateProjectMemberRoleRequest, initReq?: fm.InitReq): Promise<UpdateProjectMemberRoleResponse> {
    return fm.fetchReq<UpdateProjectMemberRoleRequest, UpdateProjectMemberRoleResponse>(`/api/v1alpha1/projects/${req["name"]}/members/${req["memberId"]}/role`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
}