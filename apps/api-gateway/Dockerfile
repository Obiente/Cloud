# Build stage for API Gateway
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
COPY apps/api-gateway/go.mod apps/api-gateway/go.sum ./apps/api-gateway/
RUN cd apps/shared && GOWORK=off go mod download && \
    cd /build/apps/api-gateway && GOWORK=off go mod download

# Build - modules already in image layer; only go-build cache is runner-local
RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    GOWORK=off go build -trimpath -ldflags="-w -s -extldflags '-static'" \
    -o /build/api-gateway /src/apps/api-gateway

# Final stage
FROM ${BASE_REGISTRY}/library/alpine:latest

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
