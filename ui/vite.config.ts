import { tanstackRouter } from '@tanstack/router-plugin/vite'
import react from '@vitejs/plugin-react'
import {
  defineConfig,
  loadEnv,
} from 'vite'
import svgr from 'vite-plugin-svgr'
import tsconfigPaths from 'vite-tsconfig-paths'

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
