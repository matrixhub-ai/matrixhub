import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import importPlugin from 'eslint-plugin-import'
import { defineConfig, globalIgnores } from 'eslint/config'
import eslintReact from '@eslint-react/eslint-plugin'
import stylistic from '@stylistic/eslint-plugin'
import tanstackRouter from '@tanstack/eslint-plugin-router'
import jsonc from 'eslint-plugin-jsonc'

export default defineConfig(
  globalIgnores(['dist', 'node_modules', 'src/routeTree.gen.ts', '.vscode']),

  // =============================================
  // Base rules for all JS/TS files
  // =============================================
  {
    files: ['**/*.{ts,tsx,js,mjs,cjs}'],
    extends: [
      js.configs.recommended,
      stylistic.configs.recommended,
    ],
    rules: {
      '@stylistic/brace-style': ['error', '1tbs'],
      '@stylistic/quote-props': ['error', 'as-needed'],

      '@stylistic/quotes': ['error', 'single'],
      '@stylistic/comma-dangle': ['error', 'always-multiline'],
      '@stylistic/object-property-newline': ['error', { allowAllPropertiesOnSameLine: false }],
      '@stylistic/object-curly-spacing': ['error', 'always'],
      '@stylistic/object-curly-newline': ['error', {
        ObjectExpression: {
          consistent: true,
          minProperties: 2,
          multiline: false,
        },
        ObjectPattern: {
          consistent: true,
          minProperties: 2,
          multiline: false,
        },
        ImportDeclaration: {
          multiline: true,
          minProperties: 3,
        },
        ExportDeclaration: {
          multiline: true,
          minProperties: 3,
        },
      }],
      '@stylistic/array-bracket-newline': ['error', 'consistent'],
      '@stylistic/array-element-newline': ['error', 'consistent'],
      '@stylistic/padding-line-between-statements': [
        'error',
        {
          blankLine: 'always',
          prev: ['interface', 'type'],
          next: '*',
        },
        {
          blankLine: 'always',
          prev: ['const', 'let', 'var'],
          next: '*',
        },
        {
          blankLine: 'any',
          prev: ['const', 'let', 'var'],
          next: ['const', 'let', 'var'],
        },
        {
          blankLine: 'always',
          prev: '*',
          next: 'return',
        },
        {
          blankLine: 'always',
          prev: '*',
          next: 'function',
        },
      ],
      '@stylistic/max-len': [
        'error',
        {
          code: 150,
          tabWidth: 2,
          ignoreUrls: true,
          ignoreRegExpLiterals: true,
          ignoreStrings: true,
          ignoreTemplateLiterals: true,
          ignoreComments: true,
        },
      ],
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'no-debugger': 'warn',
      curly: ['error', 'all'],
    },
  },

  // =============================================
  // TypeScript + React files
  // =============================================
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      tseslint.configs.strict,
      tseslint.configs.stylistic,
      reactHooks.configs.flat['recommended-latest'],
      reactRefresh.configs.vite,
      eslintReact.configs['strict-typescript'],
      tanstackRouter.configs['flat/recommended'],
    ],
    languageOptions: {
      ecmaVersion: 2024,
      globals: globals.browser,
      parser: tseslint.parser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      import: importPlugin,
    },
    settings: {
      'import/resolver': {
        typescript: {
          alwaysTryTypes: true,
          project: './tsconfig.json',
        },
      },
    },
    rules: {
      // =============================================
      // TypeScript
      // =============================================
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          prefer: 'type-imports',
          fixStyle: 'separate-type-imports',
        },
      ],

      '@typescript-eslint/await-thenable': 'error',

      // =============================================
      // Import
      // =============================================
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
      'import/no-duplicates': 'error',

      // =============================================
      // @eslint-react: JSX Syntax
      // =============================================
      '@eslint-react/jsx-shorthand-fragment': 'error',
      '@eslint-react/jsx-shorthand-boolean': 'error',
      '@eslint-react/no-useless-fragment': 'error',

      // =============================================
      // @eslint-react: Performance Optimization
      // =============================================
      // Avoid literal objects for Context.Provider value
      '@eslint-react/no-unstable-context-value': 'error',
      // Avoid reference type literals for default props
      '@eslint-react/no-unstable-default-props': 'error',

      // =============================================
      // @eslint-react: DOM Security
      // =============================================
      // target="_blank" must have rel="noreferrer noopener"
      '@eslint-react/dom/no-unsafe-target-blank': 'error',
      // Disallow javascript: URL
      '@eslint-react/dom/no-script-url': 'error',
      // Disallow unsafe sandbox combinations
      '@eslint-react/dom/no-unsafe-iframe-sandbox': 'error',
    },
  },

  // =============================================
  // set runtime for specify files
  // =============================================
  {
    files: ['**/*.{js,mjs,cjs}', 'vite.config.ts'],
    languageOptions: {
      ecmaVersion: 2024,
      globals: globals.node,
    },
  },

  // =============================================
  // JSON files (strict, no comments allowed)
  // =============================================
  ...jsonc.configs['flat/recommended-with-json'],

  // =============================================
  // JSONC files (tsconfig, etc. - comments allowed)
  // =============================================
  ...jsonc.configs['flat/recommended-with-jsonc'],
  {
    files: ['**/*.json', '**/*.jsonc'],
    extends: [
      stylistic.configs.recommended,
    ],
    rules: {
      'jsonc/indent': ['error', 2],
      'jsonc/array-bracket-spacing': ['error', 'never'],
      'jsonc/no-irregular-whitespace': 'error',
      'jsonc/object-curly-spacing': [
        'error',
        'always',
      ],
      'jsonc/object-property-newline': 'error',
      'jsonc/object-curly-newline': ['error', {
        consistent: true,
        minProperties: 2,
        multiline: false,
      }],
      'jsonc/comma-style': 'error',
    },
  },
  {
    files: ['**/tsconfig*.json'],
    rules: {
      'jsonc/no-comments': 'off',
    },
  },
)
