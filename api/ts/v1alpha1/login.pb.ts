/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type LoginRequest = {
  username?: string
  password?: string
}

export type LoginResponse = {
}

export class Login {
  static Login(req: LoginRequest, initReq?: fm.InitReq): Promise<LoginResponse> {
    return fm.fetchReq<LoginRequest, LoginResponse>(`/api/v1alpha1/login`, {...initReq, method: "POST"})
  }
}