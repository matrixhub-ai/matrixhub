# Contributing to MatrixHub

Welcome, and thank you for your interest in contributing to MatrixHub! 🎉

MatrixHub is a community-led open source project. Contributions of all kinds are
welcome — code, tests, documentation, bug reports, design feedback, reviews, and
helping other users. This guide explains how to get involved.

By participating, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md)
(the CNCF Code of Conduct).

## Ways to contribute

You don't have to write code to make a difference:

- **Report bugs** and **request features** using the [issue templates](https://github.com/matrixhub-ai/matrixhub/issues/new/choose).
- **Improve documentation** — the `docs/`, the `website/`, and inline docs.
- **Review pull requests** and help triage issues.
- **Answer questions** and help others in [Slack](https://cloud-native.slack.com/archives/C0A8UKWR8HG) and [GitHub Discussions](https://github.com/matrixhub-ai/matrixhub/discussions).
- **Write code** — bug fixes and features.

New here? Look for issues labeled
[`good first issue`](https://github.com/matrixhub-ai/matrixhub/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
and
[`help wanted`](https://github.com/matrixhub-ai/matrixhub/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22).

## Communication

- **GitHub Issues / Pull Requests** — primary development venue.
- **GitHub Discussions** — questions, ideas, and broader design conversations.
- **Slack** — [`#matrixhub`](https://cloud-native.slack.com/archives/C0A8UKWR8HG) in the CNCF Slack workspace.
- **Public roadmap** — [docs/roadmap.md](docs/roadmap.md).

## Reporting issues

- **Bugs & features:** open an issue via the [templates](https://github.com/matrixhub-ai/matrixhub/issues/new/choose).
  Before filing, please search existing [issues](https://github.com/matrixhub-ai/matrixhub/issues)
  and [pull requests](https://github.com/matrixhub-ai/matrixhub/pulls) to avoid duplicates.
- **Security vulnerabilities:** **do not** open a public issue. Follow the private
  reporting process in [SECURITY.md](SECURITY.md).

## Large features and task sub-issues

For a **large feature**, prefer opening a parent [Feature](https://github.com/matrixhub-ai/matrixhub/issues/new?template=01-FEATURE.md)
issue to track the overall goal, then split delivery into smaller task issues using
the task templates:

- [Backend Task](https://github.com/matrixhub-ai/matrixhub/issues/new?template=07-BACKEND_TASK.md)
- [UI Task](https://github.com/matrixhub-ai/matrixhub/issues/new?template=08-UI_TASK.md)
- [QA Task](https://github.com/matrixhub-ai/matrixhub/issues/new?template=09-QA_TASK.md)
- [Doc Task](https://github.com/matrixhub-ai/matrixhub/issues/new?template=10-DOC_TASK.md)

In each sub-issue, link the parent feature and fill that task template’s **Completion
checklist**. Track the sub-issues from the parent (for example list them under the
parent checklist or in the parent description).

**Do not close the parent feature issue until both are done:**

1. All related **sub-issues** are closed, and
2. The parent’s own **Completion checklist** is fully checked off

Closing a Backend / QA / Doc task while its **Completion checklist** still has
unchecked items may cause the issue to be reopened automatically.

## Helping triage issues

Issue triage helps the project keep work discoverable and actionable. Helpful
triage includes checking for duplicates, asking for missing reproduction steps or
logs, confirming whether a bug can be reproduced, and suggesting the most relevant
labels.

If you have permission to label issues, add labels that match the issue:

- **Type:** `kind/bug`, `kind/feature`, `kind/design`, `kind/dependency`,
  `kind/documentation`, `kind/support`, `kind/cleanup`, `kind/flake`,
  `kind/regression`, `kind/api-change`, `kind/failing-test`, or
  `kind/deprecation`.
- **Area:** `area/ui` or `area/test`.
- **Triage state:** `triage/needs-information`, `triage/not-reproducible`, or
  `triage/duplicate`.
- **Priority:** `priority/backlog`, `priority/important-soon`,
  `priority/important-longterm`, or `priority/critical-urgent`.

Use `good first issue` or `help wanted` only when the issue is clear enough for a
new contributor or when maintainer help is welcome.

If you do not have permission to label issues, leave a comment with your triage
notes and suggested labels. Maintainers can apply labels or ask follow-up questions from there.

Once an issue is confirmed for a release, maintainers add it to the corresponding
release milestone (for example `v0.2`).

## Claiming issues

If you want to work on an open issue, leave a comment saying that you would like
to take it. If you have permission to assign issues, you may assign yourself;
otherwise, a maintainer can assign it to you.

If an issue is already assigned, please coordinate with the assignee before
opening a competing PR. If the assignee appears inactive, comment on the issue and
ask maintainers whether it can be reassigned.

If you think someone else is the right person to fix an issue, mention them in a
comment with context. Maintainers may assign the issue once the person agrees or when there is a clear owner.

## Contribution workflow

1. **Align on scope.** Open an issue using the issue templates (see
   [Reporting issues](#reporting-issues)), or pick up an existing issue (see
   [Claiming issues](#claiming-issues) and
   [Helping triage issues](#helping-triage-issues)). For a large feature, split work
   into task sub-issues (see
   [Large features and task sub-issues](#large-features-and-task-sub-issues)).
2. **Read the issue checklist.** Each issue template may define a different
   completion or verification checklist. Make sure you understand what must be
   completed before the issue can be closed. For a parent feature, that includes
   finishing related sub-issues **and** the parent checklist.
3. **Fork** the repository and create a topic branch, following the standard
   [GitHub pull request](https://help.github.com/articles/about-pull-requests/)
   process.
4. **Agree on the approach before coding.** For large or complex work, discuss
   the approach in the issue or a GitHub Discussion first. For complex backend
   changes, write a short design doc under `docs/design/` and get maintainer
   review before coding.
5. **Set up local development when needed.** If the contribution changes code,
   use the [Developer Guide](docs/development.md) to set up the local API, UI,
   database, generated code, and tests.
6. **Make your change**, following the backend architecture rules in
   [docs/architecture.md](docs/architecture.md) and the frontend rules in
   [ui/AGENTS.md](ui/AGENTS.md).
7. **Add tests** for behavior changes and bug fixes, then run the relevant tests
   locally (see [Testing](#testing)).
8. **Update user-facing documentation when needed.** If the change affects how
   users install, configure, or use MatrixHub, update the documentation website.
   See the [Developer Guide](docs/development.md) for documentation website local
   development.
9. **Sign your commits** (see
   [Developer Certificate of Origin](#developer-certificate-of-origin-dco)).
10. **Open a pull request** using the PR template. Link the issue: use
    `Closes #123` when the PR completes all checklist items for the issue, or
    `Refs #123` when it only completes part of the work. Fill in the PR checklist
    (see [Pull request](#pull-request)).
11. **Make sure CI passes.** If static-check CI fails, see
    [Coding standards](#coding-standards) for the local commands.
12. **Request review.** Add reviewers or mention the right people in a comment.
    Reviewers may add `/lgtm`; approvers may add `/approve`. A PR needs at least
    one `/lgtm`, one `/approve`, and passing CI before it can be merged. Use
    [`OWNERS`](OWNERS) to find reviewers and approvers, or ask in Slack (see
    [Review process](#review-process)).
13. **Finish the issue after merge.** After the PR is merged, update the linked
    issue checklist and add relevant PR or documentation links. Close a
    **task** issue only when its checklist is done. Close a **parent feature**
    only when all sub-issues are closed **and** the parent checklist is done.
    If anything remains, leave a comment and hand off the remaining work to the
    right owner.

## Local development

For prerequisites, local API/UI setup, MySQL configuration, generated code,
testing commands, and troubleshooting, see [docs/development.md](docs/development.md).

For a quick non-development run via Docker Compose, see the **Quick Start** in
the [README](README.md).

## Coding standards

- **License headers:** every Go source file must start with the Apache 2.0 license
  header (enforced by the `goheader` linter).
- **Static checks:** run the local CI-required static checks before pushing:

  ```bash
  make verify               # locally runnable CI static checks
  make verify.go            # Go lint and generated-code checks only
  make verify.ui            # UI lint, typecheck, and build only
  make lint-fix             # golangci-lint with --fix
  ```

- **Generated code is not hand-edited.** Regenerate it after changing the source of
  truth:

  ```bash
  make genproto             # after editing api/proto/v1alpha1/*.proto
  make gen_openapi_sdk      # after swagger changes (test HTTP SDK)
  make generate-mocks       # after changing a mocked interface
  ```

- Follow the conventions and dependency direction described in
  [docs/architecture.md](docs/architecture.md).

## Developer Certificate of Origin (DCO)

All commits must be **signed off** to certify that you wrote the patch or otherwise
have the right to submit it under the project's open source license, per the
[Developer Certificate of Origin](https://developercertificate.org/).

Add a `Signed-off-by` line by committing with `-s`:

```bash
git commit -s -m "Your commit message"
```

This adds a line like:

```
Signed-off-by: Your Name <your.email@example.com>
```

The name and email must match your Git author identity. If you forget, you can
amend the last commit with `git commit --amend -s`, or sign off a range of commits
with `git rebase --signoff`.

## Testing

Before opening a PR, run the tests relevant to your change:

- Run unit tests with `make test.unit`.
- Run end-to-end tests with `make test.e2e` when changing API behavior, backend
  workflows, jobs, or integration paths.
- Add or update tests for behavior changes and bug fixes.
- If a test cannot be run locally, mention it in the PR.

See the [Developer Guide](docs/development.md#unit-tests) for detailed test
commands, local setup, coverage, and E2E notes.

## Pull request

- Keep PRs focused and reasonably small; link the issue they address.
- Ensure **CI is green** (lint, unit tests, and other checks) and that your commits
  are **signed off**.
- Fill in the PR template `release-note` block for user-visible changes (see
  [Release notes in pull requests](#release-notes-in-pull-requests)).

## Review process

- Reviews follow the [`OWNERS`](OWNERS) model: maintainers use `/lgtm` and `/approve`
  on PRs. A PR is merged once it has the required approvals and CI passes.
- Be responsive to review feedback; maintainers aim to review promptly but this is a
  community project, so please be patient.

See [GOVERNANCE.md](GOVERNANCE.md) for roles, decision making, and how to become a
maintainer.

## Release notes in pull requests

MatrixHub follows the [Kubernetes release notes model](https://github.com/kubernetes/community/blob/main/contributors/guide/release-notes.md):

- **Official releases only** (`vX.Y.Z`) publish release notes on the
  [GitHub Releases](https://github.com/matrixhub-ai/matrixhub/releases) page and in
  [`CHANGELOG/`](CHANGELOG/README.md). RC and dev tags do **not** publish release notes.
- **Every pull request** with a user-visible change must include a `release-note`
  block in the PR description (or `NONE` if not user-facing). The PR template already
  includes this section.
- Reviewers should check release note quality during review (clear, past tense,
  user-focused).

Example:

```release-note
Added permission-based filtering to the project list API. (#664, @contributor)
```

Maintainers aggregate these notes into `CHANGELOG/` when cutting an official release.
See [Release process](docs/release-process.md) for maintainer steps.

## License of contributions

MatrixHub is licensed under the [Apache License 2.0](LICENSE). By contributing, you
agree that your contributions will be licensed under the same license (inbound =
outbound), and you certify your right to do so via the DCO sign-off described above.

---

Thank you for helping make MatrixHub better! If anything here is unclear, open an
issue or ask in [Slack](https://cloud-native.slack.com/archives/C0A8UKWR8HG).
