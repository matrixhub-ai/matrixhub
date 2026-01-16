# Design Document: Agent.md Support in MatrixHub

## Overview

This document describes the design and implementation of comprehensive support for `agent.md` files in MatrixHub. The `agent.md` file serves as a standardized configuration and documentation file for AI agents and models, providing metadata, usage instructions, and examples.

## Background

Currently, MatrixHub primarily focuses on serving model files and provides basic support for README.md files. However, there is no first-class support for `agent.md` files, which are becoming a standard way to document AI agents with their configurations, capabilities, and usage patterns.

## Goals

1. **Parse and validate** `agent.md` files from repositories
2. **Expose agent metadata** through the existing API endpoints
3. **Provide dedicated endpoints** for retrieving agent.md content
4. **Integrate with HuggingFace API** to include agent.md in model information
5. **Maintain backward compatibility** with existing functionality

## Non-Goals

- Executing or running agent configurations
- Validating agent code or behavior
- Providing a web UI for editing agent.md files (may be added later)

## Design

### 1. Agent.md File Structure

The `agent.md` file is expected to be a Markdown file located at the root of a repository. It may contain:

- **Frontmatter** (YAML): Structured metadata about the agent
- **Body** (Markdown): Human-readable documentation

Example structure:
```markdown
---
name: "My AI Agent"
version: "1.0.0"
type: "assistant"
tags: ["coding", "automation"]
---

# My AI Agent

Description of the agent and its capabilities...
```

### 2. Component Architecture

#### 2.1 Agent Metadata Parser (`pkg/repository/agent.go`)

A new module for parsing agent.md files:

```go
type AgentMetadata struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Type        string   `json:"type"`
    Tags        []string `json:"tags"`
    Description string   `json:"description"`
    RawContent  string   `json:"rawContent,omitempty"`
}

func (r *Repository) GetAgentMetadata(ref string) (*AgentMetadata, error)
```

This function will:
1. Check if `agent.md` exists at the repository root
2. Parse the file content
3. Extract YAML frontmatter (if present)
4. Extract markdown body
5. Return structured metadata

#### 2.2 API Endpoints

##### Existing Endpoint Enhancement

Enhance the HuggingFace model info endpoints to include agent metadata:

**GET /api/models/{repo}**
**GET /api/models/{repo}/revision/{revision}**

Response will include a new optional field:
```json
{
  "id": "my-agent",
  "modelId": "my-agent",
  "sha": "abc123...",
  "siblings": [...],
  "agentMetadata": {
    "name": "My AI Agent",
    "version": "1.0.0",
    "type": "assistant",
    "tags": ["coding", "automation"]
  }
}
```

##### New Dedicated Endpoint

**GET /{repo}/agent.md** or **GET /{repo}/resolve/{revision}/agent.md**

This endpoint will serve the raw agent.md file, leveraging the existing file resolution mechanism.

**GET /api/models/{repo}/agent** (new endpoint)

Returns parsed agent metadata as JSON:
```json
{
  "name": "My AI Agent",
  "version": "1.0.0",
  "type": "assistant",
  "tags": ["coding", "automation"],
  "description": "Description from markdown body...",
  "rawContent": "full markdown content..."
}
```

### 3. Implementation Plan

#### Phase 1: Core Parser
1. Create `pkg/repository/agent.go` with parsing logic
2. Add YAML frontmatter parsing (optional dependency)
3. Handle missing or invalid agent.md gracefully

#### Phase 2: API Integration
1. Update HuggingFace handlers to include agent metadata
2. Add dedicated agent metadata endpoint
3. Ensure backward compatibility (agent metadata is optional)

#### Phase 3: Testing
1. Unit tests for agent.md parsing
2. Integration tests for API endpoints
3. Test edge cases (missing file, invalid format, etc.)

#### Phase 4: Documentation
1. Add usage examples to website docs
2. Document agent.md file format specification
3. Provide migration guide for existing repositories

### 4. Data Flow

```
Repository (Git)
    ↓
agent.md file exists?
    ↓ (yes)
Parse file with AgentMetadata parser
    ↓
Extract frontmatter (YAML) and body (Markdown)
    ↓
Return AgentMetadata struct
    ↓
Include in API responses (HF model info, dedicated endpoint)
    ↓
Client receives agent metadata
```

### 5. Error Handling

- **File not found**: Return nil metadata (not an error)
- **Invalid YAML**: Log warning, use empty metadata
- **Parse error**: Return error with details
- **Large files**: Limit agent.md to reasonable size (e.g., 1MB)

### 6. Security Considerations

1. **Content validation**: Sanitize markdown content to prevent injection attacks
2. **File size limits**: Prevent DoS by limiting agent.md file size
3. **Path traversal**: Ensure agent.md is only read from repository root
4. **No code execution**: Parser only reads and structures data

### 7. Performance Considerations

1. **Caching**: Consider caching parsed metadata per commit SHA
2. **Lazy loading**: Only parse agent.md when requested
3. **Minimal overhead**: Parsing should not significantly impact API response time

### 8. Backward Compatibility

- All agent.md features are optional and additive
- Existing repositories without agent.md continue to work unchanged
- API responses include agent metadata only when available
- No breaking changes to existing endpoints

## Future Enhancements

1. **Schema validation**: Validate agent.md against a JSON schema
2. **Web UI**: Add UI for viewing and editing agent.md
3. **Templates**: Provide agent.md templates for common agent types
4. **Multi-file support**: Support agent configurations in multiple files
5. **Version history**: Track changes to agent.md over time

## Testing Strategy

### Unit Tests
- AgentMetadata parser with various file formats
- YAML frontmatter extraction
- Edge cases (empty files, malformed YAML, etc.)

### Integration Tests
- API endpoint responses with and without agent.md
- HuggingFace API compatibility
- File serving through existing endpoints

### Manual Testing
- Create test repositories with agent.md files
- Verify API responses through HTTP requests
- Test with huggingface_hub library compatibility

## Documentation Requirements

1. **API Documentation**: Document new endpoints and response fields
2. **User Guide**: How to add agent.md to repositories
3. **Examples**: Sample agent.md files for different use cases
4. **Migration Guide**: How to add agent.md to existing repositories

## Success Criteria

1. ✅ Agent.md files can be parsed and validated
2. ✅ Metadata is exposed through API endpoints
3. ✅ HuggingFace API includes agent information
4. ✅ Comprehensive test coverage (>80%)
5. ✅ Documentation is complete and clear
6. ✅ No breaking changes to existing functionality
7. ✅ Security review passes
8. ✅ Performance impact is minimal (<10ms overhead)

## References

- MatrixHub existing blob handling: `pkg/repository/blob.go`
- HuggingFace API handlers: `pkg/backend/handler_hf.go`
- Repository tree structure: `pkg/repository/tree.go`

## Revision History

- 2026-01-16: Initial design document
