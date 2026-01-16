---
name: "Example Agent"
version: "1.0.0"
type: "assistant"
tags: ["example", "demo", "documentation"]
description: "A comprehensive example agent demonstrating agent.md features"
---

# Example Agent

This is an example `agent.md` file that demonstrates all the features and best practices for documenting AI agents in MatrixHub.

## Overview

Example Agent is a demonstration agent that showcases:

- Structured metadata using YAML frontmatter
- Rich Markdown documentation
- Code examples and usage instructions
- Best practices for agent documentation

## Features

### Core Capabilities

- **Natural Language Processing**: Understand and process user queries
- **Multi-language Support**: Work with Python, JavaScript, and Go
- **Context Management**: Maintain conversation context
- **Error Handling**: Graceful error recovery and reporting

### Advanced Features

- Code generation and completion
- Documentation generation
- Test case creation
- Code review and suggestions

## Installation

```bash
# Clone the repository
git clone http://your-matrixhub:9527/example-agent.git

# Install dependencies
cd example-agent
pip install -r requirements.txt
```

## Quick Start

### Python

```python
from example_agent import ExampleAgent

# Initialize the agent
agent = ExampleAgent(
    model="example-1.0",
    temperature=0.7
)

# Run a simple task
result = agent.run("Generate a hello world program in Python")
print(result)
```

### JavaScript

```javascript
const { ExampleAgent } = require('example-agent');

// Initialize the agent
const agent = new ExampleAgent({
  model: 'example-1.0',
  temperature: 0.7
});

// Run a simple task
agent.run('Generate a hello world program in JavaScript')
  .then(result => console.log(result));
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `EXAMPLE_MODEL` | Model version to use | `example-1.0` |
| `EXAMPLE_TEMPERATURE` | Generation temperature (0.0-1.0) | `0.7` |
| `EXAMPLE_MAX_TOKENS` | Maximum tokens in response | `2048` |
| `EXAMPLE_API_KEY` | API key for authentication | `None` |

### Configuration File

Create a `config.yaml` file:

```yaml
model: "example-1.0"
temperature: 0.7
max_tokens: 2048
features:
  code_generation: true
  documentation: true
  testing: true
```

## Usage Examples

### Code Generation

```python
agent = ExampleAgent()

# Generate a function
code = agent.generate_code(
    description="A function to calculate factorial",
    language="python"
)

print(code)
# Output:
# def factorial(n):
#     if n <= 1:
#         return 1
#     return n * factorial(n - 1)
```

### Documentation Generation

```python
agent = ExampleAgent()

# Generate documentation for code
docs = agent.generate_docs(
    code="""
    def add(a, b):
        return a + b
    """,
    format="markdown"
)

print(docs)
```

### Code Review

```python
agent = ExampleAgent()

# Review code and get suggestions
review = agent.review_code(
    code="""
    def calc(x, y):
        return x/y
    """,
    language="python"
)

print(review.suggestions)
# Output: ["Add error handling for division by zero", ...]
```

## API Reference

### Class: ExampleAgent

Main agent class for all operations.

#### Constructor

```python
ExampleAgent(
    model: str = "example-1.0",
    temperature: float = 0.7,
    max_tokens: int = 2048
)
```

**Parameters:**
- `model`: Model version to use
- `temperature`: Randomness in generation (0.0 = deterministic, 1.0 = very random)
- `max_tokens`: Maximum tokens in generated response

#### Methods

##### run(prompt: str) -> str

Execute a general task based on the prompt.

```python
result = agent.run("Write a hello world program")
```

##### generate_code(description: str, language: str) -> str

Generate code based on description.

```python
code = agent.generate_code(
    description="Sort a list of numbers",
    language="python"
)
```

##### generate_docs(code: str, format: str) -> str

Generate documentation for provided code.

```python
docs = agent.generate_docs(
    code="def hello(): return 'Hello'",
    format="markdown"
)
```

##### review_code(code: str, language: str) -> CodeReview

Review code and provide suggestions.

```python
review = agent.review_code(
    code="def divide(a, b): return a/b",
    language="python"
)
```

## Performance

### Benchmarks

| Task | Avg Time | Tokens/sec |
|------|----------|------------|
| Code Generation | 2.3s | 450 |
| Documentation | 1.8s | 520 |
| Code Review | 3.1s | 380 |

### Resource Requirements

- **Memory**: 4GB minimum, 8GB recommended
- **GPU**: Optional, improves performance by 3-5x
- **Disk**: 2GB for model files

## Limitations

### Known Limitations

1. **Context Length**: Maximum 4096 tokens (approximately 3000 words)
2. **Languages**: Best performance with Python, JavaScript, and Go
3. **Real-time**: Not suitable for streaming applications
4. **Internet**: Requires internet connection for model updates

### Roadmap

Features planned for future versions:

- [ ] Streaming support for real-time generation
- [ ] Extended context length (8192 tokens)
- [ ] Additional language support (Rust, C++)
- [ ] Fine-tuning capabilities
- [ ] Multi-agent collaboration

## Troubleshooting

### Common Issues

#### Issue: "Model not found"

**Solution**: Ensure the model is downloaded:

```bash
python -m example_agent download-model
```

#### Issue: "Out of memory"

**Solution**: Reduce max_tokens or use a smaller model:

```python
agent = ExampleAgent(
    model="example-small",
    max_tokens=1024
)
```

#### Issue: "Rate limit exceeded"

**Solution**: Add delays between requests:

```python
import time

for task in tasks:
    result = agent.run(task)
    time.sleep(1)  # Wait 1 second between requests
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone the repo
git clone http://your-matrixhub:9527/example-agent.git
cd example-agent

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dev dependencies
pip install -r requirements-dev.txt

# Run tests
pytest tests/
```

## License

MIT License - see [LICENSE](./LICENSE) for details.

## Support

- **Documentation**: [https://docs.example.com](https://docs.example.com)
- **Issues**: [GitHub Issues](https://github.com/example/example-agent/issues)
- **Slack**: [#example-agent](https://slack.example.com)
- **Email**: support@example.com

## Changelog

### v1.0.0 (2026-01-16)

- Initial release
- Core code generation features
- Documentation generation
- Code review capabilities
- Python and JavaScript support

## References

- [Agent Architecture](./docs/architecture.md)
- [Model Details](./docs/model.md)
- [API Specification](./docs/api.md)
- [Examples Repository](https://github.com/example/example-agent-examples)
