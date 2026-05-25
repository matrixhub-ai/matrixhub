# MatrixHub Governance

This document describes how the MatrixHub open source project is governed. It is intended to be lightweight and evolve with the community.

## Project scope

MatrixHub is an open-source, self-hosted model registry and distribution system for enterprise AI inference workloads. The project focuses on:

- Hugging Face–compatible access for inference clients (for example vLLM, SGLang)
- Private deployment, caching, governance, replication, and air-gapped workflows
- Cloud-native operation (Kubernetes, Helm, object storage)

Out of scope items are tracked in the project roadmap and product direction; large changes should be discussed in issues before substantial implementation.

## Roles

### Contributors

Anyone may contribute via issues, documentation, and pull requests under the [Contributing Guidelines](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md).

### Maintainers

Maintainers are listed in [MAINTAINERS.md](MAINTAINERS.md). They are responsible for:

- Reviewing and merging pull requests
- Cutting releases and publishing artifacts
- Triaging issues and security reports ([SECURITY.md](SECURITY.md))
- Upholding project quality, inclusivity, and technical direction

### Emeritus maintainers

Former maintainers who are no longer active may be listed as emeritus in [MAINTAINERS.md](MAINTAINERS.md). They are honored for past service and may be reinstated by maintainer consensus.

## Becoming a maintainer

There is no fixed quota. The bar is sustained, high-quality contribution and trust within the community.

Typical path:

1. Contribute regularly (code, reviews, docs, community support).
2. An existing maintainer proposes adding you in a pull request that updates [MAINTAINERS.md](MAINTAINERS.md).
3. At least **two-thirds of active maintainers** approve the PR (or no objection within **7 calendar days** on the PR).

Removal or emeritus status follows the same process when a maintainer is inactive or requests to step down.

## Decision making

- **Day-to-day:** Lazy consensus among maintainers on individual PRs and issues.
- **Significant changes** (architecture, breaking APIs, governance, license): Open an issue or discussion first; seek agreement among maintainers before merging.
- **Deadlock:** If maintainers cannot agree, the issue remains open until consensus or a majority vote among active maintainers (simple majority of those who vote within 14 days).

## Releases

- Releases are tagged from the default branch following project quality checks (CI, review).
- Release notes summarize user-visible changes and security fixes.
- Maintainers decide release timing and versioning (semver where applicable).

## Code of Conduct

All participants must follow the [MatrixHub Code of Conduct](CODE_OF_CONDUCT.md). Maintainers may warn, moderate, or ban contributors for violations, consistent with that policy and platform rules (GitHub, CNCF Slack).

## Security

Security reporting and response are defined in [SECURITY.md](SECURITY.md). Maintainers must treat security reports as confidential until disclosure.

## Intellectual property

- Contributions are accepted under the project’s [Apache License 2.0](LICENSE).
- Contributors should sign commits per project contribution requirements (for example DCO when enabled).
- Trademark and project asset policies may change if the project joins the CNCF; such changes will be announced to the community.

## Communication

- **Issues / PRs:** Primary development venue on GitHub.
- **Slack:** [CNCF Slack `#matrixhub`](https://cloud-native.slack.com/archives/C0A8UKWR8HG) for community discussion.
- **Public roadmap:** [docs/roadmap.md](docs/roadmap.md)

## Amending this document

Changes to `GOVERNANCE.md` require a pull request approved by at least **two-thirds of active maintainers**, with a **7-day** comment period for other maintainers and the community.

---

_This governance model is a starting template. Replace `TODO` entries in [MAINTAINERS.md](MAINTAINERS.md) before submitting a CNCF Sandbox application._
