# Security Policy

## Supported versions

MatrixHub provides security fixes for maintained release branches. Versions that are end-of-life (EOL) do not receive security updates.

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Previous minor release | Best effort |
| Older releases | No |

Check [GitHub releases](https://github.com/matrixhub-ai/matrixhub/releases) for the current supported version.

## Scope

**In scope** (please report):

- The MatrixHub source code in this repository
- Official container images (`ghcr.io/matrixhub-ai/matrixhub`) and the Helm chart
- Authentication, authorization, multi-tenancy, and data-isolation issues

**Out of scope** (please do not file as vulnerabilities):

- The public demo at [demo.matrixhub.ai](https://demo.matrixhub.ai/) and its
  documented default credentials (`admin` / `changeme`) — these are intentional for
  evaluation only.
- Publicly known third-party dependency vulnerabilities that, after triage, do
  not affect MatrixHub.
- Issues that require physical access, a compromised host, or misconfiguration
  contrary to the hardening guidance below.

If you are unsure whether something is in scope, report it — we would rather triage a
borderline report than miss a real issue.

## Security contacts

Security reports and triage are handled by the active maintainers listed in [MAINTAINERS.md](MAINTAINERS.md).

For vulnerability reports, use the private reporting channel described below (not public issues or maintainer DMs).

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

### Preferred: GitHub private security advisory

1. Go to [Security advisories](https://github.com/matrixhub-ai/matrixhub/security/advisories) for this repository.
2. Choose **Report a vulnerability** and submit details.

Maintainers will acknowledge receipt and work with you on coordinated disclosure.

We currently accept vulnerability reports **only via [GitHub Security Advisories](https://github.com/matrixhub-ai/matrixhub/security/advisories)**. A dedicated security email alias may be added later.

When submitting an advisory, include:

- Description of the issue and impact
- Steps to reproduce
- Affected versions or commits
- Any proof-of-concept or logs (avoid sharing secrets)

### Bug bounty

There is **no bug bounty program** at this time.

## Response expectations

| Stage | Target |
|-------|--------|
| Initial acknowledgment | Within **3 business days** |
| Triage and severity assessment | Within **7 business days** |
| Fix or mitigation plan | Depends on severity; critical issues prioritized |

These are goals, not guarantees. Complex issues may take longer.

## Disclosure process

1. Reporter submits a private [GitHub Security Advisory](https://github.com/matrixhub-ai/matrixhub/security/advisories).
2. Maintainers confirm the issue, assign severity, and develop a fix (often on a private branch).
3. A patched release is published; credit is given to the reporter if desired.
4. A public advisory is published via [GitHub Security Advisories (GHSA)](https://github.com/matrixhub-ai/matrixhub/security/advisories), and a **CVE ID is requested** where applicable. Release notes reference the fix.

We follow **coordinated disclosure**. We aim to disclose publicly once a fix is
available, and ask reporters to observe an embargo of up to **90 days** from the
initial report before public disclosure, to give users time to upgrade. We are happy
to coordinate timing with the reporter and other affected projects.

## Automated security scanning

MatrixHub runs the following checks in CI:

| Check | Scope | Workflow | Output |
|-------|-------|----------|--------|
| **govulncheck** | Go dependencies & stdlib | `govulncheck.yml` | PR check (blocks on known vulns) |
| **CodeQL** | Go + JS/TS static analysis | `codeql.yml` | GitHub Security tab (SARIF) |
| **Trivy** | Container image CVEs | `call-release-image.yaml` | GitHub Security tab (SARIF) |
| **Dependabot** | Go, npm, Actions, Docker | `dependabot.yml` | Automated PRs + security alerts |
| **Cosign + SBOM** | Release images | `call-release-image.yaml` | Signed images with SPDX SBOM |

All scanning workflows start **non-blocking** (report-only). After initial triage they are promoted to required checks on `main`.

## Dependency vulnerability triage

Maintainers review findings from dependency alerts, security scans, upstream
advisories, and private reports. They check affected versions, whether MatrixHub
uses the vulnerable functionality, and the impact on supported releases.
Applicable findings are prioritized for a dependency update, patch, or
mitigation, followed by validation and a patched release when warranted.
Findings assessed as not affecting MatrixHub are documented with the rationale.

## Security hardening (operators)

MatrixHub is often deployed on private networks. Operators should:

- Restrict network access to the API and admin UI
- Use strong authentication, TLS, and secrets management
- Keep dependencies and container images updated
- Follow your organization’s model-artifact and supply-chain policies

## Supply chain security

MatrixHub publishes signed artifacts to help you verify provenance:

- **Signed container images.** Release images are signed **keylessly** with
  [Cosign](https://docs.sigstore.dev/) using GitHub Actions OIDC (Fulcio/Rekor), so
  no long-lived signing keys are involved.
- **Software Bill of Materials (SBOM).** An SPDX SBOM is generated for each release
  image and attached alongside it.

Verify an image and download its SBOM:

```bash
# Verify the signature (keyless, GitHub Actions OIDC)
cosign verify \
  --certificate-identity-regexp 'https://github.com/matrixhub-ai/matrixhub/.*' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  ghcr.io/matrixhub-ai/matrixhub:<tag>

# Download the attached SBOM
cosign download sbom ghcr.io/matrixhub-ai/matrixhub:<tag>
```

## Comments on this policy

Open a pull request against this file or discuss with maintainers in the
[`#matrixhub`](https://cloud-native.slack.com/archives/C0A8UKWR8HG) channel on the
CNCF Slack.
