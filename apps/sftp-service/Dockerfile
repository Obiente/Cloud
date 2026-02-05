# Build stage for SFTP Service
FROM golang:1.25-alpine AS go-builder

# Accept build arguments for multi-arch support
ARG TARGETARCH=amd64
ARG TARGETOS=linux

# Install build dependencies
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk add --no-cache --no-scripts git make

# Set working directory
WORKDIR /build

# Enable Go build cache and module cache
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/go/pkg/mod

RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    sh -c "export GOWORK=off && \
           cd /src/apps/shared && go mod download && \
           cd /src/apps/sftp-service && go mod download && \
           CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
           GOWORK=off go build -trimpath -ldflags=\"-w -s -extldflags '-static'\" \
           -o /build/sftp-service /src/apps/sftp-service"

# Final stage
FROM alpine:latest

# Install CA certificates, curl, and netcat for health checks
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk update && apk --no-cache --no-scripts add ca-certificates curl netcat-openbsd

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/sftp-service .

# Create directory for SFTP files and host key
RUN mkdir -p /var/lib/sftp

# Expose SFTP port (2222) and HTTP health port (3020)
EXPOSE 2222 3020

# Run the service
CMD ["./sftp-service"]
