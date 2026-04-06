ARG BASE_REGISTRY=docker.io

FROM ${BASE_REGISTRY}/library/golang:1.25-alpine AS builder

# Accept build arguments for multi-arch support
ARG TARGETARCH=amd64
ARG TARGETOS=linux

WORKDIR /build

# Install build dependencies
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk add --no-cache --no-scripts git make

# Pre-download modules into image layer - exported to registry cache,
# reused on all runners when go.mod/go.sum are unchanged
ENV GOCACHE=/root/.cache/go-build
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/
COPY apps/vps-gateway/go.mod apps/vps-gateway/go.sum ./apps/vps-gateway/
RUN cd apps/shared && GOWORK=off go mod download && \
    cd /build/apps/vps-gateway && GOWORK=off go mod download

# Build service - modules in image layer; only go-build cache is runner-local
RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    GOWORK=off go build -trimpath -ldflags="-w -s -extldflags '-static'" \
    -o /build/apps/vps-gateway/vps-gateway /src/apps/vps-gateway/main.go

# Build grpcurl (baked into layer - exported to registry cache)
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Final stage
FROM ${BASE_REGISTRY}/library/alpine:latest

# Install dnsmasq and required tools
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk update && apk add --no-cache --no-scripts \
    dnsmasq \
    ca-certificates \
    tzdata \
    curl \
    bash

WORKDIR /app

# Copy binary and grpcurl from builder
COPY --from=builder /build/apps/vps-gateway/vps-gateway .
COPY --from=builder /root/go/bin/grpcurl /usr/local/bin/grpcurl

# Create directory for dnsmasq files
RUN mkdir -p /var/lib/obiente/vps-gateway

# Expose ports
EXPOSE 1537 9091

# Run the service
CMD ["./vps-gateway"]

