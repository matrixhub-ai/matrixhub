import { filesize } from 'filesize'

export function formatStorageSize(value: string | undefined) {
  if (!value) {
    return '-'
  }

  const numericValue = Number(value)

  if (!Number.isFinite(numericValue)) {
    return value
  }

  return filesize(numericValue, {
    standard: 'jedec',
    round: 1,
  }) as string
}
