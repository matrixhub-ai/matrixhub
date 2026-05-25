import { Pagination } from '@mantine/core'

export const paginationTheme = Pagination.extend({
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
})
