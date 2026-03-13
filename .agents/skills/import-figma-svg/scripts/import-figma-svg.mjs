#!/usr/bin/env node

import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

function printUsage() {
  process.stdout.write(`Usage:
  Single:
    node path/to/import-figma-svg.mjs --url <url> --name <file-name> [--dir <output-dir>]

  Batch:
    node path/to/import-figma-svg.mjs --items '<json-array>' [--dir <output-dir>] [--concurrency <n>]

  The JSON array format: [{"url":"...","name":"..."},...]

Options:
  --dir          Output directory (default: current working directory)
  --concurrency  Max parallel downloads for batch mode (default: 4)

Examples:
  node path/to/import-figma-svg.mjs --url "https://figma.com/api/mcp/asset/..." --name admin-users-nav --dir ./svgs
  node path/to/import-figma-svg.mjs --items '[{"url":"https://...","name":"icon-a"},{"url":"https://...","name":"icon-b"}]' --dir ./svgs --concurrency 6
`)
}

function parseArgs(argv) {
  const args = {
    concurrency: 4,
    dir: '.',
    items: '',
    name: '',
    url: '',
  }

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]

    if (arg === '--help' || arg === '-h') {
      printUsage()
      process.exit(0)
    }

    if (arg === '--url') {
      args.url = argv[index + 1] ?? ''
      index += 1
      continue
    }

    if (arg === '--name') {
      args.name = argv[index + 1] ?? ''
      index += 1
      continue
    }

    if (arg === '--dir') {
      args.dir = argv[index + 1] ?? ''
      index += 1
      continue
    }

    if (arg === '--items') {
      args.items = argv[index + 1] ?? ''
      index += 1
      continue
    }

    if (arg === '--concurrency') {
      args.concurrency = Number.parseInt(argv[index + 1] ?? '4', 10)
      index += 1
      continue
    }

    throw new Error(`Unknown argument: ${arg}`)
  }

  return args
}

function resolveItems(args) {
  if (args.items) {
    let parsed

    try {
      parsed = JSON.parse(args.items)
    }
    catch {
      throw new Error('--items must be valid JSON')
    }

    if (!Array.isArray(parsed) || parsed.length === 0) {
      throw new Error('--items must be a non-empty JSON array')
    }

    for (const item of parsed) {
      if (!item.url || !item.name) {
        throw new Error('Each item must have "url" and "name" fields')
      }
    }

    return parsed
  }

  if (!args.url) {
    throw new Error('Missing required --url (or use --items for batch)')
  }

  if (!args.name) {
    throw new Error('Missing required --name (or use --items for batch)')
  }

  return [{ name: args.name, url: args.url }]
}

function normalizeFileName(name) {
  const trimmed = name.trim()

  if (!trimmed) {
    throw new Error('File name cannot be empty')
  }

  return trimmed.endsWith('.svg')
    ? trimmed
    : `${trimmed}.svg`
}

function validateSvg(content, name) {
  const trimmed = content.trimStart()

  if (!trimmed.startsWith('<svg')) {
    throw new Error(`Downloaded content for "${name}" is not an SVG document`)
  }
}

async function downloadOne({ name, outputDir, url }) {
  const response = await fetch(url)

  if (!response.ok) {
    throw new Error(`[${name}] Download failed: ${response.status} ${response.statusText}`)
  }

  const svg = await response.text()

  validateSvg(svg, name)

  const outputFile = path.join(outputDir, normalizeFileName(name))

  await writeFile(outputFile, svg, 'utf8')

  return outputFile
}

async function runWithConcurrency(tasks, concurrency) {
  const results = []
  let index = 0

  async function worker() {
    while (index < tasks.length) {
      const current = index++
      results[current] = await tasks[current]()
        .then(value => ({ status: 'fulfilled', value }))
        .catch(reason => ({ status: 'rejected', reason }))
    }
  }

  const workers = Array.from(
    { length: Math.min(concurrency, tasks.length) },
    () => worker(),
  )

  await Promise.all(workers)

  return results
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  const items = resolveItems(args)
  const outputDir = path.resolve(args.dir)

  await mkdir(outputDir, { recursive: true })

  const tasks = items.map(item => () => downloadOne({
    name: item.name,
    outputDir,
    url: item.url,
  }))

  const results = await runWithConcurrency(tasks, args.concurrency)

  let hasError = false

  for (const [i, result] of results.entries()) {
    if (result.status === 'fulfilled') {
      process.stdout.write(`OK  ${result.value}\n`)
    }
    else {
      const message = result.reason instanceof Error
        ? result.reason.message
        : String(result.reason)
      process.stderr.write(`ERR ${items[i].name}: ${message}\n`)
      hasError = true
    }
  }

  if (hasError) {
    process.exit(1)
  }
}

main().catch((error) => {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exit(1)
})
