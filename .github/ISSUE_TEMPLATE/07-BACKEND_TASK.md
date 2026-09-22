---
name: Backend Task
about: A backend task. Defaults to a subtask of a parent issue
title: "[Backend] "
labels: kind/feature

---
<!-- Defaults to a subtask of a parent (tracking) issue. To use as a standalone/normal issue, overwrite the feature section below with the actual details. -->

**What is the feature and Why is this needed**:

The feature is detailed in the parent issue.

**Dev requirements**:

- **Technical design**: for a simple change, describe the approach directly in the checklist; for a complex one, write a **technical design doc** (under [docs/design/](https://github.com/matrixhub-ai/matrixhub/blob/main/docs/design/README.md), or an external doc) and get maintainer review **before coding**; link it in the checklist.
- **Review the QA test cases** for this feature and leave **at least one constructive comment** on the QA task.
- Add **unit tests (UT)** for methods with complex or important logic, especially those under `internal/domain`.
- Add **E2E tests** for this feature.
- Document **API changes**: link the swagger contract (`api/openapiv2/v1alpha1/<service>.swagger.json`), then list each new/changed proto or HTTP API as its own sub-checkbox with a short note on what it does, and tick it once implemented so frontend and QA can see which APIs are ready; write **none** if not applicable.

---

**Completion checklist**:

- [ ] **Technical design**:
- [ ] **Test cases reviewed and commented**
- [ ] **UT**
- [ ] **E2E**
- [ ] **API change**: 
  - Swagger: `api/openapiv2/v1alpha1/<service>.swagger.json`
  - [ ] `<METHOD> /api/v1alpha1/...` / `rpc Xxx(...)` — what it does

The artifacts above should be described or linked in this checklist or the associated PR.

Before closing this issue, ensure all checklist items above are completed and checked off.
