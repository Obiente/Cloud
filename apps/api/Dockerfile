# Build stage for Go API
FROM golang:1.25-alpine AS go-builder

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

# Build stage for nixpacks
FROM rust:1.75-alpine AS nixpacks-builder

# Install build dependencies
RUN apk add --no-cache musl-dev

# Install nixpacks via cargo
RUN cargo install nixpacks --locked

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS requests
# Install docker CLI and docker-compose for deployment management
# Install git for repository cloning during builds
RUN apk --no-cache add \
    ca-certificates \
    docker-cli \
    docker-compose \
    git

# Copy nixpacks binary from builder
COPY --from=nixpacks-builder /usr/local/cargo/bin/nixpacks /usr/local/bin/nixpacks

WORKDIR /app

# Copy API binary from Go builder
COPY --from=go-builder /build/api .

# Expose port
EXPOSE 3001

# Run the application
CMD ["./api"]

