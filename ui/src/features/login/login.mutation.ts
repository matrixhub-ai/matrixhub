import { Login, type LoginRequest } from '@matrixhub/api-ts/v1alpha1/login.pb'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import type { NotificationMeta } from '@/types/tanstack-query'

export function loginMutationOptions() {
  return mutationOptions({
    mutationFn: (values: LoginRequest) => Login.Login(values),
    meta: {
      errorMessage: i18n.t('login.error'),
    } satisfies NotificationMeta,
  })
}

export function logoutMutationOptions() {
  return mutationOptions({
    mutationFn: () => Login.Logout({}),
    meta: {
      errorMessage: i18n.t('login.logoutError'),
    } satisfies NotificationMeta,
  })
}
