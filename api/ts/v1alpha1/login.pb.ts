/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type LoginRequest = {
  username?: string
  password?: string
  rememberMe?: boolean
}

export type LoginResponse = {
  cookie?: string
}

export type LogoutRequest = {
}

export type LogoutResponse = {
}

export class Login {
  static Login(req: LoginRequest, initReq?: fm.InitReq): Promise<LoginResponse> {
    return fm.fetchReq<LoginRequest, LoginResponse>(`/api/v1alpha1/login`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static Logout(req: LogoutRequest, initReq?: fm.InitReq): Promise<LogoutResponse> {
    return fm.fetchReq<LogoutRequest, LogoutResponse>(`/api/v1alpha1/logout`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
}