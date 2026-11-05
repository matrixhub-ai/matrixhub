export function isProxyProject(registryUrl?: string) {
  return Boolean(registryUrl?.trim())
}

export function buildProxyDownloadCommand(
  hfEndpoint: string,
  organization: string | undefined,
  modelName: string,
  requiresToken = false,
  registryType?: string,
) {
  const modelPath = [organization?.trim(), modelName.trim()].filter(Boolean).join('/')

  if (registryType === 'REGISTRY_TYPE_MODELSCOPE') {
    const token = requiresToken ? '\nexport MODELSCOPE_API_TOKEN="<your-matrixhub-token>"' : ''

    return `export MODELSCOPE_ENDPOINT=${hfEndpoint}${token}\nms-hub download ${modelPath}`
  }
  const token = requiresToken ? '\nexport HF_TOKEN="<your-matrixhub-token>"' : ''

  return `export HF_ENDPOINT=${hfEndpoint}${token}\nhf download ${modelPath}`
}
