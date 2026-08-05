import { tanstackRouter } from '@tanstack/router-plugin/vite'
import react from '@vitejs/plugin-react'
import {
  defineConfig,
  loadEnv,
} from 'vite'
import svgr from 'vite-plugin-svgr'
import tsconfigPaths from 'vite-tsconfig-paths'

function manualChunks(id: string) {
  if (!id.includes('/node_modules/')) {
    return undefined
  }

  if (
    id.includes('/node_modules/mantine-react-table/')
    || id.includes('/node_modules/@tanstack/table')
  ) {
    return 'vendor-tanstack'
  }

  if (
    id.includes('/node_modules/react/')
    || id.includes('/node_modules/react-dom/')
    || id.includes('/node_modules/scheduler/')
  ) {
    return 'vendor-react'
  }

  if (
    id.includes('/node_modules/@mantine/')
    || id.includes('/node_modules/@floating-ui/')
  ) {
    return 'vendor-mantine'
  }

  if (id.includes('/node_modules/@tanstack/')) {
    return 'vendor-tanstack'
  }

  if (id.includes('/node_modules/@tabler/icons-react/')) {
    return 'vendor-icons'
  }

  if (
    id.includes('/node_modules/@monaco-editor/')
    || id.includes('/node_modules/monaco-editor/')
  ) {
    return 'vendor-monaco'
  }

  if (id.includes('/node_modules/diff2html/')) {
    return 'vendor-diff'
  }

  if (
    id.includes('/node_modules/i18next')
    || id.includes('/node_modules/dayjs/')
  ) {
    return 'vendor-i18n'
  }

  return 'vendor'
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    base: env.VITE_UI_BASE_PATH ?? '/',
    resolve: {
      dedupe: [
        'react',
        'react-dom',
        '@mantine/core',
        '@mantine/hooks',
        '@mantine/dates',
        'dayjs',
      ],
    },
    optimizeDeps: {
      include: ['mantine-react-table'],
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks,
        },
      },
    },
    plugins: [
      // Please make sure that '@tanstack/router-plugin' is passed before '@vitejs/plugin-react'
      tanstackRouter({
        target: 'react',
        autoCodeSplitting: true,
      }),
      react({
        babel: {
          plugins: [['babel-plugin-react-compiler']],
        },
      }),
      tsconfigPaths(),
      svgr(),
    ],
    server: {
      proxy: {
        '/api': {
          target: env.VITE_APP_API_URL,
        },
        '/_assets_proxy_': {
          target: env.VITE_APP_API_URL,
          changeOrigin: true,
          rewrite: path => path.replace(/^\/_assets_proxy_\//, ''),
        },
      },
    },
  }
})
