// Service Worker for proxying /api requests to a configurable backend.
// Only registered in dev-deploy mode (VITE_DEV_DEPLOY=true).

let apiTarget = null

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
  const targetUrl = new URL(url.pathname + url.search, apiTarget)

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
