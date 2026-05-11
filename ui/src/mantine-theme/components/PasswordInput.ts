import { PasswordInput } from '@mantine/core'

export const passwordInputTheme = PasswordInput.extend({
  defaultProps: {
    labelProps: {
      style: {
        lineHeight: '20px',
        marginBottom: '6px',
      },
    },
  },
})
