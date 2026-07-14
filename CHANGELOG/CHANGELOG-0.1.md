# MatrixHub v0.1.x release notes

## v0.1.1

MatrixHub **v0.1.1** is a patch release on the v0.1 line. It prepares a Compose-oriented runtime config next to `docker-compose.yml` so install docs can download both files from the same release tag.

### Downloads

#### Container image

```
ghcr.io/matrixhub-ai/matrixhub:v0.1.1
```

Verify signature (optional):

```bash
cosign verify ghcr.io/matrixhub-ai/matrixhub:v0.1.1 \
  --certificate-identity-regexp='.*' \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com
```

#### Helm chart (OCI)

```bash
export CHART_VERSION=0.1.1
export NAMESPACE=matrixhub

helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version "${CHART_VERSION}" \
  --namespace "${NAMESPACE}" \
  --create-namespace
```

### Changelog since v0.1.0

#### Feature

- Added `deploy/config.yaml` beside `deploy/docker-compose.yml` so Docker Compose setups can curl both artifacts from the same release tag and start with fewer manual config steps. (#814, @yitingdc)

### Upgrade notes

No breaking changes from **v0.1.0**. Existing deployments can keep their current config. New Compose installs may use the shipped `deploy/config.yaml` as the starting point (see upcoming README install steps).

---

## v0.1.0

MatrixHub **v0.1.0** is the first official release. It matches the code shipped in `v0.1.0-rc.1` and delivers the **V0 / M0–M1 baseline** described in the [roadmap](../docs/roadmap.md): a self-hosted, Hugging Face–compatible model registry for private inference clusters.

### Downloads

#### Container image

```
ghcr.io/matrixhub-ai/matrixhub:v0.1.0
```

Images are published for `linux/amd64` and `linux/arm64`, signed with [Cosign](https://docs.sigstore.dev/) (keyless), and ship with an SPDX SBOM attachment.

Verify signature (optional):

```bash
cosign verify ghcr.io/matrixhub-ai/matrixhub:v0.1.0 \
  --certificate-identity-regexp='.*' \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com
```

#### Helm chart (OCI)

```bash
export CHART_VERSION=0.1.0
export NAMESPACE=matrixhub

helm install matrixhub oci://ghcr.io/matrixhub-ai/matrixhub \
  --version "${CHART_VERSION}" \
  --namespace "${NAMESPACE}" \
  --create-namespace
```

See the [README](../README.md) and [documentation site](https://matrixhub.ai) for Docker Compose, storage, and upgrade guidance.

### Changelog since project inception

#### Feature

- Added a Hugging Face–compatible Git, Git LFS, and HTTP surface so clients such as **vLLM** and **SGLang** can use MatrixHub as a private model hub with `HF_ENDPOINT`.
- Added model repository lifecycle: create, browse, upload, download, and delete with project-scoped isolation.
- Added a Web UI for repository administration, user profile, access tokens, and platform settings.
- Added remote **registry** management with proxy-cache mode to localize public Hugging Face models on first request (pull-once, serve-all within the cluster).
- Added **sync policies** and an in-process job server to schedule registry replication tasks with resumable transfer foundations.
- Added token-based authentication, projects, roles, robots, and permission-aware project listing for multi-tenant operation.
- Added dataset repository support (initial).
- Added Docker Compose and **Helm** deployment paths with multi-arch container images published to `ghcr.io`.
- Added release automation: OCI Helm charts, Cosign signing, and SPDX SBOM generation for official tags.
- Added documentation site content at [matrixhub.ai](https://matrixhub.ai) and CNCF Slack community channel ([#matrixhub](https://cloud-native.slack.com/archives/C0A8UKWR8HG)).

#### Bug or Regression

- Fixed file size display in the UI to use SI units consistently.
- Fixed project permission resolution for list and detail APIs.
- Additional stability and UI fixes merged before `v0.1.0-rc.1`.

#### Documentation

- Published installation, operations, and development guides on the documentation site.
- Added project governance documents: Contributing, Security, Governance, and Maintainers.

### Known limitations

The following items are on the [roadmap](../docs/roadmap.md) but **not** part of v0.1.0:

- Full LDAP/OIDC/SSO integration
- Storage quotas and automated cleanup policies (design in progress)
- Model signing, malware scanning, and tag-locking promotion workflows
- S3-compatible storage as a first-class backend (PVC/local focus in this release)

Track follow-up work in [GitHub issues](https://github.com/matrixhub-ai/matrixhub/issues).

### Upgrade notes

There is no prior official release. Install from the container image or Helm chart above. To evaluate quickly, use the public demo at [demo.matrixhub.ai](https://demo.matrixhub.ai/) (credentials in the README).

---

<!-- v0.1.1 and later official releases: append new ## vX.Y.Z sections above this line. -->
