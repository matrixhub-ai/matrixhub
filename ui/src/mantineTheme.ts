import {
  Alert,
  Anchor,
  createTheme,
  Modal,
  Input,
  PasswordInput,
  rem,
  Tabs,
  Pagination,
} from '@mantine/core'
import { type CSSVariablesResolver } from '@mantine/core'

const modalSizeMap = {
  xs: rem(300),
  sm: rem(390),
  md: rem(480),
} as const

const isMappedModalSize = (size: unknown): size is keyof typeof modalSizeMap => (
  typeof size === 'string' && Object.prototype.hasOwnProperty.call(modalSizeMap, size)
)

export const mantineTheme = createTheme({
  primaryColor: 'cyan',
  components: {
    Modal: Modal.extend({
      vars: (_, props) => ({
        root: {
          '--modal-size': isMappedModalSize(props.size) ? modalSizeMap[props.size] : undefined,
        },
        content: {
          '--paper-radius': rem(12),
        },
      }),
    }),
    ModalHeader: Modal.Header.extend({
      defaultProps: {
        px: 'lg',
        py: 'md',
      },
    }),
    ModalBody: Modal.Body.extend({
      defaultProps: {
        p: 'lg',
        pt: 0,
      },
    }),
    Tabs: Tabs.extend({
      vars: () => ({
        root: {},
        list: {
          '--tabs-list-gap': rem(20),
        },
      }),
    }),
    TabsTab: Tabs.Tab.extend({
      defaultProps: {
        lh: rem(20),
        fw: 600,
        px: 12,
        pt: 8,
        pb: 6,
      },
    }),
    Alert: Alert.extend({
      defaultProps: {
        px: 'sm',
        py: 'sm',
        bd: 'none',
      },
      styles: {
        icon: {
          marginInlineEnd: rem(8),
        },
      },
    }),
    InputWrapper: Input.Wrapper.extend({
      defaultProps: {
        c: 'gray.7',
        labelProps: {
          style: {
            lineHeight: '20px',
            marginBottom: '6px',
          },
        },
      },
    }),
    PasswordInput: PasswordInput.extend({
      defaultProps: {
        labelProps: {
          style: {
            lineHeight: '20px',
            marginBottom: '6px',
          },
        },
      },
    }),
    Anchor: Anchor.extend({
      defaultProps: {
        c: 'blue.6',
      },
    }),
    Pagination: Pagination.extend({
      defaultProps: {
        boundaries: 1,
        siblings: 2,
        color: 'cyan',
        size: 20,
        radius: 4,
        gap: 8,
        withControls: false,
        withEdges: false,
      },
      styles: {
        control: {
          minWidth: 20,
          height: 20,
          fontSize: '12px',
          fontWeight: 400,
          lineHeight: '16px',
          borderColor: 'var(--mantine-color-gray-3)',
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
    }),
  },
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
})
