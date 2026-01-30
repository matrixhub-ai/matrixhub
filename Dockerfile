ARG ALPINE_VERSION=3.23
ARG NODE_VERSION=25.2
ARG GO_VERSION=1.25

ARG BASE_IMAGE_PREFIX=docker.io
ARG NPM_CONFIG_REGISTRY
ARG GOPROXY
ARG HTTP_PROXY
ARG HTTPS_PROXY

ARG LD_FLAGS

##########################################

FROM ${BASE_IMAGE_PREFIX}/library/node:${NODE_VERSION}-alpine${ALPINE_VERSION} AS web-builder

WORKDIR /app/web

ARG NPM_CONFIG_REGISTRY
ENV NPM_CONFIG_REGISTRY=${NPM_CONFIG_REGISTRY}
ARG HTTP_PROXY
ENV HTTP_PROXY=${HTTP_PROXY}
ARG HTTPS_PROXY
ENV HTTPS_PROXY=${HTTPS_PROXY}
RUN --mount=type=cache,target=/app/web/node_modules \
    --mount=type=cache,target=/root/.npm \ 
    --mount=type=bind,source=./web/package.json,target=/app/web/package.json \
    npm install

COPY web /app/web

RUN --mount=type=cache,target=/app/web/node_modules \
    --mount=type=cache,target=/root/.npm \
    npm run build

##########################################

FROM ${BASE_IMAGE_PREFIX}/library/golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /app

RUN --mount=type=cache,target=/var/cache/apk \
    apk add gcc musl-dev

ARG GOPROXY
ENV GOPROXY=${GOPROXY}
ARG HTTP_PROXY
ENV HTTP_PROXY=${HTTP_PROXY}
ARG HTTPS_PROXY
ENV HTTPS_PROXY=${HTTPS_PROXY}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=./go.mod,target=/app/go.mod \
    --mount=type=bind,source=./go.sum,target=/app/go.sum \
    go mod download

COPY cmd /app/cmd
COPY internal /app/internal
COPY pkg /app/pkg
COPY web /app/web
COPY go.mod go.sum /app/
COPY --from=web-builder /app/web/dist /app/web/dist

ARG LD_FLAGS
ENV LD_FLAGS=${LD_FLAGS}
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 go build  -ldflags="${LD_FLAGS}" -tags embedweb -o /matrixhub ./cmd/poc

##########################################

FROM ${BASE_IMAGE_PREFIX}/library/alpine:${ALPINE_VERSION} AS matrixhub

RUN --mount=type=cache,target=/var/cache/apk \
    apk add ca-certificates git s3fs-fuse && \
    update-ca-certificates

COPY --from=builder /matrixhub /usr/local/bin/matrixhub

EXPOSE 9527

ENTRYPOINT ["/usr/local/bin/matrixhub"]