#!/usr/bin/env bash
# Runs the Baltimore Data Engine pipeline (ingest -> processor -> analytics -> ui).

set -euo pipefail

# Always run from the directory this script lives in, wherever it is called from.
cd "$(dirname "${BASH_SOURCE[0]}")"

echo "=============================================="
echo " Welcome to the Baltimore Data Engine"
echo " 311 Requests x Vacant Buildings Pipeline"
echo "=============================================="
echo

if ! command -v go >/dev/null 2>&1; then
  echo "Error: 'go' was not found on this machine." >&2
  echo "Install Go 1.27 or newer (brew install go, or https://go.dev/dl/) and try again." >&2
  exit 1
fi

echo "Using $(go version)"
echo "Fetching live data (takes about 30-40 seconds)..."
echo

go run main.go
