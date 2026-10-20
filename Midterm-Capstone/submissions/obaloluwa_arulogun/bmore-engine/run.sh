# Baltimore City Multi-Paradigmatic Data Processing Engine

An concurrent data ingestion, spatial-join, and municipal analytics engine written in Go. Ingests and cross-references Open Baltimore CSV exports (**311 Service Requests** and **Vacant Building Notices**).

## Quickstart (macOS / Linux)
```bash
chmod +x run.sh
./run.sh
@'
#!/usr/bin/env bash
set -euo pipefail

BINARY_NAME="bmore-engine"
BUILD_DIR="./bin"

echo "[1/3] Compiling Multi-Paradigmatic Data Processing Engine..."
mkdir -p "${BUILD_DIR}"
go build -ldflags="-s -w" -o "${BUILD_DIR}/${BINARY_NAME}" ./cmd/bmore-engine

echo "[2/3] Verifying dataset integrity in ./data..."
"${BUILD_DIR}/${BINARY_NAME}" "$@"

echo "[3/3] Execution complete."
