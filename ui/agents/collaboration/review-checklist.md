# Review Checklist

Use this checklist to quickly judge whether a change still follows the current UI conventions.

## Must Check

- Is the change small enough in scope and clear enough in intent?
- If this is a new page, was page planning done first?
- Does the route file still mainly own route responsibilities?
- If the change includes a form, does it use `TanStack Form` with `Zod` based validation and keep Mantine as the UI layer instead of `useForm`, ad-hoc form state, or a different validation scheme?
- If the change includes a data table, does it use the project's wrapped table component or adapter instead of wiring the table library directly in the page?
- Was new user-facing copy added to locale files?
- Does the implementation continue the existing Mantine and project patterns?
- If colors or visual states were adjusted, does the change rely on component semantics first, then semantic tokens, instead of copying Figma colors directly?
- If the change created or changed a shared wrapper, shared component convention, or other stable project pattern, were the relevant `ui/agents/` docs updated in the same change?

## When Documentation Should Be Updated

- The default stack changes
- The directory boundaries change
- The team adds a new stable convention
- A new reusable example is needed

## Basic Validation

- Run at least the relevant `pnpm lint`
- Run at least the relevant `pnpm typecheck`
- Any unverified part should be called out clearly in the final note
