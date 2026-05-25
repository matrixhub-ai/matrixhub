# Security Policy

## Supported versions

MatrixHub provides security fixes for maintained release branches. Versions that are end-of-life (EOL) do not receive security updates.

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Previous minor release | Best effort |
| Older releases | No |

Check [GitHub releases](https://github.com/matrixhub-ai/matrixhub/releases) for the current supported version.

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

## Response expectations

| Stage | Target |
|-------|--------|
| Initial acknowledgment | Within **3 business days** |
| Triage and severity assessment | Within **7 business days** |
| Fix or mitigation plan | Depends on severity; critical issues prioritized |

These are goals, not guarantees. Complex issues may take longer.

## Disclosure process

1. Reporter submits a private advisory or email.
2. Maintainers confirm the issue, assign severity, and develop a fix (often on a private branch).
3. A patched release is published; credit is given to the reporter if desired.
4. A public advisory or release note describes the issue after a fix is available.

We follow coordinated disclosure: please allow reasonable time for a fix before public disclosure.

## Security hardening (operators)

MatrixHub is often deployed on private networks. Operators should:

- Restrict network access to the API and admin UI
- Use strong authentication, TLS, and secrets management
- Keep dependencies and container images updated
- Follow your organization’s model-artifact and supply-chain policies

## Comments on this policy

Open a pull request against this file or discuss with maintainers on Slack (see [MAINTAINERS.md](MAINTAINERS.md)).
