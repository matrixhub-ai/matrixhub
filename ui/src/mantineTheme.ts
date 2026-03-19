import { createTheme, rem } from '@mantine/core'

import type { CSSVariablesResolver } from '@mantine/core'

export const mantineTheme = createTheme({
  primaryColor: 'cyan',
})

export const cssVariablesResolver: CSSVariablesResolver = () => ({
  variables: {
    '--app-size-icon-basic': rem(20),
    '--app-size-icon-md': rem(16),
    '--app-size-icon-sm': rem(16),
    '--app-size-radius-mmd': rem(6),
  },
  light: {
    '--mantine-color-text': 'var(--mantine-color-gray-9)',
    '--app-color-text-default': 'var(--mantine-color-gray-9)',
    '--app-color-text-label': 'var(--mantine-color-gray-7)',
    '--app-color-text-link': 'var(--mantine-color-blue-6)',
    '--app-color-text-title': 'var(--mantine-color-gray-9)',
    '--app-color-text-error-default': 'var(--mantine-color-red-6)',
    '--app-color-text-error-disabled': 'var(--mantine-color-red-4)',
    '--app-color-background-body': 'var(--mantine-color-white)',
    '--app-color-background-success-filled-hover': 'var(--mantine-color-teal-7)',
    '--app-color-border-default': '1px solid var(--mantine-color-gray-3)',
    '--app-color-border-error-default': '1px solid var(--mantine-color-red-6)',
    '--app-color-gray-10': 'var(--mantine-color-gray-0)',
    '--app-color-gray-20': 'var(--mantine-color-gray-1)',
    '--app-color-gray-30': 'var(--mantine-color-gray-2)',
    '--app-color-gray-40': 'var(--mantine-color-gray-3)',
    '--app-color-gray-50': 'var(--mantine-color-gray-4)',
    '--app-color-gray-60': 'var(--mantine-color-gray-5)',
    '--app-color-gray-70': 'var(--mantine-color-gray-6)',
    '--app-color-gray-80': 'var(--mantine-color-gray-7)',
    '--app-color-gray-90': 'var(--mantine-color-gray-8)',
    '--app-color-gray-100': 'var(--mantine-color-gray-9)',
  },
  dark: {
    '--mantine-color-text': 'var(--mantine-color-white)',
    '--app-color-text-default': 'var(--mantine-color-white)',
    '--app-color-text-label': 'var(--mantine-color-gray-3)',
    '--app-color-text-link': 'var(--mantine-color-blue-4)',
    '--app-color-text-title': 'var(--mantine-color-dark-0)',
    '--app-color-text-error-default': 'var(--mantine-color-red-7)',
    '--app-color-text-error-disabled': 'rgba(224, 49, 49, 0.5)',
    '--app-color-background-body': 'var(--mantine-color-dark-8)',
    '--app-color-background-success-filled-hover': 'var(--mantine-color-teal-8)',
    '--app-color-border-default': '1px solid var(--mantine-color-dark-4)',
    '--app-color-border-error-default': '1px solid var(--mantine-color-red-7)',
    '--app-color-gray-10': 'var(--mantine-color-dark-6)',
    '--app-color-gray-20': 'var(--mantine-color-dark-5)',
    '--app-color-gray-30': 'var(--mantine-color-dark-4)',
    '--app-color-gray-40': 'var(--mantine-color-dark-3)',
    '--app-color-gray-50': 'var(--mantine-color-dark-1)',
    '--app-color-gray-60': 'var(--mantine-color-dark-0)',
    '--app-color-gray-70': 'var(--mantine-color-gray-3)',
    '--app-color-gray-80': 'var(--mantine-color-gray-2)',
    '--app-color-gray-90': 'var(--mantine-color-gray-1)',
    '--app-color-gray-100': 'var(--mantine-color-gray-0)',
  },
  components: {
    Pagination: {
      defaultProps: {
        boundaries: 1,
        siblings: 2,
        color: 'cyan',
        size: 20,
        radius: 4,
        gap: 8,
      },
      styles: {
        control: {
          minWidth: 20,
          height: 20,
          fontSize: '12px',
          fontWeight: 400,
          lineHeight: '16px',
          borderColor: 'var(--mantine-color-gray-3)',
          color: 'var(--mantine-color-gray-8)',
        },
        dots: {
          minWidth: 20,
          height: 20,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--mantine-color-gray-8)',
        },
      },
    },
  },
})
