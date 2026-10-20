#!/bin/sh
set -eu
cd "$(dirname "$0")"
command -v go >/dev/null 2>&1 || { echo "Install Go 1.22 or newer from https://go.dev/dl/ and reopen Terminal." >&2; exit 1; }
mkdir -p bin
go build -trimpath -o bin/analyzer ./cmd/analyzer
exec ./bin/analyzer "$@"
