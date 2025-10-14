#!/usr/bin/env bash
set -euo pipefail

export PATH="$(go env GOPATH)/bin:$PATH"

cd "$(dirname "$0")/.."

echo "[format] running go fmt ./..."
go fmt ./...
