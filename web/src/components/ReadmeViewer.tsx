import ReactMarkdown from 'react-markdown';
import { FaReadme } from 'react-icons/fa';
import './ReadmeViewer.css';

interface ReadmeViewerProps {
  content: string;
}

export function ReadmeViewer({ content }: ReadmeViewerProps) {
  return (
    <div className="readme-viewer">
      <div className="readme-header">
        <FaReadme />
        <span className="readme-title">README.md</span>
      </div>
      <div className="readme-content">
        <ReactMarkdown>{content}</ReactMarkdown>
      </div>
    </div>
  );
}
