# Model artifact scanning and admission

This branch adds opt-in revision-bound static scanning, project download policies and a model Security page for the DaoCloud MatrixHub challenge in the 2026 Shanghai Open Source Competition.

- [Technical design and recorded verification](docs/design/artifact-security-admission.md)
- [Deployment and reproduction](deploy/security-prototype/README-stage2.md)
- [REST contract](api/openapi/artifact-security.json)
- [Third-party components and licenses](deploy/security-prototype/THIRD_PARTY.md)

From a Linux/WSL checkout with Docker Engine and Compose v2:

```bash
bash deploy/security-prototype/prepare-runtime.sh
docker compose -f deploy/security-prototype/compose.yaml up -d --build
docker compose -f deploy/security-prototype/compose.yaml ps
```

Wait for scanner and ClamAV health checks, then visit http://127.0.0.1:13872. The local prototype initializes `admin` / `changeme`; change the password after login. This Compose setup binds to loopback. Production deployment requires account initialization, HTTPS, backups and the agreed community integration scope.

The scanner does not load models or execute repository code. Strict policy denies incomplete checks and medium/high-risk findings. Current budgets include 1 GiB per file and a 128 MiB Git pack; the design documents supported formats, failure behavior and measured boundaries. Migration 2 is included; back up existing databases before upgrading and retain persistent volumes.
