import {
  setFetchFn,
  type FetchFn,
} from '@matrixhub/api-ts/fetch.pb'

import i18n, {
  DEFAULT_LANGUAGE,
  normalizeLanguage,
} from '@/i18n'

const API_PATH_PREFIXES = ['/api', '/apis']

type FetchInput = Parameters<typeof window.fetch>[0]

type FetchInit = Parameters<typeof window.fetch>[1]

type MatrixHubWindow = Window & typeof globalThis & {
  __matrixhubApiRequestHeadersInstalled?: boolean
}

export type ApiRequestHeaderProvider = () => HeadersInit | null | undefined

// Keep cross-cutting SDK headers in one provider list so future headers can be
// added here without wrapping every generated SDK call.
const apiRequestHeaderProviders: ApiRequestHeaderProvider[] = [
  () => ({
    'Accept-Language': currentAcceptLanguage(),
  }),
]

function currentAcceptLanguage() {
  return (
    normalizeLanguage(i18n.resolvedLanguage)
    ?? normalizeLanguage(i18n.language)
    ?? DEFAULT_LANGUAGE
  )
}

function isRequest(input: FetchInput): input is Request {
  return typeof Request !== 'undefined' && input instanceof Request
}

function requestUrl(input: FetchInput) {
  try {
    return new URL(isRequest(input) ? input.url : input, window.location.origin)
  } catch {
    return null
  }
}

function isApiRequest(input: FetchInput) {
  const url = requestUrl(input)

  if (!url) {
    return false
  }

  return API_PATH_PREFIXES.some(prefix => (
    url.pathname === prefix || url.pathname.startsWith(`${prefix}/`)
  ))
}

function mergeRequestHeaders(input: FetchInput, init?: FetchInit) {
  const headers = new Headers(isRequest(input) ? input.headers : undefined)

  if (init?.headers) {
    new Headers(init.headers).forEach((value, key) => {
      headers.set(key, value)
    })
  }

  apiRequestHeaderProviders.forEach((provider) => {
    const providerHeaders = provider()

    if (!providerHeaders) {
      return
    }

    new Headers(providerHeaders).forEach((value, key) => {
      // Call-site headers are more specific than global defaults.
      if (!headers.has(key)) {
        headers.set(key, value)
      }
    })
  })

  return headers
}

function withApiRequestHeaders(input: FetchInput, init?: FetchInit) {
  return {
    ...init,
    headers: mergeRequestHeaders(input, init),
  }
}

// Configure the generated SDK fetch source instead of patching window.fetch,
// so this only affects SDK requests.
export function createApiRequestFetch(baseFetch: FetchFn): FetchFn {
  return (input, init) => {
    if (!isApiRequest(input)) {
      return baseFetch(input, init)
    }

    return baseFetch(input, withApiRequestHeaders(input, init))
  }
}

export function registerApiRequestHeaderProvider(provider: ApiRequestHeaderProvider) {
  apiRequestHeaderProviders.push(provider)
}

export function installApiRequestHeaders() {
  if (typeof window === 'undefined') {
    return
  }

  const matrixHubWindow = window as MatrixHubWindow

  // Keep install idempotent for dev reloads and repeated app bootstrap paths.
  if (matrixHubWindow.__matrixhubApiRequestHeadersInstalled) {
    return
  }

  setFetchFn(createApiRequestFetch(matrixHubWindow.fetch.bind(matrixHubWindow)))
  matrixHubWindow.__matrixhubApiRequestHeadersInstalled = true
}
