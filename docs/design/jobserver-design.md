# JobServer Technical Design

## Goals

JobServer runs cron-scheduled or manually triggered background jobs without
blocking API requests. Model synchronization is the first use case; cleanup and
other jobs can reuse the same framework.

- Add each job type as an independent processor with its own concurrency and
  timeout limits.
- Persist and claim jobs in the database with compare-and-swap (CAS); no
  external queue is needed.
- Run with the API server to keep phase-one deployment simple.

## Delayed-job architecture

All processors use the same delayed-job pattern. The following diagram uses
`syncPolicy processor` as the example.

```mermaid
%%{init: {"flowchart": {"nodeSpacing": 80, "rankSpacing": 90, "padding": 24}, "themeVariables": {"fontSize": "20px"}}}%%
flowchart TB
  subgraph API["API Server"]
    direction TB

    subgraph JS["JobServer"]
      direction LR
      Timer["Immediate poll<br/>and periodic ticker"]
      Processor["Sync Policy<br/>Processor"]
      Timer -->|Trigger| Processor
    end

    Service["SyncPolicyService"]
    Processor -->|Find and claim due policies| Service
  end

  subgraph DB["Database"]
    direction LR
    Policies[("sync_policies<br/>Schedule and claim")]
    Tasks[("sync_tasks<br/>Pending task")]
  end

  Service -->|Select and CAS<br/>next_run_at| Policies
  Service -->|Create| Tasks

  classDef trigger fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#172554,font-size:20px
  classDef processor fill:#EDE9FE,stroke:#7C3AED,stroke-width:3px,color:#2E1065,font-size:22px
  classDef service fill:#DCFCE7,stroke:#16A34A,stroke-width:2px,color:#14532D,font-size:20px
  classDef storage fill:#FFEDD5,stroke:#EA580C,stroke-width:2px,color:#7C2D12,font-size:20px

  class Timer trigger
  class Processor processor
  class Service service
  class Policies,Tasks storage
```

1. JobServer triggers the Sync Policy Processor at startup and on each tick.
2. The processor asks `SyncPolicyService` for policies that are due to run.
3. The service claims each policy by atomically advancing its `next_run_at`.
4. For every claimed policy, the service creates a Pending sync task for the
   next processor.

### Detailed sequence

```mermaid
sequenceDiagram
  participant Timer
  participant Processor
  participant Service
  participant Database

  Timer->>Processor: Poll
  Processor->>Service: Find due policies
  Service->>Database: Select due policy rows
  Database-->>Service: Candidates

  loop For each candidate
    Service->>Database: CAS advance next run time
    alt CAS succeeds
      Database-->>Service: Claimed
    else CAS fails
      Database-->>Service: Skipped
    end
  end

  Service-->>Processor: Claimed policies

  loop For each claimed policy
    Processor->>Service: Create Pending task
    Service->>Database: Insert Pending sync task
    Database-->>Service: Created
    Service-->>Processor: Done
  end

  Processor-->>Timer: Wait for next tick
```

The other processors reuse the same structure:

| Processor | Poll and claim | Execute |
|---|---|---|
| `syncPolicy` | Due `next_run_at`; CAS advances the schedule | Create a Pending task |
| `syncTask` | Pending task; CAS to Running | Generate Pending jobs |
| `syncJob` | Pending job; CAS to Running | By job type: run Git synchronization or cache a model |

Manual synchronization skips `syncPolicy` and creates a Pending task directly.

A sync job has a type, stored in `sync_jobs.job_type`. A `sync` job pulls or
pushes a resource for a sync task. A `model_cache` job pulls a model of a proxy
project from upstream and belongs to no sync task; its parameters, such as the
revision, are stored as JSON in `sync_jobs.params`. Jobs of both types share the
`syncJob` processor and its limits.

### Processing pipeline

**Three-stage pipeline:** policy scheduling, job generation, and transfer
execution can be tuned independently.

```mermaid
flowchart LR
  Policy["Sync Policy Processor<br/>Schedule"]
  Task["Sync Task Processor<br/>Generate jobs"]
  Job["Sync Job Processor<br/>Synchronize model"]

  Policy -->|"Pending sync task"| Task
  Task -->|"Pending sync jobs"| Job

  classDef processor fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#2E1065,font-size:18px
  class Policy,Task,Job processor
```

## Configuration

JobServer settings and defaults are defined under
[`jobServer` in `values.yaml`](../../deploy/charts/matrixhub/values.yaml).

## Phase-one boundaries

- Run one JobServer-enabled API replica.
- CAS prevents duplicate claim winners but does not provide execution leases.
- Detected failures are recorded but not retried; abandoned work is not
  recovered.
- Cancellation and job logs are local to the API server process.
- Multi-replica recovery, shared logs, retry policies, and worker metrics are
  future improvements.
