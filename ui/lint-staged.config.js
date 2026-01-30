import fs from 'fs/promises'
import path from 'path'
import { ESLint } from 'eslint'
import ts from 'typescript'

const projectRoot = process.cwd()

const prepareTsConfigForLint = (config) => {
  if (!config.compilerOptions) {
    config.compilerOptions = {}
  }
  delete config.compilerOptions.composite
  delete config.compilerOptions.tsBuildInfoFile
  const currentBaseUrl = config.compilerOptions.baseUrl ?? '.'

  config.compilerOptions.baseUrl = path.resolve(projectRoot, currentBaseUrl)

  return config
}

// Try to use TypeScript's file expansion ability to parse include patterns into specific file path sets
const expandTsIncludesToSet = (config) => {
  const emptySet = new Set()

  if (!config?.include?.length) {
    return emptySet
  }
  if (!ts || !ts.sys || !ts.sys.readDirectory) {
    return emptySet
  }

  try {
    // ts.sys.readDirectory(baseDir, extensions, excludes, includes)
    const files = ts.sys.readDirectory(projectRoot, ['ts', 'tsx', 'js', 'jsx'], config?.exclude, config.include)

    return new Set(files.map(f => path.resolve(f)))
  } catch {
    return emptySet
  }
}

// Simple function to strip comments from JSON (JSONC)
const parseJSONC = (str) => {
  // Remove single line // and multi-line /* */ comments
  // Note: This regex is simple and works for standard cases but might catch URLs etc. in strings.
  // For tsconfig structure it is generally safe.
  const cleanJson = str.replace(/\/\*[\s\S]*?\*\/|([^\\:]|^)\/\/.*$/gm, '$1')

  return JSON.parse(cleanJson)
}

const generateTSConfig = async (stagedFiles) => {
  if (!stagedFiles.length) {
    return []
  }

  // Read app and node configurations
  const [tsAppConfigRaw, tsNodeConfigRaw] = await Promise.all([
    fs.readFile(path.resolve('tsconfig.app.json'), 'utf8'),
    fs.readFile(path.resolve('tsconfig.node.json'), 'utf8'),
  ])

  const tsAppConfig = prepareTsConfigForLint(parseJSONC(tsAppConfigRaw))
  const tsNodeConfig = prepareTsConfigForLint(parseJSONC(tsNodeConfigRaw))

  const nodeFilesSet = expandTsIncludesToSet(tsNodeConfig)
  const appFilesSet = expandTsIncludesToSet(tsAppConfig)

  const resolvedStagedFiles = stagedFiles.map(f => path.resolve(f))

  const nodeFilesToCheck = resolvedStagedFiles.filter(f => nodeFilesSet.has(f))
  const appFilesToCheck = resolvedStagedFiles.filter(f => appFilesSet.has(f))

  const commands = []

  if (appFilesToCheck.length > 0) {
    // dynamic include for app + exclude node include to avoid duplicate checks
    // Explicitly include d.ts to ensure global types are available
    const srcDir = path.resolve('src/**/*.d.ts')

    tsAppConfig.include = [...appFilesToCheck, srcDir]

    const tsAppLintPath = path.resolve('node_modules/.tmp/tsconfig.app.lint.json')

    // Ensure directory exists
    await fs.mkdir(path.dirname(tsAppLintPath), {
      recursive: true,
    })
    await fs.writeFile(tsAppLintPath, JSON.stringify(tsAppConfig, null, 2))
    commands.push(`tsc -b ${tsAppLintPath} --noEmit`)
  }

  if (nodeFilesToCheck.length > 0) {
    tsNodeConfig.include = nodeFilesToCheck
    const tsNodeLintPath = path.resolve('node_modules/.tmp/tsconfig.node.lint.json')

    await fs.mkdir(path.dirname(tsNodeLintPath), {
      recursive: true,
    })
    await fs.writeFile(tsNodeLintPath, JSON.stringify(tsNodeConfig, null, 2))
    commands.push(`tsc -b ${tsNodeLintPath} --noEmit`)
  }

  return commands
}

// Handle eslint warning: File ignored by default.
const removeEslintIgnored = async (stagedFilenames) => {
  if (stagedFilenames.length === 0) {
    return ''
  }
  const eslint = new ESLint()
  const isIgnored = await Promise.all(
    stagedFilenames.map(file => eslint.isPathIgnored(file)),
  )
  const filteredFiles = stagedFilenames.filter((_, i) => !isIgnored[i])

  if (filteredFiles.length === 0) {
    return ''
  }

  return `eslint --max-warnings=0 --fix ${filteredFiles
    .map(item => `'${item}'`)
    .join(' ')}`
}

export default {
  '*.{ts,tsx}': async (files) => {
    // For TS files, we run both eslint and tsc
    const eslintCmd = await removeEslintIgnored(files)
    const tscCmds = await generateTSConfig(files)

    const commands = [...tscCmds]

    if (eslintCmd) {
      commands.push(eslintCmd)
    }

    return commands
  },
  '*.{js,jsx,mjs,cjs}': async (files) => {
    // For JS files, only run eslint (unless strict js check is enabled, assuming mainly lint here)
    const eslintCmd = await removeEslintIgnored(files)

    return eslintCmd ? [eslintCmd] : []
  },
  '*.{json,jsonc}': async (files) => {
    const eslintCmd = await removeEslintIgnored(files)

    return eslintCmd ? [eslintCmd] : []
  },
}
