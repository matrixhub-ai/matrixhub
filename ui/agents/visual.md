# Visual Conventions

Mantine usage, color decisions, and Figma/SVG asset handling.

---

## Mantine

- Prefer Mantine layout primitives: `AppShell`, `Stack`, `Group`, `Flex`, `Box`.
- Prefer `Text` and `Title` for text rendering.
- Prefer Mantine props and theme tokens for spacing — not hardcoded pixel values.
- When styling a Mantine component that already expresses state or hierarchy (e.g. `Button` variants, `Badge` colors), first use the component's own semantics and stateful API before adding explicit colors.
- Do not pile raw `div`s or inline styles where Mantine already covers the use case.
- Do not hardcode colors, font sizes, or spacing values as the default approach.

---

## Color decision order

For any color, walk the ladder top-to-bottom and stop at the first match:

1. **Component-native semantics** — does the Mantine component already express this via variant / color / state?
2. **Project semantic token** — does the design system already define an exact semantic token for this intent?
3. **Mantine semantic token / CSS variable** — e.g. `var(--mantine-primary-color-filled)`, `var(--mantine-primary-color-light)`, `var(--mantine-color-text)`, `var(--mantine-color-dimmed)`.
4. **Palette-index reference** — e.g. `var(--mantine-color-cyan-6)`, `gray.7`. **Last resort only**, because indexed colors adapt poorly when dark theme support is added later.

Do not copy Figma colors mechanically when the same intent is already covered by component defaults or existing semantic tokens. Use explicit Figma-driven color overrides only after confirming the design intent is not represented by any of steps 1–3.

---

## Figma assets

When implementing from Figma:

- Committed SVG assets belong under `src/assets/svgs`.
- Before committing a new icon SVG from Figma, first check whether `@tabler/icons-react` already provides a suitable icon. Prefer the package when it does.
- Only add a committed SVG icon when no suitable Tabler icon exists, or when the asset is brand-specific and must preserve its exact shape.
- When a Figma icon has no Tabler equivalent, confirm the download or save step before adding a new SVG asset.
- Do not leave reusable SVG markup inline in route or component files unless the SVG must be generated dynamically.
- Temporary exports and cropped comparisons may live under `.planning/<task>/` (gitignored).
