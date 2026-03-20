---
name: import-figma-svg
description: 'Inspect Figma SVG assets, prefer @tabler/icons-react when available, and download SVGs only as a fallback.'
argument-hint: 'Figma URL or node ID, asset URL(s), target file name(s), optional output directory'
---

# Import Figma SVG

Inspect Figma SVG assets from MCP URLs, prefer an existing `@tabler/icons-react` icon when possible, and download SVG files only when there is no suitable library match.

## Scope

This skill covers:

- inspecting Figma nodes or asset URLs
- checking whether a reusable icon should use `@tabler/icons-react` first
- downloading valid `.svg` files only when needed

Project-specific placement, naming, import style, and component-usage guidance should come from the calling agent or the target repo's docs.

## When to Use

- User asks to export, download, import, or save icons/SVGs from a Figma design
- User provides a Figma node URL or asset URL and wants the SVG saved locally
- Batch importing multiple SVG assets from a Figma file
- User needs help deciding whether a Figma icon should be downloaded at all

## Procedure

1. Use Figma MCP to inspect the target node when the user provides a design URL or node ID.
2. Decide whether the target is a simple reusable icon or a brand, illustration, or product-specific asset that likely needs to stay custom.
3. For reusable icons, check `@tabler/icons-react` first:
   - search by layer name, asset name, and obvious synonyms
   - inspect likely exports when the match is not exact by name
   - if a good Tabler match exists, recommend or use that icon directly and stop; do not download the SVG
4. If the user gives a design node instead of a direct asset URL, call `get_design_context` and extract the asset URLs from the returned constants.
5. Identify which extracted URLs are actual SVG assets. If a file is raster content, stop and report that it is not an SVG.
6. If no suitable Tabler icon exists and the asset should still be saved locally, ask whether the user wants the SVG downloaded. Treat automatic download as the last step, not the default.
7. After the user confirms the fallback download, choose an output directory for the current task. When the destination matters, pass `--dir` explicitly instead of relying on the script default.
8. Download the asset(s):

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

9. Before replacing an existing SVG in a codebase, check current usages with `rg`.

## Notes

- A matching Tabler icon should win over downloading a new generic SVG icon.
- Automatic download should happen only after the Tabler check fails and the user still wants a committed asset.
- Figma MCP asset URLs expire — do not store them as persistent references in code.
- The bundled script validates that the downloaded response starts with `<svg>` and fails on non-SVG assets.
- Do not encode repo-specific placement or framework-specific import guidance in this skill. The calling agent should add those instructions when needed.
- Do not delete an existing SVG in a codebase until all imports have been updated.
- Only convert `fill` or `stroke` to `currentColor` when the target usage clearly expects theme-driven icon color.
