/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as MatrixhubV1alpha1Role from "./role.pb"

export enum AccessTokenStatus {
  ACCESS_TOKEN_STATUS_UNKNOWN = "ACCESS_TOKEN_STATUS_UNKNOWN",
  ACCESS_TOKEN_STATUS_VALID = "ACCESS_TOKEN_STATUS_VALID",
  ACCESS_TOKEN_STATUS_EXPIRED = "ACCESS_TOKEN_STATUS_EXPIRED",
}

export type ResetPasswordRequest = {
  oldPassword?: string
  newPassword?: string
}

export type ResetPasswordResponse = {
}

export type ListAccessTokensRequest = {
}

export type ListAccessTokensResponse = {
  items?: AccessToken[]
}

export type AccessToken = {
  id?: number
  name?: string
  status?: AccessTokenStatus
  createdAt?: string
  expiredAt?: string
}

export type CreateAccessTokenRequest = {
  name?: string
  expireAt?: string
}

export type CreateAccessTokenResponse = {
  token?: string
}

export type DeleteAccessTokenRequest = {
  id?: number
}

export type DeleteAccessTokenResponse = {
}

export type GetProjectRolesRequest = {
}

export type GetProjectRolesResponse = {
  projectRoles?: {[key: string]: MatrixhubV1alpha1Role.ProjectRoleType}
}

export type GetCurrentUserRequest = {
}

export type GetCurrentUserResponse = {
  id?: number
  username?: string
  isAdmin?: boolean
}

export type CreateSSHKeyRequest = {
  sshKeyName?: string
  publicKey?: string
  expiredAt?: string
}

export type CreateSSHKeyResponse = {
}

export type UpdateSSHKeyRequest = {
  sshKeyId?: number
  sshKeyName?: string
  publicKey?: string
}

export type UpdateSSHKeyResponse = {
}

export type DeleteSSHKeyRequest = {
  sshKeyId?: number
}

export type DeleteSSHKeyResponse = {
}

export type ListSSHKeysRequest = {
}

export type ListSSHKeysResponse = {
  items?: SSHKey[]
}

export type SSHKey = {
  id?: number
  sshKeyName?: string
  publicKey?: string
  updatedAt?: string
  createdAt?: string
  expiredAt?: string
}

export class CurrentUser {
  static GetCurrentUser(req: GetCurrentUserRequest, initReq?: fm.InitReq): Promise<GetCurrentUserResponse> {
    return fm.fetchReq<GetCurrentUserRequest, GetCurrentUserResponse>(`/api/v1alpha1/current-user?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static ResetPassword(req: ResetPasswordRequest, initReq?: fm.InitReq): Promise<ResetPasswordResponse> {
    return fm.fetchReq<ResetPasswordRequest, ResetPasswordResponse>(`/api/v1alpha1/current-user/reset-password`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ListAccessTokens(req: ListAccessTokensRequest, initReq?: fm.InitReq): Promise<ListAccessTokensResponse> {
    return fm.fetchReq<ListAccessTokensRequest, ListAccessTokensResponse>(`/api/v1alpha1/current-user/access-tokens?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static CreateAccessToken(req: CreateAccessTokenRequest, initReq?: fm.InitReq): Promise<CreateAccessTokenResponse> {
    return fm.fetchReq<CreateAccessTokenRequest, CreateAccessTokenResponse>(`/api/v1alpha1/current-user/access-tokens`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteAccessToken(req: DeleteAccessTokenRequest, initReq?: fm.InitReq): Promise<DeleteAccessTokenResponse> {
    return fm.fetchReq<DeleteAccessTokenRequest, DeleteAccessTokenResponse>(`/api/v1alpha1/current-user/access-tokens/${req["id"]}`, {...initReq, method: "DELETE"})
  }
  static GetProjectRoles(req: GetProjectRolesRequest, initReq?: fm.InitReq): Promise<GetProjectRolesResponse> {
    return fm.fetchReq<GetProjectRolesRequest, GetProjectRolesResponse>(`/api/v1alpha1/current-user/projects/role?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static CreateSSHKey(req: CreateSSHKeyRequest, initReq?: fm.InitReq): Promise<CreateSSHKeyResponse> {
    return fm.fetchReq<CreateSSHKeyRequest, CreateSSHKeyResponse>(`/api/v1alpha1/current-user/ssh-keys`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static UpdateSSHKey(req: UpdateSSHKeyRequest, initReq?: fm.InitReq): Promise<UpdateSSHKeyResponse> {
    return fm.fetchReq<UpdateSSHKeyRequest, UpdateSSHKeyResponse>(`/api/v1alpha1/current-user/ssh-keys/${req["sshKeyId"]}`, {...initReq, method: "PUT", body: JSON.stringify(req, fm.replacer)})
  }
  static DeleteSSHKey(req: DeleteSSHKeyRequest, initReq?: fm.InitReq): Promise<DeleteSSHKeyResponse> {
    return fm.fetchReq<DeleteSSHKeyRequest, DeleteSSHKeyResponse>(`/api/v1alpha1/current-user/ssh-keys/${req["sshKeyId"]}`, {...initReq, method: "DELETE"})
  }
  static ListSSHKeys(req: ListSSHKeysRequest, initReq?: fm.InitReq): Promise<ListSSHKeysResponse> {
    return fm.fetchReq<ListSSHKeysRequest, ListSSHKeysResponse>(`/api/v1alpha1/current-user/ssh-keys?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
}