import { mountApp } from './app'

const rootElement = document.getElementById('root')

if (!rootElement) {
  throw new Error('Root element not found')
}

mountApp(rootElement)
