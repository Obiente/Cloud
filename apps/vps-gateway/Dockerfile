FROM golang:1.25-alpine AS builder

# Accept build arguments for multi-arch support
ARG TARGETARCH=amd64
ARG TARGETOS=linux

WORKDIR /build

# Install build dependencies
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk add --no-cache --no-scripts git make

# Copy go mod files for vps-gateway and shared (copy separately to improve cache hits)
COPY apps/vps-gateway/go.mod apps/vps-gateway/go.sum ./apps/vps-gateway/
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/

# Copy shared proto source code (needed for imports)
COPY apps/shared/proto ./apps/shared/proto/

# Download dependencies (this layer will be cached unless go.mod/go.sum change)
# The replace directive is already in go.mod
WORKDIR /build/apps/vps-gateway
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
WORKDIR /build
COPY apps/vps-gateway/ ./apps/vps-gateway/

# Build the application
WORKDIR /build/apps/vps-gateway
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -o vps-gateway ./main.go

# Final stage
FROM alpine:latest

# Install dnsmasq and required tools
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk update && apk add --no-cache --no-scripts \
    dnsmasq \
    ca-certificates \
    tzdata \
    curl \
    bash \
    go \
    git

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/apps/vps-gateway/vps-gateway .

# Install grpcurl for testing
RUN go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest && \
    echo 'export PATH=$PATH:/root/go/bin' >> /root/.profile && \
    echo 'export PATH=$PATH:/root/go/bin' >> /etc/profile

# Create directory for dnsmasq files
RUN mkdir -p /var/lib/obiente/vps-gateway

# Expose ports
EXPOSE 1537 9091

# Run the service
CMD ["./vps-gateway"]

