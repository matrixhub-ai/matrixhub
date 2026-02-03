/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type ProjectItem = {
  name?: string
  displayName?: string
  description?: string
}

export type CreateProjectRequest = {
  name?: string
  displayName?: string
  description?: string
}

export type GetProjectRequest = {
  name?: string
}

export type ListProjectsRequest = {
  limit?: string
  cursor?: string
  orderBy?: string[]
}

export type ListProjects = {
  items?: ProjectItem[]
  count?: string
  nextCursor?: string
}

export class Projects {
  static Get(req: GetProjectRequest, initReq?: fm.InitReq): Promise<ProjectItem> {
    return fm.fetchReq<GetProjectRequest, ProjectItem>(`/api/v1alpha1/projects/${req["name"]}?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
  static Create(req: CreateProjectRequest, initReq?: fm.InitReq): Promise<ProjectItem> {
    return fm.fetchReq<CreateProjectRequest, ProjectItem>(`/api/v1alpha1/projects/${req["name"]}`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static List(req: ListProjectsRequest, initReq?: fm.InitReq): Promise<ListProjects> {
    return fm.fetchReq<ListProjectsRequest, ListProjects>(`/api/v1alpha1/projects?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
}