# Build stage for Deployments Service
FROM golang:1.25-alpine AS go-builder

# Accept build arguments for multi-arch support
ARG TARGETARCH=amd64
ARG TARGETOS=linux

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Enable Go build cache and module cache
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/go/pkg/mod

# Copy go mod files for shared and service
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/
COPY apps/deployments-service/go.mod apps/deployments-service/go.sum ./apps/deployments-service/

# Download dependencies for shared module
WORKDIR /build/apps/shared
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Download dependencies for service module
WORKDIR /build/apps/deployments-service
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy all source code
WORKDIR /build
COPY apps/shared/ ./apps/shared/
COPY apps/deployments-service/ ./apps/deployments-service/

# Build the service from its own module directory
WORKDIR /build/apps/deployments-service
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -o /build/deployments-service .

# Build stage for nixpacks
FROM rust:1.75-alpine AS nixpacks-builder

# Install build dependencies
RUN apk add --no-cache musl-dev

# Install nixpacks via cargo with cache mounts for faster rebuilds
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=/root/.cargo/registry \
    cargo install nixpacks --locked

# Final stage
ARG TARGETARCH=amd64
ARG TARGETOS=linux
FROM alpine:latest

# Install CA certificates for HTTPS requests
# Install docker CLI and docker-compose for deployment management
# Install git and curl for repository cloning and downloading buildx
# Install netcat-openbsd for health checks
# Install bash and openssh-client for build scripts and SSH-based git operations
# Disable triggers to avoid QEMU emulation issues with apk triggers
# This layer will be cached unless package versions change
RUN apk update && APK_NO_TRIGGERS=1 apk --no-cache add \
    ca-certificates \
    docker-cli \
    docker-compose \
    git \
    curl \
    netcat-openbsd \
    bash \
    openssh-client

# Set environment variables early for better layer caching
ENV DOCKER_CLI_PLUGIN_PATH=/root/.docker/cli-plugins \
    BUILDX_VERSION=0.29.1 \
    MISE_VERSION=2025.10.13 \
    RAILPACK_VERSION=0.9.2

# Install docker-buildx plugin from Docker's official releases
# BuildKit requires buildx for advanced build features (faster builds, better caching)
# This layer will be cached unless BUILDX_VERSION changes
# Use TARGETARCH to download the correct buildx binary for the target architecture
# Map architecture names: amd64 -> amd64, arm64 -> aarch64
RUN mkdir -p /root/.docker/cli-plugins && \
    ARCH=${TARGETARCH:-amd64} && \
    if [ "$ARCH" = "arm64" ]; then ARCH="aarch64"; fi && \
    curl -fsSL "https://github.com/docker/buildx/releases/download/v${BUILDX_VERSION}/buildx-v${BUILDX_VERSION}.linux-${ARCH}" -o /tmp/docker-buildx && \
    chmod +x /tmp/docker-buildx && \
    mv /tmp/docker-buildx /root/.docker/cli-plugins/docker-buildx && \
    rm -f /tmp/docker-buildx

# Install mise (version manager used by railpack and for installing railpack)
# Mise is used by railpack to install tools like pnpm, nodejs, etc.
# Railpack expects mise at /tmp/railpack/mise/mise-{version}, so we install it there
# This layer will be cached unless MISE_VERSION changes
RUN curl https://mise.run | sh && \
    mkdir -p /tmp/railpack/mise && \
    cp /root/.local/bin/mise /tmp/railpack/mise/mise-${MISE_VERSION} && \
    chmod +x /tmp/railpack/mise/mise-${MISE_VERSION} && \
    ln -sf /tmp/railpack/mise/mise-${MISE_VERSION} /usr/local/bin/mise

# Install railpack using mise (Railway's build tool, separate from nixpacks)
# Railpack is a Go-based tool that replaces nixpacks with improved performance
# According to https://railpack.com/installation, use: mise use ubi:railwayapp/railpack@latest
# This layer will be cached unless RAILPACK_VERSION changes
RUN mkdir -p /tmp/mise-install && \
    cd /tmp/mise-install && \
    mise use ubi:railwayapp/railpack@${RAILPACK_VERSION} && \
    RAILPACK_PATH=$(mise which railpack) && \
    ln -sf "$RAILPACK_PATH" /usr/local/bin/railpack && \
    cd / && \
    rm -rf /tmp/mise-install

# Copy nixpacks binary from builder (still needed for NIXPACKS strategy)
COPY --from=nixpacks-builder /usr/local/cargo/bin/nixpacks /usr/local/bin/nixpacks

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/deployments-service .

# Expose port
EXPOSE 3005

# Run the service
CMD ["./deployments-service"]
