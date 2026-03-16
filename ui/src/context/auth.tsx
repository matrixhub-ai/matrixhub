import { type GetCurrentUserResponse } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { createContext } from 'react'

export interface AuthContextValue {
  user: GetCurrentUserResponse | null
  loading: boolean
  refetch: () => void
}

export const AuthContext = createContext<AuthContextValue>({
  user: null,
  loading: true,
  refetch: () => null,
})
