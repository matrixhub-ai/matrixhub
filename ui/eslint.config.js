import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'
import stylistic from '@stylistic/eslint-plugin'
import importPlugin from 'eslint-plugin-import'
import routerPlugin from '@tanstack/eslint-plugin-router'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import eslintReact from '@eslint-react/eslint-plugin'


export default defineConfig([
  globalIgnores(['dist', 'src/routeTree.gen.ts']), 
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      tseslint.configs.strict,
      tseslint.configs.strictTypeChecked,
      reactHooks.configs.flat['recommended-latest'],
      reactRefresh.configs.vite,
      eslintReact.configs['strict-typescript'],
      routerPlugin.configs['flat/recommended'],
      tseslint.configs.stylistic,
      stylistic.configs.recommended,
    ],
    plugins: {
      import: importPlugin,
    },
    settings: {
      'import/resolver': {
        typescript: {
          project: ['./tsconfig.app.json'],
          alwaysTryTypes: true,
        },
      },
    },
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      'import/no-duplicates': 'error',
      'import/order': [
        'error',
        {
          groups: [
            'builtin',
            'external',
            'internal',
            ['parent', 'sibling'],
            'index',
            'type',
          ],
          'newlines-between': 'always',
          alphabetize: {
            order: 'asc',
            caseInsensitive: true,
          },
        },
      ],
    },
  },
])