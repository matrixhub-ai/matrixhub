import type { Plugin } from 'vite'

interface DevDeployPluginOptions {
  enabled: boolean
  serviceWorkerPath: string
}

const MAIN_ENTRY_SCRIPT_PATTERN = /<script\s+type="module"\s+src="\/src\/main\.tsx"><\/script>/

export function devDeployPlugin({
  enabled,
  serviceWorkerPath,
}: DevDeployPluginOptions): Plugin {
  return {
    name: 'matrixhub-dev-deploy',
    transformIndexHtml: {
      order: 'pre',
      handler(html) {
        if (!enabled) {
          return html
        }

        const bootstrapScript = [
          '<script type="module">',
          '  import { bootstrapDevDeploy } from "/src/dev-deploy/bootstrap.ts"',
          '  await bootstrapDevDeploy({',
          `    serviceWorkerPath: ${JSON.stringify(serviceWorkerPath)},`,
          '  })',
          '</script>',
        ].join('\n')

        return html.replace(MAIN_ENTRY_SCRIPT_PATTERN, bootstrapScript)
      },
    },
  }
}
