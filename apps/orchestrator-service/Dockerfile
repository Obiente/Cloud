# Build stage for Orchestrator Service
ARG BASE_REGISTRY=docker.io

FROM ${BASE_REGISTRY}/library/golang:1.25-alpine AS go-builder

# Accept build arguments for multi-arch support
ARG TARGETARCH=amd64
ARG TARGETOS=linux

# Install build dependencies
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk add --no-cache --no-scripts git make

# Set working directory
WORKDIR /build

# Pre-download modules into image layer - exported to registry cache,
# reused on all runners when go.mod/go.sum are unchanged
ENV GOCACHE=/root/.cache/go-build
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/
COPY apps/vps-service/go.mod apps/vps-service/go.sum ./apps/vps-service/
COPY apps/orchestrator-service/go.mod apps/orchestrator-service/go.sum ./apps/orchestrator-service/
RUN cd apps/shared && GOWORK=off go mod download && \
    cd /build/apps/vps-service && GOWORK=off go mod download && \
    cd /build/apps/orchestrator-service && GOWORK=off go mod download

# Build - modules already in image layer; only go-build cache is runner-local
RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    GOWORK=off go build -trimpath -ldflags="-w -s -extldflags '-static'" \
    -o /build/orchestrator-service /src/apps/orchestrator-service

# Final stage
FROM ${BASE_REGISTRY}/library/alpine:latest

# Install CA certificates, curl, netcat for health checks, and Docker CLI
# Docker CLI is needed for network operations and Traefik label updates
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk update && apk --no-cache --no-scripts add ca-certificates curl netcat-openbsd docker-cli

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/orchestrator-service .

# Expose port
EXPOSE 3007

# Run the service
CMD ["./orchestrator-service"]
