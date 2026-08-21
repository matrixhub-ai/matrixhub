import assert from 'node:assert/strict'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'

import { createServer } from 'vite'

const root = fileURLToPath(new URL('../', import.meta.url))

function addedFileDiff(filename, content) {
  return [
    `diff --git a/${filename} b/${filename}`,
    'new file mode 100644',
    '--- /dev/null',
    `+++ b/${filename}`,
    '@@ -0,0 +1 @@',
    `+${content}`,
    '',
  ].join('\n')
}

function sizedFileDiff(filename, byteLength, fill = 'x') {
  const emptyDiff = addedFileDiff(filename, '')
  const contentLength = byteLength - Buffer.byteLength(emptyDiff)

  assert.ok(contentLength >= 0)

  const fillBytes = Buffer.byteLength(fill)
  const content = fill.repeat(Math.floor(contentLength / fillBytes))
    + 'x'.repeat(contentLength % fillBytes)
  const rawDiff = addedFileDiff(filename, content)

  assert.equal(Buffer.byteLength(rawDiff), byteLength)

  return rawDiff
}

function addedLinesDiff(filename, lines) {
  return [
    `diff --git a/${filename} b/${filename}`,
    'new file mode 100644',
    '--- /dev/null',
    `+++ b/${filename}`,
    `@@ -0,0 +1,${lines.length} @@`,
    ...lines.map(line => `+${line}`),
    '',
  ].join('\n')
}

test('commit diffs follow the Hugging Face file size and file count limits', async () => {
  const server = await createServer({
    appType: 'custom',
    configFile: false,
    optimizeDeps: { noDiscovery: true },
    root,
    server: { middlewareMode: true },
  })

  try {
    const {
      MAX_FILES,
      renderCommitDiff,
    } = await server.ssrLoadModule('/src/shared/components/commit-detail/commitDiff.ts')
    const message = 'The diff for this file is too large to render.'
    const linkText = 'See raw diff'

    const atLimitResult = renderCommitDiff(
      sizedFileDiff('at-limit.txt', 100_000, '界'),
      message,
      linkText,
    )

    assert.equal(atLimitResult.hasTooManyFiles, false)
    assert.equal(atLimitResult.html.includes(message), false)
    assert.match(atLimitResult.html, /at-limit\.txt/)

    const overLimitResult = renderCommitDiff(
      sizedFileDiff('over-limit.txt', 100_001, '界'),
      message,
      linkText,
    )

    assert.equal(overLimitResult.hasTooManyFiles, false)
    assert.equal(overLimitResult.html.includes(message), true)
    assert.match(overLimitResult.html, /data-commit-raw-diff-link/)
    assert.match(overLimitResult.html, />See raw diff<\/a>/)
    assert.ok(overLimitResult.html.length < 10_000)

    const changedLines = Array.from({ length: 9_000 }, (_, index) => String(index))
    const changedLinesDiff = addedLinesDiff('many-lines.txt', changedLines)

    assert.ok(Buffer.byteLength(changedLinesDiff) < 100_000)

    const changedLinesResult = renderCommitDiff(
      changedLinesDiff,
      message,
      linkText,
    )

    assert.equal(changedLinesResult.html.includes(message), false)
    assert.match(changedLinesResult.html, />8999</)

    const oversizedFilesDiff = Array.from(
      { length: MAX_FILES + 1 },
      (_, index) => sizedFileDiff(`file-${index}.txt`, 100_001),
    ).join('')

    assert.ok(Buffer.byteLength(oversizedFilesDiff) > 5_000_000)

    const fileLimitResult = renderCommitDiff(
      oversizedFilesDiff,
      message,
      linkText,
    )

    assert.equal(fileLimitResult.hasTooManyFiles, true)
    assert.match(fileLimitResult.html, new RegExp(`file-${MAX_FILES - 1}\\.txt`))
    assert.doesNotMatch(fileLimitResult.html, new RegExp(`file-${MAX_FILES}\\.txt`))
    assert.equal(fileLimitResult.html.split(message).length - 1, MAX_FILES)
  } finally {
    await server.close()
  }
})
