#!/bin/sh
set -eu
cd "$(dirname "$0")"
go test ./...
go vet ./...
go build ./...
# macOS includes shasum; sha256sum is commonly available on Linux.
if command -v shasum >/dev/null 2>&1; then
  shasum -a 256 -c data/SHA256SUMS
else
  sha256sum -c data/SHA256SUMS
fi
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
go run ./cmd/analyzer -workers 1 -out "$tmp/one" > /dev/null
go run ./cmd/analyzer -workers 4 -out "$tmp/four" > /dev/null
cmp "$tmp/one/report.md" "$tmp/four/report.md"
cmp "$tmp/one/neighborhoods.csv" "$tmp/four/neighborhoods.csv"
cmp output/report.md "$tmp/one/report.md"
cmp output/neighborhoods.csv "$tmp/one/neighborhoods.csv"
echo "Verified tests, vet, build, raw-data checksums, worker-count determinism, and bundled output."
