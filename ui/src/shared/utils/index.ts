export function filterByKeyword<T extends { name?: string }>(items: T[], keyword: string) {
  const normalized = keyword.trim().toLowerCase()

  if (!normalized) {
    return items
  }

  return items.filter(item => item.name?.toLowerCase()?.includes(normalized))
}

export function parseConventionalCommit(commitMessage: string) {
  const msg = commitMessage.trim()
  const lines = msg.split('\n').map(line => line.trimEnd())

  const result: Record<string, string | null> = {
    header: '',
    body: null,
    footer: null,
  }

  result.header = lines[0] || ''

  const restLines = lines.slice(1).filter(line => line !== '')

  if (restLines.length === 0) {
    return result
  }

  const footerStartIndex = restLines.findIndex(line =>
    /^\s*[\w-]+:\s/.test(line),
  )

  if (footerStartIndex === -1) {
    result.body = restLines.join('\n').trim() || null
  } else {
    const bodyLines = restLines.slice(0, footerStartIndex)
    const footerLines = restLines.slice(footerStartIndex)

    result.body = bodyLines.join('\n').trim() || null
    result.footer = footerLines.join('\n').trim() || null
  }

  return result
}
