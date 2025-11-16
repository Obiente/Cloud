# Build stage for VPS Service
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Enable Go build cache and module cache
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/go/pkg/mod

# Copy go mod files
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/

# Download dependencies for API module (which contains internal packages)
WORKDIR /build/apps/api
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy all source code
WORKDIR /build
COPY apps/api/ ./apps/api/
COPY apps/shared/ ./apps/shared/

# Copy service main.go into api/cmd/vps-service so it's part of the api module
COPY apps/vps-service/main.go ./apps/api/cmd/vps-service/main.go

# Build the vps service from API module directory (to access internal packages)
WORKDIR /build/apps/api
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -o /build/vps-service ./cmd/vps-service/main.go

# Final stage
FROM alpine:latest

# Install CA certificates and curl for health checks
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/vps-service .

# Expose port
EXPOSE 3008

# Run the service
CMD ["./vps-service"]

