import { CurrentUser } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { queryOptions, useQuery } from '@tanstack/react-query'

export const profileKeys = {
  accessTokens: ['access-tokens'] as const,
  sshKeys: ['ssh-keys'] as const,
}

export function accessTokensQueryOptions() {
  return queryOptions({
    queryKey: profileKeys.accessTokens,
    queryFn: () => CurrentUser.ListAccessTokens({}),
  })
}

export function sshKeysQueryOptions() {
  return queryOptions({
    queryKey: profileKeys.sshKeys,
    queryFn: () => CurrentUser.ListSSHKeys({}),
  })
}

export function useAccessTokens() {
  return useQuery(accessTokensQueryOptions())
}
