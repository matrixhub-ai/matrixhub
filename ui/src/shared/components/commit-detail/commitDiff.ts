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

    return parse(isTooBig ? `${fileDiff}\n ` : fileDiff, options)
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
