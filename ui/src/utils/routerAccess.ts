import { currentUserQueryOptions, projectRolesQueryOptions } from '@/features/auth/auth.query'
import { projectDetailQueryOptions } from '@/features/projects/projects.query'
import { queryClient } from '@/queryClient'

import type { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'

// ─── Error types ─────────────────────────────────────────────────────────────

export class ForbiddenRouteError extends Error {
  constructor(message = 'You do not have permission to access this page.') {
    super(message)
    this.name = 'ForbiddenRouteError'
    Object.setPrototypeOf(this, ForbiddenRouteError.prototype)
  }
}

export function isForbiddenRouteError(error: unknown): error is ForbiddenRouteError {
  return error instanceof ForbiddenRouteError
    || (error instanceof Error && error.name === 'ForbiddenRouteError')
}

export const forbiddenError = () => new ForbiddenRouteError()

export class NotFoundRouteError extends Error {
  constructor(message = 'The resource you are looking for does not exist.') {
    super(message)
    this.name = 'NotFoundRouteError'
    Object.setPrototypeOf(this, NotFoundRouteError.prototype)
  }
}

export function isNotFoundRouteError(error: unknown): error is NotFoundRouteError {
  return error instanceof NotFoundRouteError
    || (error instanceof Error && error.name === 'NotFoundRouteError')
}

export const notFoundError = () => new NotFoundRouteError()

// ─── SDK error helpers ────────────────────────────────────────────────────────

// SDK throws raw JSON body (not an Error instance); code 7 = PermissionDenied; message = "permission denied"
export function isSdkPermissionDenied(error: unknown): boolean {
  return (error as Record<string, unknown>)?.code === 7
}

// code 5 = NotFound; message = "not found"
export function isSdkNotFound(error: unknown): boolean {
  return (error as Record<string, unknown>)?.code === 5
}

// code 16 = Unauthenticated; http code = 401
export function isSdkUnauthenticated(error: unknown): boolean {
  return (error as Record<string, unknown>)?.code === 16
}

// ─── Access guard ─────────────────────────────────────────────────────────────

interface EnsureProjectAccessOptions {
  allowPublicRead?: boolean
  allowedRoles?: readonly ProjectRoleType[]
}

export async function ensureProjectAccess(
  projectId: string,
  options?: EnsureProjectAccessOptions,
) {
  const {
    allowPublicRead = false,
    allowedRoles,
  } = options ?? {}

  const [currentUser, projectRoles] = await Promise.all([
    queryClient.ensureQueryData(currentUserQueryOptions()),
    queryClient.ensureQueryData(projectRolesQueryOptions()),
  ])

  if (currentUser.isAdmin) {
    return
  }

  const role = projectRoles.projectRoles?.[projectId]

  if (!role) {
    await queryClient.ensureQueryData(projectDetailQueryOptions(projectId))

    // Public projects
    if (allowPublicRead) {
      return
    }

    // 403
    throw forbiddenError()
  }

  if (allowedRoles && !allowedRoles.includes(role)) {
    throw forbiddenError()
  }
}
