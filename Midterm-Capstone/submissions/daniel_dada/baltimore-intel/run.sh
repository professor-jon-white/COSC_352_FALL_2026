#!/usr/bin/env bash
# Build, test, and run Baltimore Intel on macOS.
# Usage: bash run.sh
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed. Run: brew install go" >&2
  exit 1
fi
go version

echo "== Build =="
go build -o baltimore-intel ./cmd

echo "== Tests (with race detector) =="
go test -race ./...

echo "== Run (uses data/ if present, else the bundled sample) =="
./baltimore-intel

echo "== Verify parallel join equals sequential join =="
./baltimore-intel -verify | grep VERIFY

echo "Done. The full report is in report.txt"