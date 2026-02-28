/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
export type ResetPasswordRequest = {
  password?: string
}

export type ResetPasswordResponse = {
}

export type ListAccessTokensRequest = {
}

export type ListAccessTokensResponse = {
  items?: AccessToken[]
}

export type AccessToken = {
  id?: string
  name?: string
  updatedAt?: string
  createdAt?: string
  expiredAt?: string
}

export type CreateAccessTokenRequest = {
  name?: string
  expiredAt?: string
}

export type CreateAccessTokenResponse = {
}

export type DeleteAccessTokenRequest = {
  id?: string
}

export type DeleteAccessTokenResponse = {
}

export class CurrentUser {
  static ResetPassword(req: ResetPasswordRequest, initReq?: fm.InitReq): Promise<ResetPasswordResponse> {
    return fm.fetchReq<ResetPasswordRequest, ResetPasswordResponse>(`/api/v1alpha1/current-user/reset-password`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ListAccessTokens(req: ListAccessTokensRequest, initReq?: fm.InitReq): Promise<ListAccessTokensResponse> {
    return fm.fetchReq<ListAccessTokensRequest, ListAccessTokensResponse>(`/api/v1alpha1/current-user/access-tokens?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static CreateAccessToken(req: CreateAccessTokenRequest, initReq?: fm.InitReq): Promise<CreateAccessTokenResponse> {
    return fm.fetchReq<CreateAccessTokenRequest, CreateAccessTokenResponse>(`/apis/v1alpha1/current-user/access-tokens`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteAccessToken(req: DeleteAccessTokenRequest, initReq?: fm.InitReq): Promise<DeleteAccessTokenResponse> {
    return fm.fetchReq<DeleteAccessTokenRequest, DeleteAccessTokenResponse>(`/apis/v1alpha1/current-user/access-tokens/${req["id"]}`, {...initReq, method: "DELETE"})
  }
}