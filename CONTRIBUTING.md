# Contributing to MatrixHub

Welcome, and thank you for your interest in contributing to MatrixHub! 🎉

MatrixHub is a community-led open source project. Contributions of all kinds are
welcome — code, tests, documentation, bug reports, design feedback, reviews, and
helping other users. This guide explains how to get involved.

By participating, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md)
(the CNCF Code of Conduct).

## Ways to contribute

You don't have to write code to make a difference:

- **Report bugs** and **request features**, see [Reporting issues](#reporting-issues).
- **Improve documentation** — the `docs/`, the `website/`, and inline docs.
- [**Help triage issues**](#helping-triage-issues)
- [**Review pull requests**](#review-process)
- **Answer questions** and help others in [Slack](https://cloud-native.slack.com/archives/C0A8UKWR8HG) and [GitHub Discussions](https://github.com/matrixhub-ai/matrixhub/discussions).
- **Write code** — bug fixes and features，see [Code contribution workflow](#code-contribution-workflow)

New here? Start from looking for issues labeled
[`good first issue`](https://github.com/matrixhub-ai/matrixhub/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
and
[`help wanted`](https://github.com/matrixhub-ai/matrixhub/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22).

## Code contribution workflow

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
4. **Agree on the technical proposal before coding.** For large or complex work, discuss
   the proposal in the issue or a GitHub Discussion first. For complex backend
   changes, write a short design doc under [docs/design/](docs/design/README.md)
   and get maintainer review before coding.
5. **Set up local development when needed.** If the contribution changes code,
   use the [Developer Guide](docs/development.md) to set up the local API, UI,
   database, generated code, and tests.
6. **Make your change**, following the backend architecture rules in
   [docs/code-architecture.md](docs/code-architecture.md) and the frontend rules in
   [ui/AGENTS.md](ui/AGENTS.md).
7. **Add tests** for behavior changes and bug fixes, then run the relevant tests
   locally (see [Testing](#testing)).
8. **Update user-facing documentation when needed.** If the change affects how
   users install, configure, or use MatrixHub, update the documentation website.
   See the [Website Developer Guide](docs/development.md#documentation-website-local-development) for documentation website local
   development.
9. **Sign your commits** (see
   [Developer Certificate of Origin](#developer-certificate-of-origin-dco)).
10. **Open a pull request.** Follow the [Pull request](#pull-request) guidance and
    [Release notes in pull requests](#release-notes-in-pull-requests).
11. **Make sure CI passes.** If static-check CI fails, see
    [Coding standards](#coding-standards) for the local commands.
12. **Request review.** Add reviewers or mention the right people in a comment.
    Use [`OWNERS`](OWNERS) to find reviewers and approvers, or ask in Slack.
    See [Review process](#review-process) for approval requirements.
13. **Finish the issue after merge.** After the PR is merged, update the linked
    issue checklist and add relevant PR or documentation links. Close a
    **task** issue only when its checklist is done. Close a **parent feature**
    only when all sub-issues are closed **and** the parent checklist is done.
    If anything remains, leave a comment and hand off the remaining work to the
    right owner.

## Reporting issues

- **Bugs & features:** open an issue via the [templates](https://github.com/matrixhub-ai/matrixhub/issues/new/choose).
  Before filing, please search existing [issues](https://github.com/matrixhub-ai/matrixhub/issues)
  and [pull requests](https://github.com/matrixhub-ai/matrixhub/pulls) to avoid duplicates.
- **Security vulnerabilities:** **do not** open a public issue. Follow the private
  reporting process in [SECURITY.md](SECURITY.md).

## Helping triage issues

Issue triage helps the project keep work discoverable and actionable. Helpful
triage includes checking for duplicates, asking for missing reproduction steps or
logs, confirming whether a bug can be reproduced, and suggesting the most relevant
labels.

Anyone may add or remove existing public `kind/*`, `area/*`, `priority/*`, and
`triage/*` labels with commands such as `/kind bug` and `/remove-kind bug`:

- **Type:** `kind/bug`, `kind/feature`, `kind/design`, `kind/dependency`,
  `kind/documentation`, `kind/support`, `kind/cleanup`, `kind/flake`,
  `kind/regression`, `kind/api-change`, `kind/failing-test`, or
  `kind/deprecation`.
- **Area:** `area/ui` or `area/test`.
- **Triage state:** `triage/needs-information`, `triage/not-reproducible`, or
  `triage/duplicate`.
- **Priority:** `priority/backlog`, `priority/important-soon`,
  `priority/important-longterm`, or `priority/critical-urgent`.

On issues, anyone may also use `/good-first-issue`, `/help-wanted`, and their
`/remove-*` forms. Reserve them for clear starter work or issues seeking help.

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
  [docs/code-architecture.md](docs/code-architecture.md).

## Testing

Before opening a PR, run the tests relevant to your change:

- Run [unit tests](docs/development.md#unit-tests) with `make test.unit`.
- Run [end-to-end tests](docs/development.md#end-to-end-tests) with `make test.e2e`
  when changing API behavior, backend workflows, jobs, or integration paths.
- Add or update tests for behavior changes and bug fixes.
- If a test cannot be run locally, mention it in the PR.

See the [Developer Guide](docs/development.md#unit-tests) for detailed test
commands, local setup, coverage, and E2E notes.

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

## Pull request

- Keep PRs focused and reasonably small. Use the PR template and fill in its checklist.
- Link the issue: use `Closes #123` when the PR completes all checklist items for
  the issue, or `Refs #123` when it only completes part of the work.
- Ensure **CI is green** (lint, unit tests, and other checks).
- Choose a `/kind` and fill in the PR template `release-note` block (see
  [Release notes in pull requests](#release-notes-in-pull-requests)).

## Release notes in pull requests

MatrixHub follows the [Kubernetes release notes model](https://github.com/kubernetes/community/blob/main/contributors/guide/release-notes.md).
Every pull request must choose a `/kind` and complete the `release-note` block with
a user-, API-, or operator-facing change, or `NONE`.

Anyone may correct Kind with `/kind` or `/remove-kind`. The bot derives release-note
labels from the PR body, and reviewers verify the note's accuracy and wording.

Example:

```release-note
Added permission-based filtering to the project list API.
```

The collector adds PR and author links. Maintainers should follow
[Prepare release notes](docs/release-process.md#prepare-release-notes).

## Review process

- Reviews follow the [`OWNERS`](OWNERS) model: reviewers may add `/lgtm`, and
  approvers may add `/approve`. A PR needs at least one of each and passing CI
  before it can be merged.
- Anyone can comment on PRs and help review them. Review changes carefully and
  offer constructive feedback; if you have `/lgtm` permission, use it only when
  you can responsibly endorse the PR.
- Anyone may use `/hold` to pause a PR and `/hold cancel` or `/unhold` to resume it.
- Be responsive to review feedback; maintainers aim to review promptly but this is a
  community project, so please be patient.

See [GOVERNANCE.md](GOVERNANCE.md) for roles, decision making, and how to become a
maintainer.

## License of contributions

MatrixHub is licensed under the [Apache License 2.0](LICENSE). By contributing, you
agree that your contributions will be licensed under the same license (inbound =
outbound).

---

Thank you for helping make MatrixHub better! If anything here is unclear, open an
issue or ask in [Slack](https://cloud-native.slack.com/archives/C0A8UKWR8HG).
