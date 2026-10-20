#!/bin/sh
set -eu
cd "$(dirname "$0")"
go test ./...
go build -o public-safety .
./public-safety "$@"
