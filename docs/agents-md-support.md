# Design Document: AGENTS.md Support in MatrixHub

## Overview

This document describes the design and implementation of support for `AGENTS.md` files in MatrixHub, following the [agents.md specification](https://agents.md).

## Background

The agents.md specification defines AGENTS.md as a simple, open format for guiding AI coding agents. It's a plain markdown file containing instructions for AI agents on how to work with a codebase, including development environment setup, testing procedures, and PR workflows.

MatrixHub, as an AI model registry, can benefit from supporting AGENTS.md files in model repositories to help AI agents understand how to work with specific models and their codebases.

## Goals

1. **Read and serve** AGENTS.md files from model repositories
2. **Expose content** through API endpoints
3. **Integrate with HuggingFace API** to include AGENTS.md in model information
4. **Maintain simplicity** - no parsing, just return plain markdown content
5. **Ensure security** with file size limits and path validation

## Non-Goals

- Parsing or interpreting AGENTS.md content
- Providing a web UI for editing AGENTS.md files
- Validating AGENTS.md structure or content
- Executing any code from AGENTS.md

## Design

### 1. AGENTS.md File Format

Following the [agents.md specification](https://agents.md), AGENTS.md is:
- A plain markdown file (no YAML frontmatter or special parsing)
- Located at the repository root
- Contains human-readable instructions for AI coding agents
- Typically includes sections like:
  - Development environment tips
  - Testing instructions
  - PR guidelines
  - Code quality standards
  - Common tasks and troubleshooting

### 2. Component Architecture

#### 2.1 Content Reader (`pkg/repository/agent.go`)

Simple file reader that retrieves AGENTS.md content:

```go
type AgentsContent struct {
    Content string `json:"content"` // Full markdown content
}

func (r *Repository) GetAgentsContent(ref string) (*AgentsContent, error)
```

This function:
1. Checks if AGENTS.md exists at the repository root
2. Reads the file content
3. Returns the raw markdown content
4. Enforces file size limit (1MB)

#### 2.2 API Endpoints

##### New Dedicated Endpoints

**GET /api/models/{repo}/agents**
Returns AGENTS.md content for the default branch.

**GET /api/models/{repo}/agents/revision/{revision}**
Returns AGENTS.md content for a specific branch or commit.

Response format:
```json
{
  "content": "# AGENTS Guidelines\n\n## Dev environment tips\n..."
}
```

##### Enhanced Model Info Endpoints

Existing HuggingFace-compatible endpoints now include optional `agentsContent` field:

**GET /api/models/{repo}**
**GET /api/models/{repo}/revision/{revision}**

Response includes:
```json
{
  "id": "my-model",
  "modelId": "my-model",
  "agentsContent": {
    "content": "# AGENTS Guidelines\n..."
  }
}
```

### 3. Implementation Details

#### File Reading
- File path: `AGENTS.md` (exact case, at repository root)
- Maximum size: 1MB (prevents DoS attacks)
- Returns error if file doesn't exist (ErrAgentsFileNotFound)
- No caching (reads fresh from git each time)

#### Error Handling
- **File not found**: Return `ErrAgentsFileNotFound` (not a critical error)
- **File too large**: Return `ErrAgentsFileTooLarge`
- **Read error**: Return wrapped error with context
- **Empty repository**: Return 404 from API endpoints

#### Security Measures
1. **File size limit**: 1MB maximum to prevent DoS
2. **Path validation**: Only reads from repository root, prevents traversal
3. **No code execution**: Content is returned as-is, no parsing or interpretation
4. **Read-only**: No modification of AGENTS.md through API

### 4. Data Flow

```
Client Request
    ↓
API Endpoint (/api/models/{repo}/agents)
    ↓
Repository.GetAgentsContent(ref)
    ↓
Check AGENTS.md exists in ref
    ↓
Read file content (max 1MB)
    ↓
Return AgentsContent{Content: string}
    ↓
Serialize to JSON
    ↓
HTTP Response
```

### 5. Testing Strategy

#### Unit Tests (`pkg/repository/agent_test.go`)
- AGENTS.md file exists and is readable
- AGENTS.md file doesn't exist (returns error)
- Empty AGENTS.md file
- Different branches have different content
- File size enforcement
- HasAgentsFile() helper function

#### Integration Tests (`pkg/backend/handler_agent_test.go`)
- GET /api/models/{repo}/agents returns content
- GET /api/models/{repo}/agents returns 404 when missing
- Revision-specific endpoint works correctly
- Model info includes agentsContent field
- HuggingFace API compatibility

### 6. Backward Compatibility

- All AGENTS.md features are optional and additive
- Repositories without AGENTS.md continue to work unchanged
- API responses include agentsContent only when AGENTS.md exists
- No breaking changes to existing endpoints
- All existing tests continue to pass

## Implementation Plan

### Phase 1: Core Reader ✅
- Create `pkg/repository/agent.go` with simple file reader
- Add error types (ErrAgentsFileNotFound, ErrAgentsFileTooLarge)
- Implement GetAgentsContent() function
- Add HasAgentsFile() helper

### Phase 2: API Integration ✅
- Add dedicated endpoints in `pkg/backend/handler_hf.go`
- Update HuggingFace model info handlers
- Ensure proper error handling (404 for missing files)
- Add agentsContent field to HFModelInfo struct

### Phase 3: Testing ✅
- Unit tests for file reading and error cases
- Integration tests for API endpoints
- Test different branches and revisions
- Verify HuggingFace API compatibility

### Phase 4: Documentation ✅
- Add design document (this file)
- Create example AGENTS.md file
- Update examples/README.md
- Update main README.md

## Security Considerations

1. **File Size Limit**: 1MB maximum prevents DoS attacks
2. **Path Validation**: Only reads AGENTS.md from root, prevents directory traversal
3. **No Parsing**: No YAML or structured parsing reduces attack surface
4. **Read-Only**: API only reads, never writes AGENTS.md
5. **Error Messages**: Don't leak sensitive information

## Performance Considerations

1. **Direct Read**: No caching for simplicity, reads from git each time
2. **Small Files**: AGENTS.md expected to be small (<100KB typically)
3. **Minimal Processing**: Just read and return, no parsing overhead
4. **Parallel Safe**: Read-only operations are thread-safe

## Future Enhancements

Potential improvements for future versions:

1. **Caching**: Cache AGENTS.md content by commit SHA
2. **Compression**: Compress large AGENTS.md files in responses
3. **Web UI**: View AGENTS.md in browser UI
4. **Validation**: Optional schema validation for AGENTS.md structure
5. **Search**: Make AGENTS.md content searchable
6. **Templates**: Provide AGENTS.md templates for common scenarios

## References

- [agents.md Specification](https://agents.md)
- [agents.md GitHub Repository](https://github.com/agentsmd/agents.md)
- MatrixHub blob handling: `pkg/repository/blob.go`
- HuggingFace API handlers: `pkg/backend/handler_hf.go`

## Revision History

- 2026-01-19: Initial design document for AGENTS.md support
