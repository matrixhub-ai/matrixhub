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
  // JavaScript files (non-TypeScript)
  // =============================================
  {
    files: ['**/*.{js,mjs,cjs}'],
    extends: [
      js.configs.recommended,
      stylistic.configs.recommended,
    ],
    languageOptions: {
      ecmaVersion: 2024,
      globals: globals.node,
    },
  },

  // =============================================
  // TypeScript + React files
  // =============================================
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.strict,
      tseslint.configs.stylistic,
      stylistic.configs.recommended,
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
      '@typescript-eslint': tseslint.plugin,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      import: importPlugin,
      '@stylistic': stylistic,
      '@tanstack/router': tanstackRouter,
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
      // React Hooks (Official)
      // =============================================
      ...reactHooks.configs.recommended.rules,

      // =============================================
      // React Refresh (HMR)
      // =============================================
      'react-refresh/only-export-components': [
        'warn',
        {
          allowConstantExport: true,
        },
      ],

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
      // Disallow nested component definitions (recreated on each render)
      '@eslint-react/no-nested-component-definitions': 'error',
      // Disallow nested lazy declarations
      '@eslint-react/no-nested-lazy-component-declarations': 'error',
      // Avoid literal objects for Context.Provider value
      '@eslint-react/no-unstable-context-value': 'error',
      // Avoid reference type literals for default props
      '@eslint-react/no-unstable-default-props': 'error',
      // Use lazy initialization when passing functions to useState
      '@eslint-react/prefer-use-state-lazy-initialization': 'warn',

      // =============================================
      // @eslint-react: Code Quality
      // =============================================
      // Recommended to destructure props
      '@eslint-react/prefer-destructuring-assignment': 'warn',
      // Disallow array index as key
      '@eslint-react/no-array-index-key': 'warn',
      // Disallow children as prop
      '@eslint-react/no-children-prop': 'error',

      // =============================================
      // @eslint-react: DOM Security
      // =============================================
      // target="_blank" must have rel="noreferrer noopener"
      '@eslint-react/dom/no-unsafe-target-blank': 'error',
      // Disallow javascript: URL
      '@eslint-react/dom/no-script-url': 'error',
      // iframe must have sandbox
      '@eslint-react/dom/no-missing-iframe-sandbox': 'warn',
      // Disallow unsafe sandbox combinations
      '@eslint-react/dom/no-unsafe-iframe-sandbox': 'error',
      // button must have type
      '@eslint-react/dom/no-missing-button-type': 'warn',

      // =============================================
      // @eslint-react: Web API Leak Prevention
      // =============================================
      '@eslint-react/web-api/no-leaked-event-listener': 'warn',
      '@eslint-react/web-api/no-leaked-interval': 'warn',
      '@eslint-react/web-api/no-leaked-timeout': 'warn',
      '@eslint-react/web-api/no-leaked-resize-observer': 'warn',

      // =============================================
      // @eslint-react: Hooks Enhancement
      // =============================================
      // Avoid direct setState calls in useEffect (should be in callback)
      '@eslint-react/hooks-extra/no-direct-set-state-in-use-effect': 'warn',

      // =============================================
      // @eslint-react: Naming Convention
      // =============================================
      // useState destructuring: [value, setValue]
      '@eslint-react/naming-convention/use-state': 'warn',
      // useRef naming: xxxRef
      '@eslint-react/naming-convention/ref-name': 'warn',
    },
  },

  // =============================================
  // Base rules for all JS/TS files
  // =============================================
  {
    files: ['**/*.{ts,tsx,js,mjs,cjs}'],
    plugins: {
      '@stylistic': stylistic,
    },
    rules: {
      '@stylistic/eol-last': ['error', 'always'],
      '@stylistic/quote-props': ['error', 'as-needed'],
      // Migrated from legacy config
      '@stylistic/quotes': ['error', 'single'],
      '@stylistic/jsx-quotes': ['error', 'prefer-double'],
      '@stylistic/brace-style': ['error', '1tbs', {
        allowSingleLine: false,
      }],
      '@stylistic/comma-dangle': ['error', 'always-multiline'],
      '@stylistic/comma-spacing': ['error', {
        before: false,
        after: true,
      }],
      '@stylistic/object-property-newline': ['error', {
        allowAllPropertiesOnSameLine: false,
      }],
      '@stylistic/object-curly-spacing': ['error', 'always'],
      '@stylistic/object-curly-newline': ['error', {
        ObjectExpression: {
          multiline: true,
          minProperties: 1,
        },
        ObjectPattern: {
          multiline: true,
          minProperties: 1,
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
      '@stylistic/max-statements-per-line': ['error', {
        max: 1,
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
      'no-console': ['warn', {
        allow: ['warn', 'error'],
      }],
      'no-debugger': 'warn',
      curly: ['error', 'all'],
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
    plugins: {
      '@stylistic': stylistic,
    },
    rules: {
      'jsonc/indent': ['error', 2],
      '@stylistic/eol-last': ['error', 'always'],
    },
  },
  {
    files: ['**/tsconfig*.json'],
    rules: {
      'jsonc/no-comments': 'off',
    },
  },
)
