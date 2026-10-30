// gRPC status codes, as returned in the `code` field of gateway error bodies.
export const GRPC_CODE = {
  invalidArgument: 3,
  deadlineExceeded: 4,
  notFound: 5,
  permissionDenied: 7,
  resourceExhausted: 8,
  failedPrecondition: 9,
  unavailable: 14,
  unauthenticated: 16,
} as const

/** The SDK throws the raw gateway error body (`{ code, message }`), not an `Error`. */
export function getSdkErrorCode(error: unknown): number | undefined {
  if (typeof error !== 'object' || error === null || !('code' in error)) {
    return undefined
  }

  const { code } = error as { code: unknown }

  if (typeof code !== 'number' && typeof code !== 'string') {
    return undefined
  }

  const numericCode = Number(code)

  return Number.isInteger(numericCode) ? numericCode : undefined
}
