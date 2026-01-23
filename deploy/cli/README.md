# @matrixhub/cli

CLI tool for deploying and managing MatrixHub - your self-hosted AI model registry.

## 🚀 Quick Start

No installation needed! Use `npx` to run MatrixHub directly:

```bash
npx @matrixhub/cli start
```

Or install globally:

```bash
npm install -g @matrixhub/cli
matrixhub start
```

## 📋 Prerequisites

- Docker installed and running
- Node.js 18+ (for npx/npm)

## 🎯 Commands

### `start`

Start a new MatrixHub container:

```bash
npx @matrixhub/cli start [options]
```

**Options:**
- `-p, --port <port>` - Port to expose (default: 9527)
- `-d, --data <path>` - Data directory path (default: ./data)
- `-n, --name <name>` - Container name (default: matrixhub)
- `--image <image>` - Docker image to use (default: ghcr.io/matrixhub-ai/matrixhub:main)

**Examples:**

```bash
# Start with defaults (port 9527, ./data directory)
npx @matrixhub/cli start

# Start on custom port with custom data directory
npx @matrixhub/cli start -p 8080 -d ~/matrixhub-data

# Use a specific image version
npx @matrixhub/cli start --image ghcr.io/matrixhub-ai/matrixhub:v1.0.0
```

### `stop`

Stop the running MatrixHub container:

```bash
npx @matrixhub/cli stop
```

### `restart`

Restart the MatrixHub container:

```bash
npx @matrixhub/cli restart
```

### `status`

Check the status of the MatrixHub container:

```bash
npx @matrixhub/cli status
```

### `logs`

View container logs:

```bash
npx @matrixhub/cli logs [options]
```

**Options:**
- `-f, --follow` - Follow log output (like `tail -f`)
- `--tail <lines>` - Number of lines to show from the end (default: 100)

**Examples:**

```bash
# View last 100 lines of logs
npx @matrixhub/cli logs

# Follow logs in real-time
npx @matrixhub/cli logs -f

# View last 500 lines
npx @matrixhub/cli logs --tail 500
```

### `update`

Update MatrixHub to the latest version:

```bash
npx @matrixhub/cli update
```

This will:
1. Pull the latest Docker image
2. Stop the current container
3. Remove the old container
4. Prompt you to start the new version

## 🎨 Features

- **Zero Configuration**: Works out of the box with sensible defaults
- **Interactive**: Prompts for confirmation when needed
- **Colorful Output**: Clear, easy-to-read terminal output
- **Error Handling**: Helpful error messages and recovery suggestions
- **Docker Detection**: Checks if Docker is installed before running

## 🛠️ Development

```bash
# Clone the repository
git clone https://github.com/matrixhub-ai/matrixhub.git
cd matrixhub/packages/cli

# Install dependencies
npm install

# Build
npm run build

# Link locally for testing
npm link

# Now you can use the command
matrixhub start
```

## 📄 License

Apache-2.0 - see [LICENSE](../../LICENSE) for details.

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for details.

## 📞 Support

- [GitHub Issues](https://github.com/matrixhub-ai/matrixhub/issues)
- [Slack Community](https://cloud-native.slack.com/archives/C0A8UKWR8HG)
