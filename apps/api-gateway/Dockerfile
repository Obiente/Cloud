# Build stage for API Gateway
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
COPY apps/api-gateway/go.mod apps/api-gateway/go.sum ./apps/api-gateway/

# Download dependencies for shared module
WORKDIR /build/apps/shared
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Download dependencies for service module
WORKDIR /build/apps/api-gateway
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy all source code
WORKDIR /build
COPY apps/shared/ ./apps/shared/
COPY apps/api-gateway/ ./apps/api-gateway/

# Build the service from its own module directory
WORKDIR /build/apps/api-gateway
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -o /build/api-gateway .

# Final stage
FROM alpine:latest

# Install CA certificates, curl, and netcat for health checks
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk update && apk --no-cache --no-scripts add ca-certificates curl netcat-openbsd

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/api-gateway .

# Expose port
EXPOSE 3001

# Run the service
CMD ["./api-gateway"]
