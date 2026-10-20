# Verification Record

Development checks were performed October 15, 2026 Eastern, using Go 1.27.2 on Linux AMD64. The user subsequently reported successful native macOS ARM64 verification and execution, as distinguished from the independently executed development checks below.

## Passed checks

- **Unit tests:** Model normalization, identities, date boundaries, closure eligibility, duplicate handling, invalid input, worker-count consistency, join behavior, cohorts, tied percentiles, medians, deterministic sorting, report sections and output errors.
- **Downloader tests:** Mock HTTP pagination, raw-file hashing, duplicate source-ID rejection, truncated-download rejection, and removal of incomplete files.
- **Concurrency stress fixture:** 10,000 records processed without deadlock or count loss.
- **Static checks:** `go vet ./...`.
- **Build:** `go build ./...`.
- **Race detection:** `go test -race -count=1 ./...`.
- **Full-data race detection:** Analyzer executed under the race detector with eight workers and the entire bundled extracts.
- **Determinism:** One-, four-, and eight-worker reports and CSVs compared byte for byte.
- **Input integrity:** SHA-256 checksums of both JSONL files verified.
- **Completeness checks:** Raw download counts matched source counts both before and after each download; duplicate source object IDs were rejected.
- **Independent numerical check:** A separate JavaScript calculation read the raw files and matched every exported neighborhood's request count, vacancy count, match status, cohort, and priority score.
- **macOS cross-compilation:** Analyzer built with `CGO_ENABLED=0` for `darwin/arm64` and `darwin/amd64`.

## User-reported native Mac verification

The user provided `go version go1.27.2 darwin/arm64`, reported that `sh verify.sh` completed without errors, and then ran `sh run.sh -workers 4`. They pasted the final neighborhood rows of the generated report and confirmed that `output/report.md` and `output/neighborhoods.csv` were present after the file-check command.

This records the user's confirmation, not an independently captured full Terminal log. Native Intel Mac execution has not been tested.

## What these checks do not prove

Tests do not establish that city data is complete or that the screening score is an appropriate municipal policy. The report documents missing neighborhoods, temporal mismatch, unadjusted counts, and the distinction between closure and resolution.

Cross-compilation alone does not demonstrate that the executable was run on a Mac; the separate user-reported ARM64 run is documented above. Rerun verification after any source or data changes.

The independent checking script was a development-only validation tool, not a runtime dependency. The delivered application, downloader, and tests are Go, with POSIX shell convenience scripts.
