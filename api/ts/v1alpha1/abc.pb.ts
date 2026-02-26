/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type GetAbcRequest = {
  name?: string
}

export type GetAbcResponse = {
  name?: string
}

export class Abcs {
  static GetAbc(req: GetAbcRequest, initReq?: fm.InitReq): Promise<GetAbcResponse> {
    return fm.fetchReq<GetAbcRequest, GetAbcResponse>(`/api/v1alpha1/abc/${req["name"]}?${fm.renderURLSearchParams(req, ["name"])}`, {...initReq, method: "GET"})
  }
}