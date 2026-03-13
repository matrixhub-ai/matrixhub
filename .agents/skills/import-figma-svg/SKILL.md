---
name: import-figma-svg
description: 'Download SVG assets from Figma MCP asset URLs. Use when: export SVG from Figma, import icons, download Figma assets, or batch download SVGs extracted from Figma nodes.'
argument-hint: 'Figma URL or node ID, asset URL(s), target file name(s), optional output directory'
---

# Import Figma SVG

Download SVG assets from Figma MCP asset URLs into a local directory.

## Scope

This skill only covers extracting Figma MCP asset URLs and downloading valid `.svg` files.

Project-specific placement, naming, import style, and component-usage guidance should come from the calling agent or the target repo's docs.

## When to Use

- User asks to export, download, import, or save icons/SVGs from a Figma design
- User provides a Figma node URL or asset URL and wants the SVG saved locally
- Batch importing multiple SVG assets from a Figma file

## Procedure

1. Use Figma MCP to inspect the target node when the user provides a design URL or node ID.
2. If the user gives a design node instead of a direct asset URL, call `get_design_context` and extract the asset URLs from the returned constants.
3. Identify which extracted URLs are actual SVG assets. If a file is raster content, stop and report that it is not an SVG.
4. Choose an output directory for the current task. When the destination matters, pass `--dir` explicitly instead of relying on the script default.
5. Download the asset(s):

### Single asset

```bash
node .agents/skills/import-figma-svg/scripts/import-figma-svg.mjs \
  --url "<asset-url>" --name "<file-name>" --dir "<output-dir>"
```

### Batch download (parallel)

```bash
node .agents/skills/import-figma-svg/scripts/import-figma-svg.mjs \
  --items '[{"url":"<url-1>","name":"icon-a"},{"url":"<url-2>","name":"icon-b"}]' \
  --dir "<output-dir>"
```

Options:
- `--dir <path>` — output directory (default: current working directory)
- `--concurrency <n>` — max parallel downloads (default: 4)

6. Before replacing an existing SVG in a codebase, check current usages with `rg`.

## Notes

- Figma MCP asset URLs expire — do not store them as persistent references in code.
- The bundled script validates that the downloaded response starts with `<svg>` and fails on non-SVG assets.
- Do not encode repo-specific placement or framework-specific import guidance in this skill. The calling agent should add those instructions when needed.
- Do not delete an existing SVG in a codebase until all imports have been updated.
- Only convert `fill` or `stroke` to `currentColor` when the target usage clearly expects theme-driven icon color.
