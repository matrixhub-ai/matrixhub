# Contributing Guidelines

Welcome to MatrixHub! We are excited to have you here. This project follows the CNCF [Code of Conduct](CODE_OF_CONDUCT.md). An excerpt:

_As contributors and maintainers of this project, and in the interest of fostering an open and welcoming community, we pledge to respect all people who contribute through reporting issues, posting feature requests, updating documentation, submitting pull requests or patches, and other activities._

## Before You Start

- Read the [Code of Conduct](CODE_OF_CONDUCT.md).
- Check open [issues](https://github.com/matrixhub/matrixhub/issues) and [pull requests](https://github.com/matrixhub/matrixhub/pulls) to avoid duplicate work.

## Local Development

### Prerequisites

- [Go](https://go.dev/)
- [Node.js](https://nodejs.org/en)

### Build and run

```bash
make build
./bin/matrixhub
```

Open the app at http://localhost:9527.

## Faster Builds with Proxies

If downloads are slow, you can use HTTP(S) proxies and an image mirror:

```bash
export HTTP_PROXY=http://your-proxy:port
export HTTPS_PROXY=http://your-proxy:port
export BASE_IMAGE_PREFIX=m.daocloud.io/docker.io
make build
```

Or configure the Go module proxy:

```bash
export GOPROXY=https://goproxy.io,direct
make build
```

## Run with Docker Compose

Build and start the local stack:

```bash
make compose-deploy
```

## Run in local environment

```bash
go run main.go
```

## Contact

- [Slack](https://cloud-native.slack.com/archives/C0A8UKWR8HG) - Join us in the CNCF Slack workspace.
