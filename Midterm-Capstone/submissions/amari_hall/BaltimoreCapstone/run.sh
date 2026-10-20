
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "Baltimore City Intelligence Engine"
echo "Checking Go installation..."

if ! command -v go >/dev/null 2>&1; then
    echo "ERROR: Go is not installed."
    exit 1
fi

echo "Running automated tests..."
go test -count=1 ./...

echo ""
echo "Building application..."
go build -o baltimore-engine .

echo ""
echo "Running Baltimore City Intelligence Engine..."
./baltimore-engine
