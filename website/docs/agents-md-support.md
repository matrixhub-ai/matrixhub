# AGENTS.md Support in MatrixHub

MatrixHub supports `AGENTS.md` files following the [agents.md specification](https://agents.md) - a simple, open format for guiding AI coding agents.

## What is AGENTS.md?

AGENTS.md is a plain markdown file containing instructions for AI coding agents on how to work with your codebase. Think of it as a README for AI agents.

It typically includes:
- Development environment setup tips
- Testing procedures and commands
- Code quality guidelines
- PR workflow and requirements
- Common tasks and troubleshooting

## Creating an AGENTS.md File

Create a file named `AGENTS.md` (uppercase) at the root of your repository:

```markdown
# AGENTS Guidelines for This Repository

## Dev environment tips
- Build the project with `go build`
- Run tests with `go test ./...`
- Use `go mod tidy` to sync dependencies

## Testing instructions
- All tests must pass before merging
- Run `go test -v` for verbose output
- Add tests for new features

## PR instructions
- Title format: [component] Brief description
- Run linter before committing
- Link to related issues
```

## Accessing AGENTS.md via API

### Get AGENTS.md Content

**Default branch:**
```bash
curl http://localhost:9527/api/models/my-repo/agents
```

**Specific branch or commit:**
```bash
curl http://localhost:9527/api/models/my-repo/agents/revision/main
curl http://localhost:9527/api/models/my-repo/agents/revision/feature-branch
curl http://localhost:9527/api/models/my-repo/agents/revision/abc1234
```

**Response:**
```json
{
  "content": "# AGENTS Guidelines for This Repository\n\n## Dev environment tips\n..."
}
```

### Model Info Includes AGENTS.md

When you request model information, it automatically includes AGENTS.md content if available:

```bash
curl http://localhost:9527/api/models/my-repo
```

**Response:**
```json
{
  "id": "my-repo",
  "modelId": "my-repo",
  "sha": "abc123...",
  "siblings": [...],
  "agentsContent": {
    "content": "# AGENTS Guidelines...\n"
  }
}
```

If AGENTS.md doesn't exist, the `agentsContent` field is omitted.

## Example AGENTS.md

See [examples/AGENTS.md](../../examples/AGENTS.md) for a complete example.

## Best Practices

### 1. Keep It Simple

AGENTS.md is plain markdown - no YAML frontmatter, no special syntax:

```markdown
# Project Guidelines

## Setup
- Step 1
- Step 2

## Testing
- Run tests
- Check coverage
```

### 2. Be Specific

Include exact commands and paths:

```markdown
## Build the project
\`\`\`bash
cd cmd/myapp
go build -o ../../bin/myapp
\`\`\`
```

### 3. Cover Common Tasks

Help AI agents understand frequent operations:

```markdown
## Common Tasks

### Adding a new API endpoint
1. Create handler in `pkg/api/handlers/`
2. Register route in `pkg/api/router.go`
3. Add tests in `pkg/api/handlers/handler_test.go`
```

### 4. Update When Changes Occur

Keep AGENTS.md in sync with your development process:

```markdown
## Recent Changes
- 2026-01: Migrated to Go 1.22+
- 2026-01: Switched from Make to Task for builds
```

## API Reference

### GET /api/models/{repo}/agents

Get AGENTS.md content for the default branch.

**Parameters:**
- `repo` (path): Repository name (without .git suffix)

**Response:** 200 OK with AgentsContent JSON, or 404 if not found

### GET /api/models/{repo}/agents/revision/{revision}

Get AGENTS.md content for a specific revision.

**Parameters:**
- `repo` (path): Repository name
- `revision` (path): Branch name or commit SHA

**Response:** 200 OK with AgentsContent JSON, or 404 if not found

### GET /api/models/{repo}

Get model info (includes AGENTS.md if available).

**Response:** Model info JSON with optional `agentsContent` field

## Troubleshooting

### AGENTS.md not showing up

1. Check the file is named `AGENTS.md` (uppercase, at repository root)
2. Verify the file is committed and pushed to the branch
3. Check the file is not empty
4. Try accessing via the revision endpoint to verify branch name

### File too large error

AGENTS.md has a 1MB size limit. If you need more space:
- Move lengthy documentation to separate files
- Link to external docs from AGENTS.md
- Keep only essential instructions in AGENTS.md

### Wrong content returned

Make sure you're requesting the correct branch:
```bash
# Specify the exact branch
curl http://localhost:9527/api/models/my-repo/agents/revision/main
```

## Specification

For the full agents.md specification, see: https://agents.md

## See Also

- [Design Document](../docs/agents-md-support.md) - Technical design details
- [Example AGENTS.md](../../examples/AGENTS.md) - Complete example
- [agents.md Specification](https://agents.md) - Official spec
