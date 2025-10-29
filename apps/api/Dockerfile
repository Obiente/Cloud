# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go workspace files first
COPY go.work go.work.sum ./

# Copy go mod files for API
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
WORKDIR /build/apps/api
RUN go mod download

# Copy all source code
WORKDIR /build
COPY . .

# Build the application
WORKDIR /build/apps/api
RUN go build -o /build/api ./main.go

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/api .

# Expose port
EXPOSE 3001

# Run the application
CMD ["./api"]

