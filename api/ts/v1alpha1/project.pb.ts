/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type GetProjectRequest = {
  name?: string
}

export type GetProjectResponse = {
  name?: string
}

export type CreateProjectRequest = {
  name?: string
}

export type CreateProjectResponse = {
}

export class Projects {
  static GetProject(req: GetProjectRequest, initReq?: fm.InitReq): Promise<GetProjectResponse> {
    return fm.fetchReq<GetProjectRequest, GetProjectResponse>(`/api/v1alpha1/projects/${req["name"]}?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
  static CreateProject(req: CreateProjectRequest, initReq?: fm.InitReq): Promise<CreateProjectResponse> {
    return fm.fetchReq<CreateProjectRequest, CreateProjectResponse>(`/api/v1alpha1/projects`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
}