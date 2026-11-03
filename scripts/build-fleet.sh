#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:-dev}"
mkdir -p dist
for role in control node; do
  CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${version}" -o "dist/${role}" "./cmd/${role}"
done
(cd dist && sha256sum control node > fleet-sha256sums.txt)
