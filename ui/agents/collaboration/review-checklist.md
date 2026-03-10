# Review Checklist

Use this checklist to quickly judge whether a change still follows the current UI conventions.

## Must Check

- Is the change small enough in scope and clear enough in intent?
- If this is a new page, was page planning done first?
- Does the route file still mainly own route responsibilities?
- Was new user-facing copy added to locale files?
- Does the implementation continue the existing Mantine and project patterns?

## When Documentation Should Be Updated

- The default stack changes
- The directory boundaries change
- The team adds a new stable convention
- A new reusable example is needed

## Basic Validation

- Run at least the relevant `pnpm lint`
- Run at least the relevant `pnpm typecheck`
- Any unverified part should be called out clearly in the final note
