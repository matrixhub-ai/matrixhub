---
slug: /enterprise-private-huggingface
title: Why Financial Institutions and Enterprises Need a Private Hugging Face
description: Learn how teams can keep using models without code changes in isolated networks while keeping model assets controlled and access clearly bounded.
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

When deploying large models in highly regulated industries such as finance, government, and energy, teams usually face two challenges:

1. **Getting models into the private network:** Public model hubs are inaccessible, and manually copying hundreds of gigabytes of weights can take days.
2. **Managing model permissions:** Models are critical assets, so there must be clear boundaries around who can download, modify, and manage them.

Both matter. A **self-hosted, Hugging Face-compatible private model hub** solves them together: it centralizes model storage and access control while allowing development teams to use familiar tools and workflows.

{/* truncate */}

<div className="enterprise-blog">

## The big picture: bring models in, store them centrally, and share them internally

```mermaid
flowchart LR
    SOURCE["Model sources<br/>Public hubs · other MatrixHub instances · local files"]
    HUB["MatrixHub<br/>Private model hub"]
    USERS["Development teams / inference nodes<br/>HF_ENDPOINT points internally"]

    SOURCE -->|"Proxy pull · remote sync · one-time transfer"| HUB
    HUB -->|"Cache + distribution"| USERS
```

Models are stored centrally in the private model hub. Everyone retrieves them internally without depending on the public internet. Developers only need to point `HF_ENDPOINT` at the internal address—**no code changes required**.

There are three ways to bring models into the private network, depending on network connectivity:

- **Proxy pull:** In an internet-connected environment, the first request pulls and caches a model from a public source. All later requests use the internal cache.
- **Remote sync:** Automatically synchronize with MatrixHub in another environment to keep models consistent.
- **One-time transfer:** In a fully air-gapped environment, obtain the model files elsewhere and transfer them in as a complete set.

:::tip[Further reading]

For detailed proxy-pull and cache-distribution procedures, see:

- [Distributing DeepSeek v4](./2026-04-27-deepseek-v4-distribution.md)
- [Distributing Models to an Internal vLLM Cluster](./2026-04-27-examples.md)

:::

## Access isolation: define who can work with each model

Models are critical assets. MatrixHub protects them through three layers:

```mermaid
flowchart LR
    PROJECT["Layer 1<br/>Project space<br/>Defines who can see assets"]
    ROLE["Layer 2<br/>Member role<br/>Defines what people can do"]
    TOKEN["Layer 3<br/>Access credentials<br/>Robot accounts and personal access tokens"]

    PROJECT --> ROLE --> TOKEN
```

### Layer 1: project spaces—an independent space for each team

Each team or business unit has its own project. For example, development, staging, and production can use `dev`, `stage`, and `prod`. Projects are hidden from one another by default, providing natural isolation. The following three projects demonstrate this behavior.

#### 1. Administrator view: all projects coexist

After signing in, an administrator can see every project on the platform:

![Project list](./images/01-project-list.png)

#### 2. Each project has its own members

Members belong to their respective projects and do not overlap:

<Tabs groupId="project-members">
  <TabItem value="dev" label="dev" default>

  ![Members of the dev project](./images/02-dev-members.png)

  </TabItem>
  <TabItem value="stage" label="stage">

  ![Members of the stage project](./images/03-stage-members.png)

  </TabItem>
  <TabItem value="prod" label="prod">

  ![Members of the prod project](./images/04-prod-members.png)

  </TabItem>
</Tabs>

#### 3. Members see only their own projects

When members of the three projects sign in, each person can see only the project they belong to:

<Tabs groupId="member-login">
  <TabItem value="dev" label="dev" default>

  ![Dev member signed in](./images/05-dev-login.png)

  </TabItem>
  <TabItem value="stage" label="stage">

  ![Stage member signed in](./images/06-stage-login.png)

  </TabItem>
  <TabItem value="prod" label="prod">

  ![Prod member signed in](./images/07-prod-login.png)

  </TabItem>
</Tabs>

Everyone uses the same sign-in page, but members of different teams see different project sets. This is the clearest expression of an independent project space.

### Layer 2: member roles—control what people can do within a project

| Role | Capabilities | Typical use case |
| --- | --- | --- |
| Administrator | Manage all project settings and members | Platform operator responsible for creating projects and assigning members |
| Editor | Upload and modify models | ML engineer who trains or fine-tunes models and publishes new versions |
| Viewer | View and download, but not upload or modify | Colleague who needs model access, or a read-only model-pull pipeline |

To let someone view and download models without uploading or changing them, assign the Viewer role.

The project member list shows each member's account and role:

![Project member list](./images/08-member-role-list.png)

These permissions are enforced. If a Viewer attempts to upload a model, the request is rejected:

![Model upload denied for a Viewer](./images/09-viewer-upload-denied.png)

### Layer 3: access credentials—robot accounts and personal access tokens

People use signed-in accounts in the browser. Model downloads and uploads, along with CLI/API operations for CI/CD and inference services, require access credentials. MatrixHub provides two approaches:

| Approach | Identity represented | Best suited for | Permissions and lifecycle |
| --- | --- | --- | --- |
| Robot account | An independent machine identity | CI/CD, inference services, and shared pipelines | Platform administrators set its project scope, permissions, and expiration independently; its token can be disabled or refreshed separately |
| Personal access token | The signed-in user | Personal development, debugging, and local `hf` commands | Inherits the user's existing project roles and permissions; the user can create, expire, or delete it |

Both approaches use a token for CLI/API authentication, but the token represents a different identity and therefore serves a different operational purpose.

#### Approach 1: robot accounts—an independent identity for programs

Platform administrators can create robot accounts for CI/CD, inference services, and other programs, then explicitly limit their projects and permissions:

![Create a robot account](./images/10-robot-create.png)

After creation, robot accounts can be reviewed, disabled, deleted, or have their tokens refreshed centrally. Their lifecycle does not depend on an employee's personal account:

![Robot account list](./images/11-robot-list.png)

#### Approach 2: personal access tokens—use an individual's existing permissions

A user can also create a personal access token for local `hf` commands, personal scripts, or debugging. The token inherits the user's existing project roles and permissions; it grants no additional access. The full value is displayed only once when created and should be saved immediately and stored securely:

![Personal access token](./images/12-access-token.png)

## Adding models: manage in-house models centrally

After fine-tuning, bring the local model under centralized MatrixHub management in four steps.

#### 1. Create the target model repository

Sign in to the console and create a model repository under the target project. The account performing the upload needs the project Administrator or Editor role.

#### 2. Check the local model files

Before uploading, verify that the directory contains the weights, configuration, tokenizer, and other runtime files. This prevents missing dependencies from being discovered only after the model is stored:

![Local model files ready for upload](./images/13-local-model-files.png)

#### 3. Configure the MatrixHub endpoint and upload

In a local terminal, point `HF_ENDPOINT` to the internal MatrixHub instance, then use `hf upload` to upload the complete model directory:

```bash
export HF_ENDPOINT=http://<internal-matrixhub-address>
hf upload <project-name>/<model-name> ./<local-model-directory> .
```

When the terminal shows file hashing, LFS transfer progress, and the final repository URL, the client-side upload has completed:

![Run hf upload in the terminal](./images/14-hf-upload.png)

#### 4. Verify the model files and version history

Return to the console and verify the result from both the model details and commit history views:

<Tabs groupId="model-verification">
  <TabItem value="files" label="Model details" default>

  Use the model details page to confirm that weights, configuration, and other files are stored completely.

  ![Model details](./images/15-model-detail.png)

  </TabItem>
  <TabItem value="commits" label="Commit history">

  Every upload or modification creates a commit, making model versions and changes traceable.

  ![Model commit history](./images/16-model-commits.png)

  </TabItem>
</Tabs>

Once stored, models no longer remain scattered across individual machines. Versions, files, and change history are managed centrally, while teams continue using familiar tools.

## Distributing models in isolated and controlled environments

There are two approaches, depending on whether the target environment can connect to the model source.

### Scenario 1: fully air-gapped environment—one-time transfer

```mermaid
flowchart LR
    CONNECTED["Connected environment<br/>Download directly or cache with MatrixHub"]
    TRANSFER["One-time transfer"]
    AIRGAP["Isolated network<br/>Import into MatrixHub for team access"]

    CONNECTED --> TRANSFER --> AIRGAP
```

- **Connected environment:** Download the model files directly. If you cache them with MatrixHub before transferring them, their versions and file structure move together, avoiding a separate re-upload step.
- **Isolated network:** Import the model into the deployed MatrixHub instance, after which the team can retrieve it normally.

:::info[Operation note]

The download and import procedures follow the process described in “Adding models” above: use `hf download` to retrieve the model and `hf upload` to import it.

:::

### Scenario 2: connected environments—automated remote sync

When a staging environment can connect to the private network—or two data centers can reach each other—**remote sync** can automatically copy models between them.

```mermaid
flowchart LR
    CONNECTED_HUB["Connected environment<br/>MatrixHub"]
    TARGET_HUB["Private network / target environment<br/>MatrixHub"]

    CONNECTED_HUB <-->|"Pull / push · optionally scheduled"| TARGET_HUB
```

Configuration steps:

1. In the console, create a target registry and configure MatrixHub in the connected environment as the remote registry:

![Connected MatrixHub environment](./images/17-connected-matrixhub.png)

![Target registry configuration](./images/18-target-registry.png)

2. Create a sync rule and specify the target registry, resources, and trigger:

![Sync rule](./images/19-sync-rule.png)

3. After triggering synchronization, view progress and logs on the task page:

![Sync task](./images/20-sync-task.png)

4. When synchronization finishes, inspect the synchronized model files in the target project:

![Synchronized model files](./images/21-synced-model-files.png)

#### Why remote sync matters

Remote sync does more than copy files once. It turns model movement across environments into a **repeatable, controlled, and observable** process.

| Core capability | Problem addressed | Operational value |
| --- | --- | --- |
| Bidirectional movement | Some environments need to pull models in, while others need to push models out | One mechanism covers both model intake and distribution across staging zones, private networks, and multiple data centers |
| Automated execution | Manual copying must be repeated and can easily miss new versions | Scheduled runs and manual triggers reduce repetitive operations and help target environments receive the required versions sooner |
| Controlled transfer | Large model transfers can saturate a link, while name conflicts can overwrite existing assets unexpectedly | Bandwidth limits and overwrite policies make transfers more predictable and reduce their impact on production networks and existing models |
| End-to-end visibility | Manual transfers are opaque, making it difficult to identify which model failed and why | Every sync creates a task and splits execution by model; visible progress and logs make troubleshooting and result review easier |

For example, after caching a new model in an internet-connected staging environment, MatrixHub can synchronize it to the private network on a schedule. The same policy model can keep required model versions aligned across multiple data centers. Once platform administrators configure the policies centrally, model movement no longer depends on temporary scripts or constant manual supervision.

## Conclusion: bring models—and their governance—inside

An enterprise “private Hugging Face” is more than a place to store model files. It is internal model infrastructure that connects **access, governance, and distribution**:

- **Keep familiar workflows:** Development teams continue using Hugging Face tools and point `HF_ENDPOINT` to the internal MatrixHub instance without changing application code.
- **Manage model assets centrally:** In-house and external models, along with their version history, remain on one platform instead of being scattered across personal machines and temporary storage.
- **Define clear permission boundaries:** Project spaces and member roles determine who can see and modify assets, while robot accounts and personal access tokens represent machine and human identities separately.
- **Control movement across environments:** Proxy caching, one-time transfers, and remote sync cover connected, isolated, and multi-data-center environments; bandwidth policies, task progress, and logs keep transfers controlled and observable.

For regulated industries, the goal is not merely to download models faster. Models must be able to **enter the network, remain governed, move efficiently, and leave a trace**. That is the role MatrixHub is designed to play inside the enterprise network.

</div>
