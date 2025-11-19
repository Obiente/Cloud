# Build stage for Orchestrator Service
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Enable Go build cache and module cache
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/go/pkg/mod

# Copy go mod files for shared and service
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/
COPY apps/orchestrator-service/go.mod apps/orchestrator-service/go.sum ./apps/orchestrator-service/

# Download dependencies for shared module
WORKDIR /build/apps/shared
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Download dependencies for service module
WORKDIR /build/apps/orchestrator-service
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy all source code
WORKDIR /build
COPY apps/shared/ ./apps/shared/
COPY apps/orchestrator-service/ ./apps/orchestrator-service/

# Build the service from its own module directory
WORKDIR /build/apps/orchestrator-service
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -o /build/orchestrator-service .

# Final stage
FROM alpine:latest

# Install CA certificates, curl, netcat for health checks, and Docker CLI
# Docker CLI is needed for network operations and Traefik label updates
RUN apk --no-cache add ca-certificates curl netcat-openbsd docker-cli

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/orchestrator-service .

# Expose port
EXPOSE 3007

# Run the service
CMD ["./orchestrator-service"]
