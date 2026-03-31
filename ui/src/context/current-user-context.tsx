import { type GetCurrentUserResponse } from '@matrixhub/api-ts/v1alpha1/current_user.pb.ts'

import { createNamedContext } from '@/utils/createNamedContext'

export const CurrentUserContext = createNamedContext<GetCurrentUserResponse>('CurrentUserContext')
