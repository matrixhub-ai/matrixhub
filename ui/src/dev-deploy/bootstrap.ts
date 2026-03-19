const STORAGE_KEY = 'dev-api-proxy-target'

interface ToolbarController {
  setTarget: (target: string) => void
}

interface BootstrapOptions {
  serviceWorkerPath: string
}

function normalizeApiTarget(target: string) {
  try {
    const url = new URL(target.trim())

    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return null
    }

    const trimmedPath = url.pathname.replace(/\/+$/, '')

    url.pathname = trimmedPath || '/'
    url.search = ''
    url.hash = ''

    return url.toString()
  } catch {
    return null
  }
}

function applyStyles(element: HTMLElement, styles: Partial<CSSStyleDeclaration>) {
  Object.assign(element.style, styles)
}

function createModalShell() {
  const overlay = document.createElement('div')

  applyStyles(overlay, {
    position: 'fixed',
    inset: '0',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'rgba(0,0,0,0.5)',
    zIndex: '10000',
  })

  const box = document.createElement('div')

  applyStyles(box, {
    background: '#fff',
    borderRadius: '12px',
    padding: '32px',
    minWidth: '400px',
    maxWidth: '480px',
    boxShadow: '0 8px 32px rgba(0,0,0,0.2)',
  })

  overlay.append(box)
  document.body.append(overlay)

  return {
    box,
    destroy() {
      overlay.remove()
    },
  }
}

function createHeading(text: string) {
  const heading = document.createElement('h2')

  applyStyles(heading, {
    margin: '0 0 8px',
    fontSize: '20px',
  })
  heading.textContent = text

  return heading
}

function createDescription() {
  const paragraph = document.createElement('p')

  applyStyles(paragraph, {
    margin: '0 0 20px',
    color: '#666',
    fontSize: '14px',
  })

  paragraph.append(
    document.createTextNode('Enter the backend base URL to proxy '),
    createCode('/api'),
    document.createTextNode(' requests. Both '),
    createCode('https://host'),
    document.createTextNode(' and '),
    createCode('https://host/path'),
    document.createTextNode(' are supported.'),
  )

  return paragraph
}

function createCode(text: string) {
  const code = document.createElement('code')

  code.textContent = text

  return code
}

function createButton(label: string, enabled = true) {
  const button = document.createElement('button')

  button.type = 'button'
  button.disabled = !enabled
  button.textContent = label

  applyStyles(button, {
    marginTop: '12px',
    width: '100%',
    padding: '10px 0',
    fontSize: '14px',
    fontWeight: '600',
    color: '#fff',
    background: enabled ? '#228be6' : '#aaa',
    border: 'none',
    borderRadius: '8px',
    cursor: enabled ? 'pointer' : 'not-allowed',
  })

  return button
}

function setButtonEnabled(button: HTMLButtonElement, enabled: boolean) {
  button.disabled = !enabled
  button.style.background = enabled ? '#228be6' : '#aaa'
  button.style.cursor = enabled ? 'pointer' : 'not-allowed'
}

function showStatus(message: string) {
  const shell = createModalShell()
  const paragraph = document.createElement('p')

  applyStyles(paragraph, {
    margin: '0',
    color: '#666',
    fontSize: '14px',
    textAlign: 'center',
  })
  paragraph.textContent = message
  shell.box.append(paragraph)

  return shell.destroy
}

function promptForTarget(defaultValue: string) {
  return new Promise<string>((resolve) => {
    const shell = createModalShell()
    const input = document.createElement('input')
    const submitButton = createButton('Connect', false)

    applyStyles(input, {
      width: '100%',
      padding: '10px 12px',
      fontSize: '14px',
      border: '1px solid #ddd',
      borderRadius: '8px',
      outline: 'none',
      boxSizing: 'border-box',
    })
    input.type = 'url'
    input.placeholder = 'https://192.168.1.100:4443/backend'
    input.value = defaultValue

    const submit = () => {
      const normalizedTarget = normalizeApiTarget(input.value)

      if (!normalizedTarget) {
        return
      }

      shell.destroy()
      resolve(normalizedTarget)
    }

    const updateButtonState = () => {
      setButtonEnabled(submitButton, !!normalizeApiTarget(input.value))
    }

    input.addEventListener('input', updateButtonState)
    input.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' && normalizeApiTarget(input.value)) {
        submit()
      }
    })
    submitButton.addEventListener('click', submit)

    shell.box.append(
      createHeading('Dev Deploy Configuration'),
      createDescription(),
      input,
      submitButton,
    )

    updateButtonState()
    queueMicrotask(() => input.focus())
  })
}

function promptForCertificateTrust(target: string) {
  return new Promise<void>((resolve) => {
    const shell = createModalShell()
    const description = document.createElement('p')
    const trustButton = createButton('Open Backend to Trust Certificate')
    const retryButton = createButton('Done, Retry Connection')

    applyStyles(description, {
      margin: '0 0 8px',
      color: '#666',
      fontSize: '14px',
    })
    description.append(
      document.createTextNode('Cannot reach '),
      createCode(target),
      document.createTextNode('. If using a self-signed certificate, open the link below and accept it in your browser.'),
    )

    trustButton.style.background = '#f59f00'
    trustButton.addEventListener('click', () => {
      window.open(target, '_blank', 'noopener,noreferrer')
    })
    retryButton.addEventListener('click', () => {
      shell.destroy()
      resolve()
    })

    shell.box.append(
      createHeading('Certificate Not Trusted'),
      description,
      trustButton,
      retryButton,
    )
  })
}

function createToolbar(target: string, onEdit: () => void): ToolbarController {
  const toolbar = document.createElement('div')
  const indicator = document.createElement('span')
  const label = document.createElement('span')
  const editButton = document.createElement('button')

  applyStyles(toolbar, {
    position: 'fixed',
    bottom: '0',
    left: '0',
    right: '0',
    background: '#1a1b1e',
    color: '#c1c2c5',
    padding: '6px 16px',
    fontSize: '12px',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    zIndex: '10000',
    fontFamily: 'monospace',
  })

  applyStyles(indicator, {
    color: '#51cf66',
    fontWeight: '700',
  })
  indicator.textContent = 'DEV'

  const setTarget = (nextTarget: string) => {
    label.textContent = `API -> ${nextTarget}`
  }

  setTarget(target)

  editButton.type = 'button'
  editButton.textContent = 'Edit'
  editButton.addEventListener('click', onEdit)
  applyStyles(editButton, {
    marginLeft: 'auto',
    background: 'transparent',
    border: '1px solid #555',
    color: '#c1c2c5',
    borderRadius: '4px',
    padding: '2px 10px',
    cursor: 'pointer',
    fontSize: '12px',
  })

  toolbar.append(indicator, label, editButton)
  document.body.append(toolbar)

  return {
    setTarget,
  }
}

async function waitForActivation(serviceWorker: ServiceWorker) {
  if (serviceWorker.state === 'activated') {
    return serviceWorker
  }

  await new Promise<void>((resolve) => {
    serviceWorker.addEventListener('statechange', () => {
      if (serviceWorker.state === 'activated') {
        resolve()
      }
    })
  })

  return serviceWorker
}

async function registerProxyServiceWorker(serviceWorkerPath: string) {
  if (!('serviceWorker' in navigator)) {
    throw new Error('Service Worker is not supported in this browser.')
  }

  const registration = await navigator.serviceWorker.register(serviceWorkerPath)
  const serviceWorker = registration.active ?? registration.installing ?? registration.waiting

  if (serviceWorker) {
    return waitForActivation(serviceWorker)
  }

  const readyRegistration = await navigator.serviceWorker.ready

  if (!readyRegistration.active) {
    throw new Error('Unable to activate API proxy Service Worker.')
  }

  return readyRegistration.active
}

function sendApiTarget(serviceWorker: ServiceWorker | null, target: string) {
  const activeServiceWorker = serviceWorker ?? navigator.serviceWorker.controller

  activeServiceWorker?.postMessage({
    type: 'SET_API_TARGET',
    target,
  })
}

async function ensureTargetReachable(target: string) {
  while (true) {
    try {
      await fetch(target, { mode: 'no-cors' })

      return
    } catch {
      if (!target.startsWith('https://')) {
        return
      }

      await promptForCertificateTrust(target)
    }
  }
}

async function applyApiTarget(serviceWorker: ServiceWorker, target: string) {
  const normalizedTarget = normalizeApiTarget(target)

  if (!normalizedTarget) {
    throw new Error('Invalid API target.')
  }

  localStorage.setItem(STORAGE_KEY, normalizedTarget)
  sendApiTarget(serviceWorker, normalizedTarget)
  await ensureTargetReachable(normalizedTarget)

  return normalizedTarget
}

function showFatalError(message: string) {
  const shell = createModalShell()
  const description = document.createElement('p')

  applyStyles(description, {
    margin: '0',
    color: '#c92a2a',
    fontSize: '14px',
    lineHeight: '1.5',
  })
  description.textContent = message

  shell.box.append(createHeading('Dev Deploy Initialization Failed'), description)
}

export async function bootstrapDevDeploy({
  serviceWorkerPath,
}: BootstrapOptions) {
  try {
    const disposeLoading = showStatus('Initializing service worker...')
    const serviceWorker = await registerProxyServiceWorker(serviceWorkerPath)

    disposeLoading()

    let apiTarget = normalizeApiTarget(localStorage.getItem(STORAGE_KEY) ?? '')

    while (!apiTarget) {
      apiTarget = await promptForTarget(localStorage.getItem(STORAGE_KEY) ?? '')
    }

    if (!apiTarget) {
      throw new Error('API target is required.')
    }

    const disposeChecking = showStatus('Checking backend connectivity...')

    apiTarget = await applyApiTarget(serviceWorker, apiTarget)
    disposeChecking()

    const toolbar = createToolbar(apiTarget, async () => {
      const nextTarget = await promptForTarget(apiTarget ?? '')
      const disposeEditChecking = showStatus('Checking backend connectivity...')

      apiTarget = await applyApiTarget(serviceWorker, nextTarget)
      disposeEditChecking()
      toolbar.setTarget(apiTarget)
    })

    await import('../main.tsx')
  } catch (error) {
    const detail = error instanceof Error ? error.message : 'Unknown error.'

    showFatalError(detail)
    throw error
  }
}
