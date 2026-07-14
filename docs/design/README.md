# Technical Design Docs

This directory holds technical design docs for non-trivial MatrixHub features and changes.

For a **simple change**, describe the approach directly in the issue/PR checklist — no doc needed.
For a **complex change**, add a design doc here (or an external doc) and get maintainer review **before coding**, then link it from the related Backend Task issue.

## Conventions

- One Markdown file per feature: `docs/design/<feature>-design.md` (kebab-case, e.g. `cleanup-design.md`).
- Keep it focused: Context / Motivation, Proposed design, API changes, Test plan, Alternatives.
- **Architecture diagrams** are welcome — use Mermaid in Markdown, or add image files (e.g. PNG) alongside the doc in `docs/design/`.
- Update the doc as the design evolves; it should reflect what was actually built.

## Index

- [Cleanup](./cleanup-design.md)
- [Jobserver](./jobserver-design.md)
<!-- Add new design docs here -->
