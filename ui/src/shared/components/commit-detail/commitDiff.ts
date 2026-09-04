import { html as formatDiffHtml, parse } from 'diff2html'

export const MAX_FILES = 50
const MAX_FILE_DIFF_BYTES = 100_000

interface CommitDiffRender {
  html: string
  hasTooManyFiles: boolean
}

export function renderCommitDiff(
  rawDiff: string,
  diffTooBigMessage: string,
  rawDiffLinkText: string,
): CommitDiffRender {
  const message = `${diffTooBigMessage} <a href="#" class="commit-detail-raw-diff-link" data-commit-raw-diff-link>${rawDiffLinkText}</a>`
  const fileDiffs = rawDiff.split(/(?=^diff --git )/m).filter(Boolean)
  const diffJson = fileDiffs.slice(0, MAX_FILES).flatMap((fileDiff) => {
    const isTooBig = new TextEncoder().encode(fileDiff).byteLength > MAX_FILE_DIFF_BYTES
    const options = isTooBig
      ? {
          diffMaxChanges: 0,
          diffTooBigMessage: () => message,
        }
      : undefined

    const parsedDiff = parse(isTooBig ? `${fileDiff}\n ` : fileDiff, options)

    if (isTooBig && parsedDiff[0]) {
      let addedLines = 0
      let deletedLines = 0
      let inHunk = false

      for (let lineStart = 0; lineStart < fileDiff.length;) {
        const lineEnd = fileDiff.indexOf('\n', lineStart)

        if (fileDiff.startsWith('@@', lineStart)) {
          inHunk = true
        } else if (inHunk && fileDiff[lineStart] === '+') {
          addedLines++
        } else if (inHunk && fileDiff[lineStart] === '-') {
          deletedLines++
        }

        lineStart = lineEnd === -1 ? fileDiff.length : lineEnd + 1
      }

      parsedDiff[0].addedLines = addedLines
      parsedDiff[0].deletedLines = deletedLines
    }

    return parsedDiff
  })

  return {
    html: diffJson.length > 0
      ? formatDiffHtml(diffJson, {
          drawFileList: true,
          matching: 'lines',
          outputFormat: 'side-by-side',
        })
      : '',
    hasTooManyFiles: fileDiffs.length > MAX_FILES,
  }
}
