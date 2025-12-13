FROM golang:1.25-alpine AS builder

# Accept build arguments for multi-arch support
ARG TARGETARCH=amd64
ARG TARGETOS=linux

WORKDIR /build

# Install build dependencies
# Use --no-scripts to disable triggers and avoid QEMU emulation issues
RUN apk add --no-cache --no-scripts git make

RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    sh -c "export GOWORK=off && \
           cd /src/apps/shared && go mod download && \
           cd /src/apps/vps-gateway && go mod download && \
           CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
           GOWORK=off go build -a -installsuffix cgo \
           -o /build/apps/vps-gateway/vps-gateway /src/apps/vps-gateway/main.go"

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

