#!/usr/bin/env node
/**
 * find-orphan-files.ts — Find orphaned (unused) static assets in the website.
 *
 * Usage:
 *   npx tsx scripts/find-orphan-files.ts
 */

import fs from 'node:fs';
import path from 'node:path';

const WEBSITE_DIR = path.resolve(__dirname, '..');

const ASSET_EXTENSIONS = new Set([
  '.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp', '.ico',
  '.pdf', '.woff', '.woff2', '.eot', '.ttf',
]);

const EXCLUDE_DIRS = new Set(['node_modules', '.docusaurus', 'build', '.git']);

const SOURCE_EXTENSIONS = new Set(['.md', '.mdx', '.ts', '.tsx', '.js', '.jsx', '.css', '.scss', '.json', '.yaml', '.yml']);

// Precompiled regex patterns
const ASSET_EXT_PATTERN = /\.(png|jpg|jpeg|gif|svg|webp|ico|pdf|woff|woff2|eot|ttf)/i;
const FRONT_MATTER_PATTERN = /^---\n[\s\S]*?\n---\n/;
const MARKDOWN_IMAGE_PATTERN = /!\[([^\]]*)\]\(([^)]+)\)/g;
const MARKDOWN_LINK_PATTERN = /\[([^\]]*)\]\(([^)]+)\)/g;
const JSX_IMG_PATTERN = /<img\s+[^>]*src=["']([^"']+)["'][^>]*>/gi;
const JSX_IMAGE_PATTERN = /<Image\s+[^>]*src=["']([^"']+)["'][^>]*>/gi;
const REQUIRE_PATTERN = /require\(['"]([^'"]+)['"]\)(?:\.default)?/gi;
const SRC_HREF_PATTERN = /(?:src|href)=["']([^"']+)["']/gi;
const CONFIG_VALUE_PATTERN = /(?:favicon|logo|src|image):\s*['"]([^"']+)['"]/gi;
const CSS_URL_PATTERN = /url\(["']?([^"')]+)["']?\)/gi;

function collectFiles(dir: string, extensions: Set<string>, excludeDirs: Set<string>): string[] {
  const results: string[] = [];
  if (!fs.existsSync(dir)) return results;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (excludeDirs.has(entry.name)) continue;
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...collectFiles(fullPath, extensions, excludeDirs));
    } else if (entry.isFile()) {
      if (extensions.has(path.extname(entry.name).toLowerCase())) {
        results.push(path.relative(WEBSITE_DIR, fullPath));
      }
    }
  }
  return results;
}

function normalizeUrl(url: string): string {
  return url.split('?')[0].split('#')[0];
}

function resolvePath(rawRef: string, sourceDir: string): string {
  // Skip URLs and data URIs
  if (/^https?:\/\//i.test(rawRef) || /^data:/i.test(rawRef)) return '';

  const ref = rawRef.trim();

  // @site/ prefix
  if (ref.startsWith('@site/')) {
    const candidate = path.join(WEBSITE_DIR, ref.replace(/^@site\//, ''));
    return fs.existsSync(candidate) ? path.relative(WEBSITE_DIR, candidate) : '';
  }

  // Absolute path from site root
  if (ref.startsWith('/')) {
    const relRef = ref.slice(1);
    // Docusaurus /img/ maps to static/img/
    let candidate = path.join(WEBSITE_DIR, 'static', relRef);
    if (fs.existsSync(candidate)) return path.relative(WEBSITE_DIR, candidate);
    // Try direct path
    candidate = path.join(WEBSITE_DIR, relRef);
    return fs.existsSync(candidate) ? path.relative(WEBSITE_DIR, candidate) : '';
  }

  // Relative paths (./ or ../)
  if (ref.startsWith('./') || ref.startsWith('../')) {
    let candidate = path.resolve(sourceDir, ref);
    if (fs.existsSync(candidate)) return path.relative(WEBSITE_DIR, candidate);

    // i18n special case
    if (sourceDir.includes('docusaurus-plugin-content-')) {
      const i18nMatch = sourceDir.match(/i18n\/([^/]+)\/docusaurus-plugin-content-(blog|docs)/);
      if (i18nMatch) {
        const refFileName = ref.replace(/^\.\//, '').replace(/^images\//, '');
        candidate = path.join(WEBSITE_DIR, 'i18n', i18nMatch[1], 'images', refFileName);
        if (fs.existsSync(candidate)) return path.relative(WEBSITE_DIR, candidate);
      }
    }
    return '';
  }

  // Plain path - try multiple locations
  const candidates = [
    path.resolve(sourceDir, ref),
    path.join(WEBSITE_DIR, ref),
    path.join(WEBSITE_DIR, 'static', ref),
  ];

  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) return path.relative(WEBSITE_DIR, candidate);
  }

  return '';
}

function extractRefsFromMarkdown(content: string, sourceDir: string): string[] {
  // Strip front matter
  const cleanContent = content.replace(FRONT_MATTER_PATTERN, '');

  const refs: string[] = [];

  // Markdown images: ![alt](url)
  for (const match of cleanContent.matchAll(MARKDOWN_IMAGE_PATTERN)) {
    const url = normalizeUrl(match[2]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  // Markdown links that might be assets
  for (const match of cleanContent.matchAll(MARKDOWN_LINK_PATTERN)) {
    const url = normalizeUrl(match[2]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  // JSX <img> tags
  for (const match of cleanContent.matchAll(JSX_IMG_PATTERN)) {
    const url = normalizeUrl(match[1]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  // JSX <Image> components
  for (const match of cleanContent.matchAll(JSX_IMAGE_PATTERN)) {
    const url = normalizeUrl(match[1]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  // require() calls
  for (const match of cleanContent.matchAll(REQUIRE_PATTERN)) {
    const url = normalizeUrl(match[1]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  return refs;
}

function extractRefsFromCode(content: string, sourceDir: string): string[] {
  const refs: string[] = [];

  // src/href attributes
  for (const match of content.matchAll(SRC_HREF_PATTERN)) {
    const url = normalizeUrl(match[1]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  // Config values (favicon, logo, etc.)
  for (const match of content.matchAll(CONFIG_VALUE_PATTERN)) {
    const url = normalizeUrl(match[1]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  // CSS url()
  for (const match of content.matchAll(CSS_URL_PATTERN)) {
    const url = normalizeUrl(match[1]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  // require() calls
  for (const match of content.matchAll(REQUIRE_PATTERN)) {
    const url = normalizeUrl(match[1]);
    if (ASSET_EXT_PATTERN.test(url)) {
      refs.push(resolvePath(url, sourceDir));
    }
  }

  return refs;
}

function collectAllAssets(): string[] {
  const results: string[] = [];
  for (const dir of ['static', 'docs', 'blog', 'i18n']) {
    const dirPath = path.join(WEBSITE_DIR, dir);
    if (fs.existsSync(dirPath)) {
      results.push(...collectFiles(dirPath, ASSET_EXTENSIONS, EXCLUDE_DIRS));
    }
  }
  return results;
}

function collectAllSourceFiles(): string[] {
  return collectFiles(WEBSITE_DIR, SOURCE_EXTENSIONS, EXCLUDE_DIRS);
}

function main() {
  console.log('🔍 Scanning for asset files...');
  const assetFiles = collectAllAssets();
  console.log(`   Found ${assetFiles.length} asset files`);

  console.log('🔍 Scanning source files for asset references...');
  const resolvedRefs = new Set<string>();

  const sourceFiles = collectAllSourceFiles();
  console.log(`   Parsing ${sourceFiles.length} source files...`);

  for (const file of sourceFiles) {
    const fullPath = path.join(WEBSITE_DIR, file);
    const sourceDir = path.dirname(fullPath);

    try {
      const content = fs.readFileSync(fullPath, 'utf8');
      const refs = file.endsWith('.md') || file.endsWith('.mdx')
        ? extractRefsFromMarkdown(content, sourceDir)
        : extractRefsFromCode(content, sourceDir);

      for (const ref of refs) {
        if (ref) resolvedRefs.add(ref);
      }
    } catch {
      // Skip unreadable files
    }
  }

  // Also check docusaurus.config.ts explicitly
  const configPath = path.join(WEBSITE_DIR, 'docusaurus.config.ts');
  if (fs.existsSync(configPath)) {
    const content = fs.readFileSync(configPath, 'utf8');
    for (const ref of extractRefsFromCode(content, path.dirname(configPath))) {
      if (ref) resolvedRefs.add(ref);
    }
  }

  console.log(`   Found ${resolvedRefs.size} unique resolved references`);

  // Find orphans
  const orphanFiles = assetFiles.filter(f => !resolvedRefs.has(f)).sort();
  const usedCount = assetFiles.length - orphanFiles.length;

  console.log('');
  console.log('==========================================');
  console.log(' Orphaned Asset Files Report');
  console.log('==========================================');
  console.log(` Total assets scanned: ${assetFiles.length}`);
  console.log(` Used files:           ${usedCount}`);
  console.log(` Orphaned files:       ${orphanFiles.length}`);
  console.log('==========================================');
  console.log('');

  if (orphanFiles.length === 0) {
    console.log('🎉 No orphaned files found!');
    return;
  }

  console.log('📋 Orphaned files:');
  console.log('');
  for (const file of orphanFiles) {
    console.log(`  ❌ ${file}`);
  }
  console.log('');
  console.log('Run the following to delete all orphaned files:');
  console.log('');
  console.log(`  cd ${WEBSITE_DIR}`);
  for (const file of orphanFiles) {
    console.log(`  rm '${file}'`);
  }
  console.log('');
  console.log(`❌ Found ${orphanFiles.length} orphaned file(s). Remove them before proceeding.`);
  process.exit(1);
}

main();