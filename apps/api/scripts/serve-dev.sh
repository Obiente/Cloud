#!/usr/bin/env bash


# Always add Go bin to PATH so Nx can find installed tools
export PATH="$(go env GOPATH)/bin:$PATH"

echo "[serve-dev] starting in $(pwd)"

# Ensure Air is installed
if ! command -v air >/dev/null 2>&1; then
  echo "[serve-dev] Installing Air hot-reload tool..."
  go install github.com/air-verse/air@latest
fi

# Run Air without exec so Nx can capture logs
# Add -v for verbose logging
exec air -c .air.toml -v