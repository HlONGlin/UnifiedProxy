#!/usr/bin/env bash
# Run only via orca-linux from its /workspace mount.
set -euo pipefail
[[ "$PWD" == /workspace && -f /.dockerenv ]] || { echo 'Run inside orca-linux at /workspace' >&2; exit 1; }
export GOROOT=/root/go GOPATH=/root/unifiedproxy-gopath
export PATH="$GOROOT/bin:$PATH"
stage="${1:?stage name required}"
[[ "$stage" =~ ^[a-zA-Z0-9_-]+$ ]] || exit 2
mkdir -p dist/design-audit
log="dist/design-audit/${stage}-go.log"
{
  printf 'stage=%s cwd=%s system=%s\n' "$stage" "$PWD" "$(uname -s)"
  go version
  go build ./...
  echo 'go build ./...: PASS'
  go vet ./...
  echo 'go vet ./...: PASS'
  go test ./...
  echo 'go test ./...: PASS'
} 2>&1 | tee "$log"
