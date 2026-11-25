# ModelScope model interoperability

## Motivation

MatrixHub currently discovers upstream repositories through Hugging Face and
MatrixHub APIs and exposes HF-compatible downloads. Adding a ModelScope URL as
an HF provider does not supply ModelScope discovery, SDK authentication, or the
file-list and download responses expected by ModelScope clients.

This proposal adds model interoperability in both directions. It also keeps a
cold proxy's metadata requests independent of downloading its weight files.

## Proposed design

- Add the registry enum value `REGISTRY_TYPE_MODELSCOPE = 3`. Existing enum values
  and stored registry records keep their meanings.
- Register a paginated ModelScope discovery adapter for pull rules. Validate
  authenticated connections through the upstream login API. Exchange upstream
  SDK tokens for the separate Git token used to fetch private Git repositories.
- Preserve Git branches and tags and reuse MatrixHub's existing Git/LFS layout.
  Explicit pull/cache jobs fetch and verify complete weight objects. Proxy
  requests refresh Git refs under a per-repository lock and a short freshness
  interval, then return metadata without waiting for the LFS download.
- Add ModelScope-compatible model info, revision, recursive file-list and file
  download routes under `/api/v1/models/{namespace}/{repo}`. Run the existing
  project authorization hook before preparing or opening a repository.
- Serve ordinary Git blobs and cached LFS objects through the existing resolver.
  For an uncached LFS object, return HEAD metadata from its Git pointer and proxy
  GET/Range requests to the configured upstream with a pinned commit and path.
  Complete GET responses stream into the client and the validating LFS store.
  A partial response is never stored as a complete object. Range-only downloads
  therefore do not populate a complete LFS cache entry.
- Accept MatrixHub access tokens in bearer headers and ModelScope's
  `m_session_id` token cookie. Keep upstream credentials on the server, strip
  credentials when following file-download redirects to a different origin,
  and refuse redirects when posting an SDK token to the login API.

Repository implementations own the upstream network and Git operations. Domain
services coordinate project/registry lookup and metadata preparation. HTTP
handlers own authorization, protocol envelopes and streaming responses.

## API and UI changes

The project detail response gains the additive `registry_type` string field.
Download and vLLM instructions can then default to ModelScope only for an
explicit ModelScope provider and to HF for other providers. Users can select a
client independently of the project's upstream source.

Registry creation offers ModelScope with its base URL and credentials. Pull-rule
forms limit ModelScope resources to models and validate unsupported dataset
selections before submitting. 

## Compatibility boundaries

The ModelScope protocol routes cover model reads. Dataset compatibility and
ModelScope write/login APIs on MatrixHub are deferred. The `master` download
revision aliases Git HEAD when no actual `master` branch or tag exists, so
ModelScope clients using their default revision can read repositories created
on `main`. Explicit existing branch, tag and commit identifiers retain their
Git meaning.

No database migration is required. The upstream default branch is propagated
through model metadata updates, including existing model records initialized
on `main` before a proxy fetch.

## Test plan

- Unit tests for discovery pagination, provider round-tripping, private Git token
  exchange, upstream credentials, default branches and corrupt LFS rejection.
- HTTP tests for metadata and uncached HEAD, byte ranges, full GET caching,
  existing cached reads, permissions, revisions and invalid paths.
- Optional real ModelScope SDK/CLI tests and an opt-in public upstream pull test
  using temporary storage. These tests do not require a shared deployment.
- Frontend checks for provider defaults, ordinary HF behavior, private-token
  instructions, revision propagation and ModelScope resource validation.
- Existing API E2E smoke suites against an isolated local server and database.

## Alternatives

Increasing client timeouts leaves metadata requests coupled to model size and
cancels expensive work when the caller disconnects. Redirecting clients to the
upstream would complicate private credentials and bypass MatrixHub's data path.
The proposed read-through routes keep the normal client entry points while
returning metadata promptly.
