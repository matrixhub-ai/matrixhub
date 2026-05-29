# MatrixHub Internal Auto-Deploy

When a PR merges to `main`, GitHub Actions builds the image, then the `deploy-internal` job runs on a self-hosted runner inside the internal K8s cluster to execute `helm upgrade`.

## Architecture

```
PR merge → GitHub Actions → push image + chart (cloud) → deploy-internal (self-hosted runner in K8s) → helm upgrade
```

The runner pod makes an **outbound** HTTPS connection to GitHub (long-poll for jobs). No inbound ports required.

The runner is **org-level** by default (registered against `matrixhub-ai`), so any repo in the org's runner group can use it.

## Auth modes

Pick **one** when bootstrapping (see comments in `runner-deployment.yaml`):

| Mode | Pros | Cons | When to use |
|------|------|------|-------------|
| **A — Registration token + PVC** | Owner only needs to issue a 1-hour token; no long-lived secret in K8s | Loses identity if PVC is lost — needs a fresh token from the owner | Standard. Recommended starting point. |
| **B — PAT (admin:org)** | Self-healing across PVC loss; no owner involvement after setup | Long-lived secret stored in K8s; need a bot account with org admin | High-availability or no-bus-factor setups. |

The runner's long-term identity is stored in `/runner/.runner` + `.credentials` (lifetime ≈ 6 months, auto-renewed). The PVC `github-runner-data` persists these across pod restarts.

## Setup — Mode A (registration token + PVC)

### 1. Org owner creates a runner group (one-time)

```
https://github.com/organizations/matrixhub-ai/settings/actions/runner-groups
```

- New group → `internal-deploy`
- Repository access → Selected repositories → `matrixhub`

### 2. Org owner generates a registration token

UI: `Settings → Actions → Runners → New self-hosted runner` shows a token.

Or via API (anyone with `admin:org` scope on a PAT):

```bash
curl -X POST -H "Authorization: Bearer <admin-pat>" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/orgs/matrixhub-ai/actions/runners/registration-token
```

The returned `token` is valid for 1 hour.

### 3. Apply the manifests + secret

```bash
kubectl apply -f deploy/autoupdate/runner-deployment.yaml

kubectl -n matrixhub-runner create secret generic github-runner-secret \
  --from-literal=runner-token=AAAA_THE_1H_TOKEN
```

### 4. (Optional) Wipe the secret after registration

The credentials are now in the PVC. The registration token has been consumed and is useless:

```bash
kubectl -n matrixhub-runner delete secret github-runner-secret
```

## Setup — Mode B (PAT)

In `runner-deployment.yaml`, swap the env block:

```yaml
# comment out RUNNER_TOKEN, enable ACCESS_TOKEN
- name: ACCESS_TOKEN
  valueFrom:
    secretKeyRef:
      name: github-runner-secret
      key: github-pat
```

Then:

```bash
kubectl -n matrixhub-runner create secret generic github-runner-secret \
  --from-literal=github-pat=ghp_xxx
kubectl apply -f deploy/autoupdate/runner-deployment.yaml
```

## Verify

```bash
kubectl -n matrixhub-runner get pods
kubectl -n matrixhub-runner logs -l app=github-runner --tail=20
```

Expect:
```
√ Connected to GitHub
Listening for Jobs
```

And in GitHub:
```
https://github.com/organizations/matrixhub-ai/settings/actions/runners
```
The runner should appear as **Idle**.

## Test end-to-end

Merge any PR to `main`. The `deploy-internal` job should be dispatched to this runner and complete `helm upgrade` against the internal cluster.

## Customization

| Env var on runner Pod | Default | Effect |
|-----------------------|---------|--------|
| `CHART_REGISTRY` | `oci://ghcr.io/matrixhub-ai` | Where to pull the chart from |
| `DEPLOY_NAMESPACE` | `matrixhub` | Target namespace for `helm upgrade` |
| `VALUES_FILE` | `/etc/matrixhub/values-internal.yaml` | Helm values file (from ConfigMap) |

Per-environment values live in the `github-runner-values` ConfigMap (also editable post-deploy without changing the workflow).

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Pod CrashLoop, "401 Unauthorized" | Owner deleted the runner from GitHub UI, or PVC was reset | Wipe PVC + new registration token + re-bootstrap |
| Pod CrashLoop, "Token is expired" (mode A, first start) | Took > 1 hour to apply | Get a fresh token, recreate secret, restart pod |
| `helm upgrade` fails RBAC | Cluster RBAC blocks ClusterRole binding | Adjust the ClusterRole in `runner-deployment.yaml` |
| `helm pull` 403 | Auth flow broken | Check `helm registry login` step in workflow has access to a working GITHUB_TOKEN |
| Runner appears offline | Pod evicted / node down | `kubectl describe pod` and check events |

## Files

- `runner-deployment.yaml` — Namespace + SA + ClusterRoleBinding + PVC + ConfigMap + Deployment
- `values-internal.yaml` — Source-of-truth for internal Helm values (also embedded in the ConfigMap above)
