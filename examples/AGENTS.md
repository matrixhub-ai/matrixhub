# AGENTS Guidelines for MatrixHub Example Repository

This file follows the [agents.md specification](https://agents.md) - a simple, open format for guiding AI coding agents.

## Overview

This example repository demonstrates best practices for AI model management in MatrixHub. The AGENTS.md file provides instructions to help AI coding agents understand how to work with this repository effectively.

## Dev Environment Tips

- MatrixHub uses Go 1.25+ and requires git for repository management
- Build the project with `go build -o bin/matrixhub cmd/matrixhub/main.go`
- Run locally with `./bin/matrixhub -addr :9527 -data ./data`
- Use `go mod tidy` to keep dependencies synchronized
- The main package is in `cmd/matrixhub/main.go`
- Core logic resides in `pkg/` packages: backend, repository, lfs, queue

## Repository Structure

- `cmd/matrixhub/` - Main application entry point
- `pkg/backend/` - HTTP handlers and API endpoints
- `pkg/repository/` - Git repository operations and file access
- `pkg/lfs/` - Git LFS support for large files
- `pkg/queue/` - Background job processing
- `web/` - Frontend web UI (if building with embedweb tag)
- `docs/` - Design documents and technical documentation
- `examples/` - Example files and templates

## Testing Instructions

- Run all tests: `go test ./pkg/...`
- Run specific package tests: `go test ./pkg/repository -v`
- Run with coverage: `go test ./pkg/... -cover`
- Tests use temporary directories and clean up automatically
- Git operations in tests require git to be installed

## Code Quality

- Run linter: `make ci-lint` or use golangci-lint directly
- Format code: `go fmt ./...`
- Verify modules: `go mod verify && go mod tidy`
- Follow existing code patterns and naming conventions
- Keep functions focused and testable

## Adding New Features

1. **Design First**: Create a design document in `docs/` describing the feature
2. **Tests First**: Write tests before implementation when possible  
3. **Minimal Changes**: Make the smallest changes necessary
4. **Documentation**: Update relevant docs and examples
5. **Security**: Run security scans before committing

## API Development

- New endpoints go in `pkg/backend/handler_*.go` files
- Follow HuggingFace-compatible API patterns where applicable
- Use gorilla/mux for routing
- Return proper HTTP status codes (404 for not found, 500 for errors)
- Log errors appropriately

## Working with Repositories

- Use `pkg/repository` package for all git operations
- Don't use git commands directly - use the repository abstraction
- Handle missing files gracefully (return specific errors)
- Respect file size limits to prevent DoS
- Support both branches and commit SHAs as references

## PR Instructions

- Title format: `[component] Brief description` (e.g., `[backend] Add AGENTS.md support`)
- Always run `make verify` before committing (includes lint, fmt, tests)
- Keep PRs focused on a single change or feature
- Update tests for any code changes
- Update documentation for user-facing changes
- Link to related issues or design docs

## Security Guidelines

- Never commit secrets or credentials
- Validate all user input
- Implement file size limits for uploads
- Use path validation to prevent traversal attacks
- Run `codeql` scanner before submitting PRs
- Check dependencies for known vulnerabilities

## Common Tasks

### Adding a New API Endpoint

```go
// In pkg/backend/handler_*.go
func (h *Handler) handleNewFeature(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    // Implementation
}

// In registryHuggingFace or similar router function
r.HandleFunc("/api/new-feature/{param}", h.handleNewFeature).Methods(http.MethodGet)
```

### Reading a File from Repository

```go
blob, err := repo.Blob(ref, "path/to/file")
if err != nil {
    // Handle error
}
reader, err := blob.NewReader()
defer reader.Close()
content, err := io.ReadAll(reader)
```

### Adding a New Test

```go
func TestNewFeature(t *testing.T) {
    t.Run("SuccessCase", func(t *testing.T) {
        // Setup
        // Execute
        // Assert
    })
    t.Run("ErrorCase", func(t *testing.T) {
        // Test error handling
    })
}
```

## Troubleshooting

- **Build fails**: Run `go mod tidy` to sync dependencies
- **Tests fail**: Check if git is installed and configured
- **Lint errors**: Run `make lint-fix` to auto-fix many issues
- **Import errors**: Verify go.mod and run `go mod download`

## Additional Resources

- [MatrixHub Documentation](../website/docs/)
- [Design Documents](../docs/)
- [Contributing Guide](../CONTRIBUTING.md)
- [agents.md Specification](https://agents.md)

## Questions?

- Check existing issues and PRs for similar questions
- Review design documents for architectural decisions
- Look at existing code for patterns and examples
- Ask in Slack or GitHub discussions
