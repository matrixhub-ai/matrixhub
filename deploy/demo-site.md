# Demo Site Operations

This document describes how maintainers deploy and operate the public MatrixHub
demo at <https://demo.matrixhub.ai/>.

## Ownership and Access

The demo is maintained by repository maintainers who have access to the GitHub
`demo` environment. The environment must provide these secrets:

- `DEPLOY_HOST`: the demo server address.
- `DEPLOY_SSH_KEY`: the SSH key used by the deployment workflow.

The workflow connects to the server as `root` and manages the deployment under
`/root/matrixhub/deploy`.

## Deployment

The [Deploy Demo Site](../.github/workflows/deploy-demosite.yml) workflow is
started manually with an image tag.

1. Open **Actions** -> **Deploy Demo Site** -> **Run workflow**.
2. Enter the image tag to deploy, or keep the default `latest` tag for the
   current demo build. The workflow accepts normal Docker tags, including
   `latest`.
3. Run the workflow and wait for all steps to complete.
4. Open <https://demo.matrixhub.ai/> and verify that the login page loads and the
   documented demo account can sign in.

During deployment, the workflow:

- copies the explicit MySQL Compose file, demo-only Nginx overlay, and
  MySQL-backed MatrixHub configuration to the demo server;
- creates an `.env` file with the selected image tag on the first deployment
  and random MySQL credentials for a new database, while preserving existing
  MySQL credentials on later deployments;
- pulls the selected MatrixHub image and starts the Compose services; and
- installs or refreshes the scheduled data-reset task.

## Data Reset

The demo is disposable. The deployment workflow installs
`/root/matrixhub/reset.sh` and runs it every six hours with cron:

```text
0 */6 * * *
```

The reset stops the Compose services, removes the MySQL database and MatrixHub
data directories, and starts the services again. It deletes all users,
projects, models, and other data created through the demo. The `.env` file and
the Nginx ACME certificate volume are preserved, so the recreated MySQL instance
continues using the same credentials.

Reset output is written to `/root/matrixhub/logs/reset.log`. A maintainer can run
`/root/matrixhub/reset.sh` manually when an immediate reset is required.

## Routine Maintenance

After a deployment or reset:

1. Confirm that <https://demo.matrixhub.ai/> responds over HTTPS.
2. Confirm that the documented demo account can sign in.
3. Check that the `mysql`, `matrixhub`, and `nginx` Compose services are running.
4. If a scheduled reset fails, inspect `/root/matrixhub/logs/reset.log` and the
   Compose service logs.

Deployment configuration should be changed in this repository and applied
through the workflow. Avoid relying on manual edits on the demo server because
the next deployment can overwrite them.

## Rollback

To roll back the application, rerun **Deploy Demo Site** with the last known-good
version tag. The workflow updates only the MatrixHub image tag and does not
reset demo data as part of deployment.

If the older image cannot use the current disposable demo database after a
schema migration, run `/root/matrixhub/reset.sh` after the rollback. This removes
all demo data and recreates the database using the selected image.
