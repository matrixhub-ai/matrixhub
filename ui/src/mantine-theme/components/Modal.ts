import { Modal, rem } from '@mantine/core'

const modalSizeMap = {
  xs: rem(300),
  sm: rem(390),
  md: rem(480),
} as const

const isMappedModalSize = (size: unknown): size is keyof typeof modalSizeMap => (
  typeof size === 'string' && Object.prototype.hasOwnProperty.call(modalSizeMap, size)
)

export const modalTheme = Modal.extend({
  defaultProps: {
    size: 'md',
  },
  vars: (_, props) => ({
    root: {
      '--modal-size': isMappedModalSize(props.size) ? modalSizeMap[props.size] : undefined,
    },
    content: {
      '--paper-radius': rem(12),
    },
  }),
})
