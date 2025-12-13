# Build stage for Vps Service
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

## Build using bind mounts so source changes invalidate cache correctly
# Make the bind mount writable and disable Go workspace mode to avoid writing go.work.sum
RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    sh -c "export GOWORK=off && \
           cd /src/apps/shared && go mod download && \
           cd /src/apps/vps-service && go mod download && \
           CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
           GOWORK=off go build -trimpath -ldflags=\"-w -s -extldflags '-static'\" \
           -o /build/vps-service /src/apps/vps-service"

# Final stage
FROM alpine:latest

# Install CA certificates, curl, and netcat for health checks
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk update && apk --no-cache --no-scripts add ca-certificates curl netcat-openbsd

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/vps-service .

# Expose port
EXPOSE 3008

# Run the service
CMD ["./vps-service"]
