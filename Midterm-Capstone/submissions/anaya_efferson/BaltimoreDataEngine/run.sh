#!/bin/sh
set -eu

cd "$(dirname "$0")"

echo "Building Baltimore City Data Analysis Engine..."
go build -o baltimore-engine .

echo "Running Baltimore City Data Analysis Engine..."
./baltimore-engine