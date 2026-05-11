import { Input } from '@mantine/core'

export const inputWrapperTheme = Input.Wrapper.extend({
  defaultProps: {
    c: 'gray.7',
    labelProps: {
      style: {
        lineHeight: '20px',
        marginBottom: '6px',
      },
    },
  },
})
