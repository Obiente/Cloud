# Build stage for Superadmin Service
FROM golang:1.25-alpine AS go-builder

# Accept build arguments for multi-arch support
ARG TARGETARCH
ARG TARGETOS=linux

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Enable Go build cache and module cache
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/go/pkg/mod

# Copy go mod files for shared, vps-service, and service
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/
COPY apps/vps-service/go.mod apps/vps-service/go.sum ./apps/vps-service/
COPY apps/superadmin-service/go.mod apps/superadmin-service/go.sum ./apps/superadmin-service/

# Download dependencies for shared module
WORKDIR /build/apps/shared
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Download dependencies for vps-service module
WORKDIR /build/apps/vps-service
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Download dependencies for service module
WORKDIR /build/apps/superadmin-service
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy all source code
WORKDIR /build
COPY apps/shared/ ./apps/shared/
COPY apps/vps-service/ ./apps/vps-service/
COPY apps/superadmin-service/ ./apps/superadmin-service/

# Build the service from its own module directory
WORKDIR /build/apps/superadmin-service
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -o /build/superadmin-service .

# Final stage
FROM alpine:latest

# Install CA certificates, curl, and netcat for health checks
RUN apk --no-cache add ca-certificates curl netcat-openbsd

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/superadmin-service .

# Expose port
EXPOSE 3011

# Run the service
CMD ["./superadmin-service"]
