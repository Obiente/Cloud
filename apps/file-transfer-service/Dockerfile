# Build stage for File Transfer Service
ARG BASE_REGISTRY=docker.io

FROM ${BASE_REGISTRY}/library/golang:1.25-alpine AS go-builder

ARG TARGETARCH=amd64
ARG TARGETOS=linux

RUN apk add --no-cache --no-scripts git make

WORKDIR /build

ENV GOCACHE=/root/.cache/go-build
COPY apps/shared/go.mod apps/shared/go.sum ./apps/shared/
COPY apps/file-transfer-service/go.mod apps/file-transfer-service/go.sum ./apps/file-transfer-service/
RUN cd apps/shared && GOWORK=off go mod download && \
    cd /build/apps/file-transfer-service && GOWORK=off go mod download

RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /src/apps/file-transfer-service && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    GOWORK=off go build -trimpath -ldflags="-w -s -extldflags '-static'" \
    -o /build/file-transfer-service .

FROM ${BASE_REGISTRY}/library/alpine:latest

RUN apk update && apk --no-cache --no-scripts add ca-certificates curl netcat-openbsd

WORKDIR /app

COPY --from=go-builder /build/file-transfer-service .

EXPOSE 2222 3022

CMD ["./file-transfer-service"]
