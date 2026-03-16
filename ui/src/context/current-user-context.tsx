import { type GetCurrentUserResponse } from '@matrixhub/api-ts/v1alpha1/current_user.pb.ts'
import { createContext } from 'react'

export const CurrentUserContext = createContext<GetCurrentUserResponse | undefined>(undefined)
