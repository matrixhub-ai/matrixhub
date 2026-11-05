import type { GuideSnippet } from '@/features/models/guides/GuideDrawer'

export type ModelCommandType = 'upload' | 'download'

export const HF_TOKEN_PLACEHOLDER = '<your-matrixhub-token>'

/** Default cache dir used by huggingface_hub when no --local-dir is given. */
export const HF_DEFAULT_CACHE_DIR = '~/.cache/huggingface/hub'

export function buildInstallHfSnippet(): GuideSnippet {
  return {
    lang: 'bash',
    code: 'pip install -U huggingface_hub',
  }
}

/** Download only needs the endpoint; upload also needs a token to write. */
export function buildHfEnvSnippet(type: ModelCommandType, hfEndpoint: string): GuideSnippet {
  const lines = [`export HF_ENDPOINT="${hfEndpoint}"`]

  if (type === 'upload') {
    lines.push(`export HF_TOKEN="${HF_TOKEN_PLACEHOLDER}"`)
  }

  return {
    lang: 'bash',
    code: lines.join('\n'),
  }
}

export function buildDownloadSnippet(modelPath: string, revision?: string): GuideSnippet {
  return {
    lang: 'bash',
    code: `hf download "${modelPath}"${revision ? ` --revision "${revision}"` : ''}`,
  }
}

/** Derive a placeholder local folder from the model name so the example reads naturally. */
export function buildLocalPathPlaceholder(modelPath: string): string {
  return `/path/to/${modelPath.split('/').pop() || 'model'}`
}

export function buildUploadSnippet(modelPath: string): GuideSnippet {
  return {
    lang: 'bash',
    code: `hf upload \\\n    "${modelPath}" \\\n    "${buildLocalPathPlaceholder(modelPath)}" \\\n    "."`,
  }
}

export function buildModelScopeInstallSnippet(): GuideSnippet {
  return {
    lang: 'bash',
    code: 'pip install -U modelscope modelscope-hub',
  }
}

export function buildModelScopeEnvSnippet(endpoint: string): GuideSnippet {
  return {
    lang: 'bash',
    code: `export MODELSCOPE_ENDPOINT="${endpoint}"`,
  }
}

export function buildModelScopeDownloadSnippet(modelPath: string, revision?: string): GuideSnippet {
  return {
    lang: 'bash',
    code: `ms-hub download "${modelPath}"${revision ? ` --revision "${revision}"` : ''} --local-dir "./${modelPath.split('/').pop()}"`,
  }
}

export function buildModelScopePythonSnippet(modelPath: string, endpoint: string, revision?: string): GuideSnippet {
  const lines = [
    'from modelscope import snapshot_download',
    '',
    'model_dir = snapshot_download(',
    `    ${JSON.stringify(modelPath)},`,
    `    endpoint=${JSON.stringify(endpoint)},`,
  ]

  if (revision) {
    lines.push(`    revision=${JSON.stringify(revision)},`)
  }
  lines.push(')')

  return {
    lang: 'python',
    code: lines.join('\n'),
  }
}
