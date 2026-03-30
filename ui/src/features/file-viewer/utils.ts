import type { FileCategory } from './types'

const MARKDOWN_EXTENSIONS = new Set(['.md', '.mdx'])

const CODE_EXTENSIONS = new Set([
  '.json',
  '.yaml',
  '.yml',
  '.py',
  '.txt',
  '.toml',
  '.cfg',
  '.ini',
  '.sh',
  '.bash',
  '.js',
  '.ts',
  '.jsx',
  '.tsx',
  '.css',
  '.scss',
  '.less',
  '.html',
  '.xml',
  '.dockerfile',
  '.gitignore',
  '.gitattributes',
  '.env',
  '.conf',
  '.log',
  '.rst',
  '.tex',
  '.r',
  '.go',
  '.rs',
  '.java',
  '.c',
  '.cpp',
  '.h',
  '.rb',
  '.php',
  '.swift',
  '.kt',
  '.lua',
  '.sql',
  '.graphql',
  '.proto',
])

const EXTENSION_TO_LANGUAGE: Record<string, string> = {
  '.json': 'json',
  '.yaml': 'yaml',
  '.yml': 'yaml',
  '.py': 'python',
  '.txt': 'plaintext',
  '.toml': 'ini',
  '.cfg': 'ini',
  '.ini': 'ini',
  '.sh': 'shell',
  '.bash': 'shell',
  '.js': 'javascript',
  '.ts': 'typescript',
  '.jsx': 'javascript',
  '.tsx': 'typescript',
  '.css': 'css',
  '.scss': 'scss',
  '.less': 'less',
  '.html': 'html',
  '.xml': 'xml',
  '.dockerfile': 'dockerfile',
  '.env': 'plaintext',
  '.conf': 'plaintext',
  '.log': 'plaintext',
  '.rst': 'plaintext',
  '.tex': 'latex',
  '.r': 'r',
  '.go': 'go',
  '.rs': 'rust',
  '.java': 'java',
  '.c': 'c',
  '.cpp': 'cpp',
  '.h': 'c',
  '.rb': 'ruby',
  '.php': 'php',
  '.swift': 'swift',
  '.kt': 'kotlin',
  '.lua': 'lua',
  '.sql': 'sql',
  '.graphql': 'graphql',
  '.proto': 'protobuf',
  '.md': 'markdown',
  '.mdx': 'markdown',
}

const CODE_BASENAMES = new Set([
  'dockerfile',
  'makefile',
  'cmakelists.txt',
  '.gitignore',
  '.gitattributes',
])

export function getFileExtension(name: string | undefined): string {
  if (!name) {
    return ''
  }
  const dotIndex = name.lastIndexOf('.')

  if (dotIndex === -1) {
    return ''
  }

  return name.slice(dotIndex).toLowerCase()
}

export function classifyFile(name: string | undefined): FileCategory {
  const ext = getFileExtension(name)

  if (MARKDOWN_EXTENSIONS.has(ext)) {
    return 'markdown'
  }
  if (CODE_EXTENSIONS.has(ext)) {
    return 'code'
  }

  // Also treat extensionless known files as code
  const basename = name?.split('/').pop()?.toLowerCase() ?? ''

  if (CODE_BASENAMES.has(basename)) {
    return 'code'
  }

  return 'binary'
}

export function getMonacoLanguage(name: string | undefined): string {
  const ext = getFileExtension(name)

  return EXTENSION_TO_LANGUAGE[ext] ?? 'plaintext'
}
