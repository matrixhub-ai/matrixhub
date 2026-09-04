import { Anchor } from '@mantine/core'
import { createLink } from '@tanstack/react-router'
import * as React from 'react'

import type { AnchorProps } from '@mantine/core'
import type { LinkComponent } from '@tanstack/react-router'

type TableCellAnchorProps = Omit<AnchorProps, 'href'>

const TableCellAnchor = ({
  ref, ...props
}: TableCellAnchorProps & { ref?: React.RefObject<HTMLAnchorElement | null> }) => {
  return (
    <Anchor
      ref={ref}
      size="sm"
      fw={600}
      underline="never"
      {...props}
    />
  )
}

const TableCellLinkComponent = createLink(TableCellAnchor)

/**
 * Standard in-table navigation link (e.g. a resource name cell that links to
 * its detail page). Matches the 14px body cell text and drops the underline;
 * pass any `Anchor` prop to override the defaults.
 */
export const TableCellLink: LinkComponent<typeof TableCellAnchor> = (
  props,
) => {
  return <TableCellLinkComponent {...props} />
}
