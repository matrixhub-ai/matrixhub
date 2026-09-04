import {
  Text,
  Tooltip,
} from '@mantine/core'
import {
  useRef,
  useState,
} from 'react'

import type {
  TextProps,
  TooltipProps,
} from '@mantine/core'
import type { MouseEvent, ReactNode } from 'react'

export interface TruncatedTextProps extends Omit<TextProps, 'truncate' | 'children'> {
  /** Full value. Used both as rendered content and as the tooltip label. */
  value: ReactNode
  /** Tooltip label when it should differ from `value` (e.g. `value` is a link). */
  tooltipLabel?: ReactNode
  /** Rendered content when it should differ from `value` (e.g. an `Anchor`). */
  children?: ReactNode
  /** Extra props forwarded to the tooltip. */
  tooltipProps?: Omit<TooltipProps, 'children' | 'label'>
  /** Composed with the internal overflow check rather than replacing it. */
  onMouseEnter?: (event: MouseEvent<HTMLDivElement>) => void
}

/**
 * Single-line cell text that truncates on overflow and reveals the full value
 * on hover. The tooltip only activates when the text actually overflows, so
 * short values do not get a redundant hover card.
 */
export function TruncatedText({
  value,
  tooltipLabel,
  children,
  tooltipProps,
  onMouseEnter,
  ...textProps
}: TruncatedTextProps) {
  const contentRef = useRef<HTMLDivElement>(null)
  const [overflowing, setOverflowing] = useState(false)

  // Composed rather than passed straight to `Text`, so a caller-provided
  // `onMouseEnter` cannot replace the overflow check and silently disable the
  // tooltip.
  const handleMouseEnter = (event: MouseEvent<HTMLDivElement>) => {
    const element = contentRef.current

    if (element) {
      setOverflowing(element.scrollWidth > element.clientWidth)
    }

    onMouseEnter?.(event)
  }

  const content = (
    <Text
      ref={contentRef}
      size="sm"
      truncate="end"
      {...textProps}
      onMouseEnter={handleMouseEnter}
    >
      {children ?? value}
    </Text>
  )

  return (
    <Tooltip
      label={tooltipLabel ?? value}
      disabled={!overflowing}
      multiline
      maw={480}
      withArrow
      openDelay={200}
      {...tooltipProps}
    >
      {content}
    </Tooltip>
  )
}
