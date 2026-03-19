// Service Worker for proxying /api requests to a configurable backend.
// Only registered in dev-deploy mode (VITE_DEV_DEPLOY=true).

let apiTarget = null

function buildTargetUrl(requestUrl, target) {
  const sourceUrl = new URL(requestUrl)
  const targetUrl = new URL(target)
  const basePath = targetUrl.pathname === '/'
    ? ''
    : targetUrl.pathname.replace(/\/+$/, '')

  targetUrl.pathname = `${basePath}${sourceUrl.pathname}`.replace(/\/{2,}/g, '/')
  targetUrl.search = sourceUrl.search
  targetUrl.hash = ''

  return targetUrl
}

self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

self.addEventListener('message', (event) => {
  if (event.data?.type === 'SET_API_TARGET') {
    apiTarget = event.data.target
  }
})

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url)

  if (!apiTarget || !url.pathname.startsWith('/api')) {
    return
  }

  const targetUrl = buildTargetUrl(event.request.url, apiTarget)

  const headers = new Headers(event.request.headers)
  headers.delete('host')
  headers.delete('cookie')

  const proxyRequest = new Request(targetUrl.toString(), {
    method: event.request.method,
    headers,
    body: event.request.body,
    mode: 'cors',
    credentials: 'omit',
    duplex: event.request.body ? 'half' : undefined,
  })

  event.respondWith(
    fetch(proxyRequest).catch((err) => {
      return new Response(JSON.stringify({
        error: 'API proxy failed',
        detail: err.message,
      }), {
        status: 502,
        headers: { 'Content-Type': 'application/json' },
      })
    }),
  )
})
