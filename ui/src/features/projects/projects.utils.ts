import type { Project } from '@matrixhub/api-ts/v1alpha1/project.pb'

export function isProxyProject(registryUrl?: string) {
  return Boolean(registryUrl?.trim())
}

// Proxy projects mirror their upstream organization, so nothing can be written to them.
export function excludeProxyProjects(projects: Project[]) {
  return projects.filter(project => !project.enabledRegistry)
}

export function buildProxyDownloadCommand(
  hfEndpoint: string,
  organization: string | undefined,
  modelName: string,
  requiresToken = false,
) {
  const modelPath = [organization?.trim(), modelName.trim()].filter(Boolean).join('/')
  const token = requiresToken ? '\nexport HF_TOKEN="<your-matrixhub-token>"' : ''

  return `export HF_ENDPOINT=${hfEndpoint}${token}\nhf download ${modelPath}`
}
