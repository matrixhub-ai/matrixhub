import '@tanstack/react-router'

declare module '@tanstack/react-router' {
  interface StaticDataRouteOption {
    navName?: string
  }
}
