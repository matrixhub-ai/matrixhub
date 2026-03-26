import { Anchor } from '@mantine/core'
import { createLink } from '@tanstack/react-router'
import * as React from 'react'

import type { AnchorProps } from '@mantine/core'
import type { LinkComponent } from '@tanstack/react-router'

type MartineAnchorProps = Omit<AnchorProps, 'href'>

const MantineLinkComponent = ({
  ref, ...props
}: MartineAnchorProps & { ref?: React.RefObject<HTMLAnchorElement | null> }) => {
  return <Anchor ref={ref} {...props} />
}

const AnchorLinkComponent = createLink(MantineLinkComponent)

export const AnchorLink: LinkComponent<typeof MantineLinkComponent> = (
  props,
) => {
  return <AnchorLinkComponent {...props} />
}

export default AnchorLink
