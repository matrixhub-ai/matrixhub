export function isProxyProject(registryUrl?: string) {
  return Boolean(registryUrl?.trim())
}

export function buildProxyDownloadCommand(
  hfEndpoint: string,
  organization: string | undefined,
  modelName: string,
) {
  const modelPath = [organization?.trim(), modelName.trim()].filter(Boolean).join('/')

  return `export HF_ENDPOINT=${hfEndpoint}\nhf download ${modelPath}`
}
