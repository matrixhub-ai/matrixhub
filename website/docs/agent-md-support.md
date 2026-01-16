# Agent.md Support in MatrixHub

MatrixHub provides comprehensive support for `agent.md` files, allowing you to document and configure AI agents with structured metadata and rich Markdown content.

## What is agent.md?

The `agent.md` file is a standardized configuration and documentation file for AI agents and models. It combines:

- **YAML Frontmatter**: Structured metadata about the agent
- **Markdown Body**: Human-readable documentation, usage instructions, and examples

## File Format

### Basic Structure

Create an `agent.md` file at the root of your repository:

```markdown
---
name: "My AI Agent"
version: "1.0.0"
type: "assistant"
tags: ["coding", "automation"]
description: "A brief description of the agent"
---

# My AI Agent

Detailed documentation about your agent goes here.

## Features

- Feature 1
- Feature 2

## Usage Examples

\`\`\`python
# Example code
agent.run("task")
\`\`\`
```

### Metadata Fields

The YAML frontmatter supports the following fields (all optional):

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name of the agent |
| `version` | string | Version identifier (e.g., "1.0.0") |
| `type` | string | Agent type (e.g., "assistant", "model", "tool") |
| `tags` | array | List of tags for categorization |
| `description` | string | Brief description of the agent |

### Markdown Body

The content after the frontmatter can include:

- Detailed descriptions
- Usage instructions
- Code examples
- Configuration options
- Limitations and known issues
- References and links

## Adding agent.md to Your Repository

### Method 1: Git Command Line

```bash
# Clone your repository
git clone http://your-matrixhub/my-agent.git
cd my-agent

# Create agent.md
cat > agent.md << 'EOF'
---
name: "Code Assistant"
version: "1.0.0"
type: "assistant"
tags: ["coding", "python", "javascript"]
description: "An AI assistant for code generation and review"
---

# Code Assistant

This agent helps with coding tasks including:
- Code generation
- Code review
- Bug fixing
- Documentation

## Supported Languages

- Python
- JavaScript
- Go
EOF

# Commit and push
git add agent.md
git commit -m "Add agent.md"
git push origin main
```

### Method 2: HuggingFace CLI

If you're using MatrixHub as a HuggingFace replacement:

```bash
# Set your MatrixHub endpoint
export HF_ENDPOINT=http://your-matrixhub:9527

# Create agent.md locally
cat > agent.md << 'EOF'
---
name: "My Agent"
version: "1.0.0"
---
# Agent documentation
EOF

# Use git to push (HF CLI uses git under the hood)
git clone http://your-matrixhub/my-agent.git
cd my-agent
cp /path/to/agent.md .
git add agent.md
git commit -m "Add agent.md"
git push
```

## Accessing Agent Metadata

MatrixHub exposes agent metadata through multiple API endpoints.

### 1. Dedicated Agent Metadata Endpoint

Get parsed agent metadata as JSON:

```bash
# Get agent metadata from default branch
curl http://your-matrixhub:9527/api/models/my-agent/agent

# Get agent metadata from specific branch/revision
curl http://your-matrixhub:9527/api/models/my-agent/agent/revision/v2
```

Response example:

```json
{
  "name": "Code Assistant",
  "version": "1.0.0",
  "type": "assistant",
  "tags": ["coding", "python", "javascript"],
  "description": "An AI assistant for code generation and review",
  "rawContent": "---\nname: \"Code Assistant\"\n..."
}
```

### 2. Model Info Endpoint (HuggingFace API)

Agent metadata is automatically included in model info responses:

```bash
curl http://your-matrixhub:9527/api/models/my-agent
```

Response example:

```json
{
  "id": "my-agent",
  "modelId": "my-agent",
  "sha": "abc123...",
  "siblings": [...],
  "agentMetadata": {
    "name": "Code Assistant",
    "version": "1.0.0",
    "type": "assistant",
    "tags": ["coding", "python", "javascript"],
    "description": "An AI assistant for code generation and review"
  }
}
```

### 3. Raw File Download

Download the raw `agent.md` file:

```bash
# Download from default branch
curl http://your-matrixhub:9527/my-agent/resolve/main/agent.md

# Download from specific revision
curl http://your-matrixhub:9527/my-agent/resolve/v2/agent.md
```

## Python Usage Example

```python
import requests

# Configure MatrixHub endpoint
MATRIXHUB_URL = "http://your-matrixhub:9527"

def get_agent_info(agent_id):
    """Get agent metadata from MatrixHub"""
    url = f"{MATRIXHUB_URL}/api/models/{agent_id}/agent"
    response = requests.get(url)
    
    if response.status_code == 404:
        print(f"Agent {agent_id} has no agent.md file")
        return None
    
    response.raise_for_status()
    return response.json()

# Get agent metadata
agent_info = get_agent_info("my-agent")
if agent_info:
    print(f"Agent: {agent_info['name']}")
    print(f"Version: {agent_info['version']}")
    print(f"Tags: {', '.join(agent_info['tags'])}")
```

## JavaScript/Node.js Usage Example

```javascript
const axios = require('axios');

const MATRIXHUB_URL = 'http://your-matrixhub:9527';

async function getAgentInfo(agentId) {
  try {
    const response = await axios.get(
      `${MATRIXHUB_URL}/api/models/${agentId}/agent`
    );
    return response.data;
  } catch (error) {
    if (error.response?.status === 404) {
      console.log(`Agent ${agentId} has no agent.md file`);
      return null;
    }
    throw error;
  }
}

// Usage
getAgentInfo('my-agent').then(info => {
  if (info) {
    console.log(`Agent: ${info.name}`);
    console.log(`Version: ${info.version}`);
    console.log(`Tags: ${info.tags.join(', ')}`);
  }
});
```

## Best Practices

### 1. Version Your Agents

Always include a version field in your agent.md:

```yaml
---
name: "My Agent"
version: "2.1.0"
---
```

Use semantic versioning (MAJOR.MINOR.PATCH) to track changes.

### 2. Provide Clear Examples

Include practical usage examples in the markdown body:

```markdown
## Quick Start

\`\`\`python
from my_agent import Agent

agent = Agent()
result = agent.run("Write a hello world program")
print(result)
\`\`\`
```

### 3. Document Limitations

Be transparent about what your agent can and cannot do:

```markdown
## Limitations

- Does not support real-time streaming
- Maximum context length: 4096 tokens
- Best suited for Python and JavaScript code
```

### 4. Keep Descriptions Concise

The description field should be a brief summary (1-2 sentences):

```yaml
description: "A coding agent specialized in Python development and testing"
```

Save detailed explanations for the markdown body.

### 5. Use Meaningful Tags

Tags help with discovery and organization:

```yaml
tags: ["nlp", "text-generation", "gpt", "transformer"]
```

## File Size Limits

- Maximum `agent.md` file size: **1MB**
- Files exceeding this limit will be rejected

## Optional vs Required

All agent.md features are **optional**:

- Repositories without `agent.md` work normally
- API returns 404 for missing `agent.md` files
- Frontmatter fields are all optional
- You can have just markdown without frontmatter

## Example agent.md Files

### Example 1: Code Generation Agent

```markdown
---
name: "CodeGen Pro"
version: "3.2.1"
type: "code-generator"
tags: ["coding", "python", "javascript", "typescript"]
description: "Advanced code generation with multi-language support"
---

# CodeGen Pro

CodeGen Pro is an advanced AI agent for code generation across multiple programming languages.

## Supported Languages

- Python 3.8+
- JavaScript (ES6+)
- TypeScript
- Go
- Rust

## Usage

\`\`\`python
from codegen import CodeGenPro

agent = CodeGenPro()
code = agent.generate(
    "Create a REST API server with authentication",
    language="python"
)
\`\`\`

## Configuration

Configure the agent using environment variables:

- `CODEGEN_MODEL`: Model to use (default: "codegen-16b")
- `CODEGEN_TEMPERATURE`: Generation temperature (default: 0.2)
```

### Example 2: Data Analysis Agent

```markdown
---
name: "DataBot"
version: "1.5.0"
type: "assistant"
tags: ["data", "analysis", "pandas", "visualization"]
description: "AI assistant for data analysis and visualization tasks"
---

# DataBot - Data Analysis Assistant

DataBot helps you analyze datasets and create visualizations.

## Features

- Automated data cleaning
- Statistical analysis
- Chart generation
- Report creation

## Quick Start

\`\`\`python
from databot import DataBot

bot = DataBot()
bot.load_csv("sales.csv")
bot.analyze()
bot.plot("revenue_by_month")
\`\`\`
```

### Example 3: Minimal Agent

```markdown
# Simple Agent

This is a minimal agent.md with no frontmatter.

The parser will extract a description from the markdown content automatically.
```

## Troubleshooting

### Agent metadata not showing up

1. Ensure `agent.md` is at the repository root (not in a subdirectory)
2. Verify the file is committed and pushed to the branch
3. Check that YAML frontmatter is valid (use a YAML validator)

### Invalid YAML errors

If your YAML frontmatter has syntax errors:

- The parser will skip the frontmatter gracefully
- The markdown body will still be available
- Check YAML syntax with online validators

### File too large error

If you get "file too large" errors:

- Keep `agent.md` under 1MB
- Move lengthy documentation to separate files
- Link to external docs from agent.md

## API Reference

### GET /api/models/{repo}/agent

Get agent metadata for the default branch.

**Parameters:**
- `repo` (path): Repository name (without .git suffix)

**Response:** Agent metadata JSON or 404 if not found

### GET /api/models/{repo}/agent/revision/{revision}

Get agent metadata for a specific branch/revision.

**Parameters:**
- `repo` (path): Repository name
- `revision` (path): Branch name or commit SHA

**Response:** Agent metadata JSON or 404 if not found

### GET /api/models/{repo}

Get model info (includes agent metadata if available).

**Response:** Model info JSON with optional `agentMetadata` field

## See Also

- [Design Document](../docs/design-agent-md-support.md) - Technical design details
- [API Documentation](./api-reference.md) - Complete API reference
- [Contributing Guide](../CONTRIBUTING.md) - How to contribute
