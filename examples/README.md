# MatrixHub Examples

This directory contains example files demonstrating MatrixHub features and best practices.

## Files

### AGENTS.md

A comprehensive example of an `AGENTS.md` file following the [agents.md specification](https://agents.md).

**What is AGENTS.md?**

AGENTS.md is a simple, open format for guiding AI coding agents. It's a plain markdown file containing instructions for AI agents on how to work with your codebase, including:

- Development environment setup
- Testing instructions
- Code quality guidelines
- PR workflow
- Common tasks and troubleshooting

**Usage:**

```bash
# Copy this example to your repository
cp examples/AGENTS.md /path/to/your/repo/AGENTS.md

# Customize it for your project
vi /path/to/your/repo/AGENTS.md

# Commit and push
cd /path/to/your/repo
git add AGENTS.md
git commit -m "Add AGENTS.md"
git push
```

**Viewing via API:**

```bash
# Get AGENTS.md content for a repository
curl http://localhost:9527/api/models/your-repo/agents

# Get AGENTS.md for a specific branch
curl http://localhost:9527/api/models/your-repo/agents/revision/main
```

## Documentation

For more information on AGENTS.md support in MatrixHub:

- [agents.md specification](https://agents.md)
- [Design Document](../docs/design-agent-md-support.md)

## Contributing Examples

Have a great example to share? Contribute it!

1. Create your example file
2. Add a description to this README
3. Submit a pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.
