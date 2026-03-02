/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type CLILoginRequest = {
}

export type CLILoginResponse = {
}

export class HfCompatibility {
  static CLILogin(req: CLILoginRequest, initReq?: fm.InitReq): Promise<CLILoginResponse> {
    return fm.fetchReq<CLILoginRequest, CLILoginResponse>(`/api/whoami-v2?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
}