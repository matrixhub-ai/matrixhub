import {
  CurrentUser, type GetCurrentUserResponse, type GetProjectRolesResponse,
} from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { Projects, ProjectType } from '@matrixhub/api-ts/v1alpha1/project.pb'

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

// ─── Auth cache ───────────────────────────────────────────────────────────────
// Caches in-flight and resolved promises for 30 seconds so that multiple
// beforeLoad / loader hooks in the same navigation share the same request.

const CACHE_TTL = 30_000

interface CacheEntry<T> {
  promise: Promise<T>
  expiresAt: number
}

let userCache: CacheEntry<GetCurrentUserResponse> | null = null
let rolesCache: CacheEntry<GetProjectRolesResponse['projectRoles']> | null = null

export function getCachedUser() {
  if (!userCache || Date.now() > userCache.expiresAt) {
    const promise = CurrentUser.GetCurrentUser({})
      .catch((err) => {
        userCache = null
        throw err
      })

    userCache = {
      promise,
      expiresAt: Date.now() + CACHE_TTL,
    }
  }

  return userCache.promise
}

export function getCachedProjectRoles() {
  if (!rolesCache || Date.now() > rolesCache.expiresAt) {
    const promise = CurrentUser.GetProjectRoles({})
      .then(r => r.projectRoles ?? {})
      .catch((err) => {
        rolesCache = null
        throw err
      })

    rolesCache = {
      promise,
      expiresAt: Date.now() + CACHE_TTL,
    }
  }

  return rolesCache.promise
}

/** Call after login / logout to force fresh data on next access. */
export function invalidateAuthCache() {
  userCache = null
  rolesCache = null
}

// ─── Access guard ─────────────────────────────────────────────────────────────

export async function ensureProjectAccess(
  projectId: string,
  allowedRoles?: readonly ProjectRoleType[],
) {
  const [currentUser, projectRoles] = await Promise.all([
    getCachedUser(),
    getCachedProjectRoles(),
  ])

  if (currentUser.isAdmin) {
    return
  }

  const role = projectRoles?.[projectId]

  if (!role) {
    // Determine whether to show 403 (public project) or 404 (private project).
    const project = await Projects.GetProject({ name: projectId })

    if (project.type === ProjectType.PROJECT_TYPE_PUBLIC) {
      // 403
      throw new ForbiddenRouteError()
    } else {
      // 404
      throw new NotFoundRouteError()
    }
  }

  if (allowedRoles && !allowedRoles.includes(role)) {
    throw new ForbiddenRouteError()
  }
}
