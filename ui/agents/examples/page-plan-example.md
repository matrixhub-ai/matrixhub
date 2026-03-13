# Page Plan Example: Models List Within A Project

This example shows how to combine the requirement, `figma`, existing code, and API planning into a page plan that can be executed directly.

## 1. Inputs

- Route / requirement:
  - Route: `src/routes/(auth)/(app)/projects/$projectId/models/index.tsx`
  - Goal: show the models list under a project and provide an entry point for creating a new model
- Figma:
  - Confirms the visual hierarchy: header, action area, table or list body, and empty state area
  - Confirms layout rhythm: spacing between header and list, action button placement on the top right, and list items shown as either table rows or cards
  - If the design includes reusable SVG assets that should live in the repo, call `$import-figma-svg` to extract asset URLs from Figma MCP and download them
  - For committed UI assets in this repo, store downloaded SVGs under `src/assets/svgs`; for temporary comparisons, store them under `.planning/<task>/`
  - If the target UI already uses SVG React components, note that expected import style in the page plan, for example `?react`
- Related existing pages:
  - `src/routes/(auth)/(app)/projects/index.tsx`
  - `src/routes/(auth)/(app)/models/index.tsx`
  - `src/routes/(auth)/route.tsx`

## 2. Key Decisions

- Route file:
  - Keep it at `src/routes/(auth)/(app)/projects/$projectId/models/index.tsx`
  - Let it only read `projectId`, mount the page, and keep route-level metadata
- Feature page:
  - Add `src/features/projects/pages/ProjectModelsPage.tsx`
- Need new components:
  - Do not over-split in the first version
  - Keep the header area, toolbar area, and list area inside `ProjectModelsPage` first
  - If the list item structure becomes clearly reusable, extract `components/ModelList.tsx`
- Need new locale files:
  - `src/locales/en/projects/models.json`
  - `src/locales/zh/projects/models.json`

## 3. Layout Plan

- Main sections:
  - page title area
  - right-side action area with the create button
  - list area
  - empty state area
- Which parts come from Figma:
  - visual hierarchy
  - relative placement of the title and button
  - spacing and density in the list area
- Which parts should follow existing code or Mantine defaults:
  - use `Group` or `Flex` for the page header
  - use `Stack` for overall page layout
  - prefer Mantine's built-in layout and table patterns for the list container
  - if Figma does not specify a detail, follow existing list pages and Mantine defaults first

## 4. Page States

- Loading:
  - show a loading state for the list, either placeholders or a simple loading message
- Empty:
  - show empty-state copy and the create button
- Error:
  - show an error message and a retry action
- Success:
  - show the models list

## 5. API Plan

- SDK module:
  - `@matrixhub/api-ts/v1alpha1/model.pb`
- Read operations:
  - `Models.ListModels({ project, page, pageSize, search })`
  - `project` comes from the route param
  - `page`, `pageSize`, and `search` come from page state or search params
- First version needs:
  - the list response and pagination data only
  - no separate detail request if the page only shows the list
- First version does not need:
  - a new global `src/api` layer
  - shared query wrappers if only this page is using the call pattern
  - unrelated model write APIs
- Open questions:
  - whether search state should live in route search params or local page state

## 6. Locale Plan

- Namespace:
  - `projects.models`
- Expected keys:
  - `projects.models.title`
  - `projects.models.actions.create`
  - `projects.models.empty.title`
  - `projects.models.empty.description`
  - `projects.models.error.retry`

## 7. Open Questions

- Is a table better than cards for this list? Confirm against the final Figma
- Should the empty state include an illustration? If Figma does not provide one, stay consistent with the current page style

## 8. Implementation Order

1. Create the route adapter
2. Create `ProjectModelsPage`
3. Add locale files
4. Confirm the API contract
5. Revisit whether a dedicated list component is needed
