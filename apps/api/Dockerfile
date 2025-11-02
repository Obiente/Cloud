# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Enable Go build cache and module cache
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/go/pkg/mod

# Copy go workspace files first (for better layer caching)
COPY go.work go.work.sum ./

# Copy go mod files for API (copy separately to improve cache hits)
COPY apps/api/go.mod apps/api/go.sum ./apps/api/

# Download dependencies (this layer will be cached unless go.mod/go.sum change)
WORKDIR /build/apps/api
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy only the API source code (proto files are already generated inside apps/api)
WORKDIR /build
COPY apps/api/ ./apps/api/

# Build the application with optimizations
WORKDIR /build/apps/api
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -extldflags '-static'" \
    -o /build/api ./main.go

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS requests
# Install docker CLI and docker-compose for deployment management
RUN apk --no-cache add \
    ca-certificates \
    docker-cli \
    docker-compose

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/api .

# Expose port
EXPOSE 3001

# Run the application
CMD ["./api"]

