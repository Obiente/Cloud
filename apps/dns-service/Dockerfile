# Build stage for DNS Service
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

# Copy service main.go into api/cmd/dns-service so it's part of the api module
COPY apps/dns-service/main.go ./apps/api/cmd/dns-service/main.go

# Build the dns service from API module directory (to access internal packages)
WORKDIR /build/apps/api
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -o /build/dns-service ./cmd/dns-service/main.go

# Final stage
FROM alpine:latest

# Install CA certificates
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/dns-service .

# Expose DNS ports
EXPOSE 53/udp 53/tcp

# Run the service
CMD ["./dns-service"]

