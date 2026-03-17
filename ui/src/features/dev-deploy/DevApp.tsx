/* eslint-disable @eslint-react/hooks-extra/no-direct-set-state-in-use-effect */
/* eslint-disable @eslint-react/web-api/no-leaked-event-listener */
/* eslint-disable react-refresh/only-export-components */
import { MantineProvider } from '@mantine/core'
import { RouterProvider } from '@tanstack/react-router'
import {
  StrictMode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import { createRoot } from 'react-dom/client'

import '@mantine/core/styles.css'
import '../../index.css'
import { mantineTheme } from '../../mantineTheme'
import { router } from '../../router'

const STORAGE_KEY = 'dev-api-proxy-target'

function useServiceWorker() {
  const swRef = useRef<ServiceWorker | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    navigator.serviceWorker
      .register('/api-proxy-sw.js')
      .then((reg) => {
        const sw = reg.active || reg.installing || reg.waiting

        if (sw) {
          if (sw.state === 'activated') {
            swRef.current = sw
            setReady(true)
          } else {
            sw.addEventListener('statechange', () => {
              if (sw.state === 'activated') {
                swRef.current = sw
                setReady(true)
              }
            })
          }
        }
      })
  }, [])

  const sendTarget = useCallback((target: string) => {
    const sw = swRef.current ?? navigator.serviceWorker.controller

    sw?.postMessage({
      type: 'SET_API_TARGET',
      target,
    })
  }, [])

  return {
    ready,
    sendTarget,
  }
}

const modalOverlay: React.CSSProperties = {
  position: 'fixed',
  inset: 0,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  background: 'rgba(0,0,0,0.5)',
  zIndex: 10000,
}

const modalBox: React.CSSProperties = {
  background: '#fff',
  borderRadius: 12,
  padding: 32,
  minWidth: 400,
  maxWidth: 480,
  boxShadow: '0 8px 32px rgba(0,0,0,0.2)',
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 12px',
  fontSize: 14,
  border: '1px solid #ddd',
  borderRadius: 8,
  outline: 'none',
  boxSizing: 'border-box',
}

function btnStyle(enabled: boolean): React.CSSProperties {
  return {
    marginTop: 12,
    width: '100%',
    padding: '10px 0',
    fontSize: 14,
    fontWeight: 600,
    color: '#fff',
    background: enabled ? '#228be6' : '#aaa',
    border: 'none',
    borderRadius: 8,
    cursor: enabled ? 'pointer' : 'not-allowed',
  }
}

function ConfigModal({
  defaultValue,
  onSubmit,
}: {
  defaultValue: string
  onSubmit: (url: string) => void
}) {
  const [value, setValue] = useState(defaultValue)

  return (
    <div style={modalOverlay}>
      <div style={modalBox}>
        <h2 style={{
          margin: '0 0 8px',
          fontSize: 20,
        }}
        >
          Dev Deploy Configuration
        </h2>
        <p style={{
          margin: '0 0 20px',
          color: '#666',
          fontSize: 14,
        }}
        >
          Enter the backend API server URL to proxy
          {' '}
          <code>/api</code>
          {' '}
          requests.
        </p>
        <input
          type="url"
          placeholder="https://192.168.1.100:4443"
          value={value}
          onChange={e => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && value.trim()) {
              onSubmit(value.trim())
            }
          }}
          style={inputStyle}
        />
        <button
          type="button"
          disabled={!value.trim()}
          onClick={() => onSubmit(value.trim())}
          style={btnStyle(!!value.trim())}
        >
          Connect
        </button>
      </div>
    </div>
  )
}

function CertTrustPrompt({
  target,
  onRetry,
}: {
  target: string
  onRetry: () => void
}) {
  return (
    <div style={modalOverlay}>
      <div style={modalBox}>
        <h2 style={{
          margin: '0 0 8px',
          fontSize: 20,
        }}
        >
          Certificate Not Trusted
        </h2>
        <p style={{
          margin: '0 0 8px',
          color: '#666',
          fontSize: 14,
        }}
        >
          Cannot reach
          {' '}
          <code>{target}</code>
          . If using a self-signed certificate, open the link below and accept it in your browser.
        </p>
        <button
          type="button"
          onClick={() => window.open(`${target}/api`, '_blank')}
          style={{
            ...btnStyle(true),
            background: '#f59f00',
          }}
        >
          Open Backend to Trust Certificate
        </button>
        <button
          type="button"
          onClick={onRetry}
          style={btnStyle(true)}
        >
          Done, Retry Connection
        </button>
      </div>
    </div>
  )
}

function DevToolbar({
  target, onEdit,
}: {
  target: string
  onEdit: () => void
}) {
  return (
    <div style={{
      position: 'fixed',
      bottom: 0,
      left: 0,
      right: 0,
      background: '#1a1b1e',
      color: '#c1c2c5',
      padding: '6px 16px',
      fontSize: 12,
      display: 'flex',
      alignItems: 'center',
      gap: 12,
      zIndex: 10000,
      fontFamily: 'monospace',
    }}
    >
      <span style={{
        color: '#51cf66',
        fontWeight: 700,
      }}
      >
        DEV
      </span>
      <span>
        API →
        {' '}
        {target}
      </span>
      <button
        type="button"
        onClick={onEdit}
        style={{
          marginLeft: 'auto',
          background: 'transparent',
          border: '1px solid #555',
          color: '#c1c2c5',
          borderRadius: 4,
          padding: '2px 10px',
          cursor: 'pointer',
          fontSize: 12,
        }}
      >
        Edit
      </button>
    </div>
  )
}

type Phase = 'loading' | 'config' | 'cert-check' | 'cert-untrusted' | 'ready'

function DevAppRoot() {
  const {
    ready: swReady, sendTarget,
  } = useServiceWorker()
  const [apiTarget, setApiTarget] = useState(
    () => localStorage.getItem(STORAGE_KEY) || '',
  )
  const [phase, setPhase] = useState<Phase>('loading')

  // Determine initial phase once SW is ready
  useEffect(() => {
    if (!swReady) {
      return
    }

    if (!apiTarget) {
      setPhase('config')
    } else {
      applyTarget(apiTarget)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [swReady])

  function applyTarget(target: string) {
    localStorage.setItem(STORAGE_KEY, target)
    sendTarget(target)
    setApiTarget(target)
    // Check connectivity
    setPhase('cert-check')
    fetch(target, { mode: 'no-cors' })
      .then(() => setPhase('ready'))
      .catch(() => {
        if (target.startsWith('https')) {
          setPhase('cert-untrusted')
        } else {
          // HTTP backend unreachable - still proceed, will fail on API calls
          setPhase('ready')
        }
      })
  }

  if (phase === 'loading') {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        color: '#666',
      }}
      >
        Initializing service worker...
      </div>
    )
  }

  if (phase === 'config') {
    return (
      <MantineProvider theme={mantineTheme}>
        <ConfigModal
          defaultValue={apiTarget}
          onSubmit={applyTarget}
        />
      </MantineProvider>
    )
  }

  if (phase === 'cert-check') {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        color: '#666',
      }}
      >
        Checking backend connectivity...
      </div>
    )
  }

  if (phase === 'cert-untrusted') {
    return (
      <MantineProvider theme={mantineTheme}>
        <CertTrustPrompt
          target={apiTarget}
          onRetry={() => applyTarget(apiTarget)}
        />
      </MantineProvider>
    )
  }

  return (
    <StrictMode>
      <MantineProvider theme={mantineTheme}>
        <RouterProvider router={router} />
        <DevToolbar target={apiTarget} onEdit={() => setPhase('config')} />
      </MantineProvider>
    </StrictMode>
  )
}

export function mountDevApp(rootElement: HTMLElement) {
  createRoot(rootElement).render(<DevAppRoot />)
}
