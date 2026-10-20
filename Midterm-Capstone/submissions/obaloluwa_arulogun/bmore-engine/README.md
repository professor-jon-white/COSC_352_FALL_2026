# Baltimore City Multi-Paradigmatic Data Processing Engine

An concurrent data ingestion, spatial-join, and municipal analytics engine written in Go. Ingests and cross-references Open Baltimore CSV exports (**311 Service Requests** and **Vacant Building Notices**).

## Quickstart (macOS / Linux)
```bash
chmod +x run.sh
./run.sh
go run ./cmd/bmore-engine -seed
