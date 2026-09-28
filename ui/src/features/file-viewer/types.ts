export type FileCategory = 'markdown' | 'code' | 'binary'

/**
 * Minimal file shape that FileViewer components depend on.
 * Structurally compatible with the API `File` type, but decoupled
 * so API changes don't ripple through the component layer.
 */
export interface FileViewerFile {
  name?: string
  size?: string
  url?: string
  lfs?: boolean
  sha256?: string
  xetHash?: string
  pointerSize?: string
  commit?: {
    id?: string
    authorName?: string
    message?: string
    committerDate?: string
  }
}
