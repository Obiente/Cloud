FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files (build context is repository root)
COPY apps/vps-gateway/go.mod apps/vps-gateway/go.sum ./apps/vps-gateway/

# Download dependencies (this layer will be cached unless go.mod/go.sum change)
WORKDIR /build/apps/vps-gateway
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
WORKDIR /build
COPY apps/vps-gateway/ ./apps/vps-gateway/

# Build the application
WORKDIR /build/apps/vps-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vps-gateway ./main.go

# Final stage
FROM alpine:latest

# Install dnsmasq and required tools
RUN apk add --no-cache \
    dnsmasq \
    ca-certificates \
    tzdata \
    curl \
    bash \
    go \
    git

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/vps-gateway .

# Install grpcurl for testing
RUN go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest && \
    echo 'export PATH=$PATH:/root/go/bin' >> /root/.profile && \
    echo 'export PATH=$PATH:/root/go/bin' >> /etc/profile

# Create directory for dnsmasq files
RUN mkdir -p /var/lib/vps-gateway

# Expose ports
EXPOSE 8080 9091

# Run the service
CMD ["./vps-gateway"]

