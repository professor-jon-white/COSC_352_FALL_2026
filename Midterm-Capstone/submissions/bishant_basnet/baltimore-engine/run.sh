#!/usr/bin/env bash
# Build and run the Baltimore Multi-Paradigm Data Processing Engine on macOS.
# Usage: ./run.sh [extra engine flags, e.g. -workers 8 -top 15]
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed. Install it with:  brew install go   (or from https://go.dev/dl/)"
  exit 1
fi

echo "==> Building ($(go version))"
mkdir -p bin
go build -o bin/baltimore-engine ./cmd/engine
go build -o bin/gensample ./cmd/gensample

CRIME="${CRIME:-data/crime.csv}"
REQUESTS="${REQUESTS:-data/311.csv}"

if [[ ! -f "$CRIME" || ! -f "$REQUESTS" ]]; then
  echo "==> Real Open Baltimore files not found at $CRIME and $REQUESTS"
  echo "    Generating a SYNTHETIC sample so the engine can be verified (see README)."
  ./bin/gensample -out data/sample
  CRIME=data/sample/crime_sample.csv
  REQUESTS=data/sample/311_sample.csv
fi

echo "==> Running engine on $CRIME + $REQUESTS"
./bin/baltimore-engine -crime "$CRIME" -requests "$REQUESTS" -out report.md "$@"
