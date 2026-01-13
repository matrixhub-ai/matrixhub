import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import type { BlobContent } from '../api/client';
import './FileViewer.css';

interface FileViewerProps {
  content: BlobContent | null;
  filename: string;
}

// Get language from filename extension
function getLanguage(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() || '';
  const languageMap: Record<string, string> = {
    'js': 'javascript',
    'jsx': 'jsx',
    'ts': 'typescript',
    'tsx': 'tsx',
    'py': 'python',
    'rb': 'ruby',
    'go': 'go',
    'rs': 'rust',
    'java': 'java',
    'c': 'c',
    'cpp': 'cpp',
    'cc': 'cpp',
    'h': 'c',
    'hpp': 'cpp',
    'cs': 'csharp',
    'php': 'php',
    'html': 'html',
    'css': 'css',
    'scss': 'scss',
    'sass': 'sass',
    'json': 'json',
    'xml': 'xml',
    'yaml': 'yaml',
    'yml': 'yaml',
    'md': 'markdown',
    'markdown': 'markdown',
    'sh': 'bash',
    'bash': 'bash',
    'sql': 'sql',
    'dockerfile': 'dockerfile',
    'makefile': 'makefile',
  };
  return languageMap[ext] || 'text';
}

export function FileViewer({ content, filename }: FileViewerProps) {
  const language = getLanguage(filename);

  return (
    <div className="file-viewer">
      <div className="file-header">
        <span className="file-name">{filename}</span>
        {content && content.size !== undefined ?
          <span className="file-size">{formatSize(content ? content.size : 0)}</span>
          : ''}
      </div>
      {content && content.content &&
        <div className="file-content">
          <SyntaxHighlighter
            language={language}
            style={oneLight}
            showLineNumbers
            lineNumberStyle={{ minWidth: '3em', paddingRight: '1em', textAlign: 'right', color: '#999' }}
            customStyle={{ margin: 0, background: 'white', fontSize: '14px' }}
          >
            {content.content}
          </SyntaxHighlighter>
        </div>
      }
    </div>
  );
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} bytes`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
