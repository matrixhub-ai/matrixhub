/** How a proxy-project model is downloaded: pulled by the MatrixHub server, or with the hf CLI. */
export type ProxyDownloadMethod = 'server' | 'cli'

export function isProxyProject(registryUrl?: string) {
  return Boolean(registryUrl?.trim())
}

/** Users who cannot pull on the server keep the CLI-only flow. */
export function resolveProxyDownloadMethod(canPullOnServer: boolean, method: ProxyDownloadMethod): ProxyDownloadMethod {
  return canPullOnServer ? method : 'cli'
}

/** Locale keys (under `projects.detail.proxyDownload.steps`) of the download steps, in order. */
export function getProxyDownloadStepKeys(canPullOnServer: boolean) {
  return canPullOnServer
    ? ['modelName', 'method', 'completion'] as const
    : ['modelName', 'command', 'completion'] as const
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
