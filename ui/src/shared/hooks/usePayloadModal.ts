import { useState } from 'react'

export function usePayloadModal<T>() {
  const [payload, setPayload] = useState<T | null>(null)

  const open = (nextPayload: T) => {
    setPayload(nextPayload)
  }

  const close = () => {
    setPayload(null)
  }

  return {
    opened: payload !== null,
    payload,
    open,
    close,
  }
}
