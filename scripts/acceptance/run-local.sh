#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
go test ./...
go vet ./...
node --check web/fleet.js
python_bin="${PYTHON:-python3}"
if ! "$python_bin" --version >/dev/null 2>&1; then
  [[ -z "${PYTHON:-}" ]] && python --version >/dev/null 2>&1 || { echo 'Python 3 is required' >&2; exit 1; }
  python_bin=python
fi
"$python_bin" -m py_compile scripts/acceptance/check-fleet.py scripts/acceptance/browser-smoke.py scripts/acceptance/local-live.py scripts/acceptance/systemd-live.py
if [[ "${UNIFIEDPROXY_FLEET_INTEGRATION:-0}" == 1 ]]; then
  command -v tar >/dev/null && command -v xz >/dev/null
  go test ./internal/app -run TestFleetRealSidecarProcesses -count=1 -v
fi
printf '%s\n' 'Local checks passed. Live servers require check-fleet.py and actual protocol clients.'
