import { FaFolder } from 'react-icons/fa';
import { SiGithub, SiGitlab, SiBitbucket, SiHuggingface } from "react-icons/si";
import type { JSX } from 'react/jsx-dev-runtime';


export function prefixIcon(prefix: string): JSX.Element {
  if (prefix.startsWith('github.com/')) {
    return <SiGithub />;
  }

  if (prefix.startsWith('gitlab.com/')) {
    return <SiGitlab />;
  }

  if (prefix.startsWith('bitbucket.org/')) {
    return <SiBitbucket />;
  }

  if (prefix.startsWith('huggingface.co/')) {
    return <SiHuggingface />;
  }

  return <FaFolder />;
}
