#!/usr/bin/env bash
# Build and run the Baltimore cross-dataset report.
# Usage: ./run.sh [flags]      (e.g. ./run.sh -data /path/to/csvs -workers 4)
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed. On macOS run: brew install go   (or download from https://go.dev/dl/)" >&2
  exit 1
fi

go build -o bin/baltimore-intel .
exec ./bin/baltimore-intel "$@"
