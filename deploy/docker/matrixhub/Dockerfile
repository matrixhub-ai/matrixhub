ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.23

ARG BASE_IMAGE_PREFIX=docker.io
ARG GOPROXY
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG LD_FLAGS
ARG NODE_IMAGE_VERSION=22

##########################################

FROM --platform=$BUILDPLATFORM ${BASE_IMAGE_PREFIX}/library/golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /workspace

ARG GOPROXY
ENV GOPROXY=${GOPROXY}
# HTTP_PROXY/HTTPS_PROXY are predefined Docker build args: declaring them as
# ARG makes them available as env vars during RUN, without being persisted
# into the image config.
ARG HTTP_PROXY
ARG HTTPS_PROXY

RUN --mount=type=cache,target=/var/cache/apk \
    apk add gcc musl-dev

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

ARG TARGETARCH
ARG LD_FLAGS
ENV LD_FLAGS=${LD_FLAGS}
RUN --mount=type=cache,target=/go/pkg/mod \
    GOOS=linux GOARCH=$TARGETARCH \
    go build -ldflags="${LD_FLAGS}" -o /matrixhub ./cmd/matrixhub

##########################################
FROM ${BASE_IMAGE_PREFIX}/node:${NODE_IMAGE_VERSION} AS ui
ENV CI=true
WORKDIR /app/ui
ARG HTTP_PROXY
ARG HTTPS_PROXY
COPY ui /app/ui/
COPY api /app/api/
RUN corepack enable
RUN pnpm install --frozen-lockfile
RUN pnpm build

##########################################
FROM ${BASE_IMAGE_PREFIX}/library/alpine:${ALPINE_VERSION} AS matrixhub

# Proxy args are declared (but not ENV'd) so networked steps below can use
# them without persisting proxy settings into the runtime image.
ARG HTTP_PROXY
ARG HTTPS_PROXY

RUN --mount=type=cache,target=/var/cache/apk \
    apk add ca-certificates git && \
    update-ca-certificates

COPY --from=builder /matrixhub /usr/local/bin/matrixhub

COPY db/migrations /etc/matrixhub/migrations

COPY --from=ui /app/ui/dist /app

EXPOSE 9527

CMD ["/usr/local/bin/matrixhub", "apiserver"]
