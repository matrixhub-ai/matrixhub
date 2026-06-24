import { useClipboard } from '@/shared/hooks/useClipboard'

import type { ReactNode } from 'react'

export interface CopyButtonProps {
  /** Children callback that receives the current status and copy function. */
  children: (payload: {
    copied: boolean
    copy: () => void
  }) => ReactNode
  /** Value copied to the clipboard when copy is called. */
  value: string
  /** Copied status timeout in ms. */
  timeout?: number
}

export function CopyButton({
  children,
  timeout = 1000,
  value,
}: CopyButtonProps) {
  const clipboard = useClipboard({
    timeout,
  })
  const copy = () => clipboard.copy(value)

  return (
    <>
      {children({
        copy,
        copied: clipboard.copied,
      })}
    </>
  )
}

CopyButton.displayName = 'CopyButton'
