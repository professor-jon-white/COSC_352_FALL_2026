# Baltimore Cross-Dataset Intelligence — build & run targets (macOS / Linux)
BIN     := bin/baltimore-intel
DATA    ?= data/raw
OUT     ?= reports
YEAR    ?= 2022
WORKERS ?= $(shell sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)

.PHONY: help build demo sample fetch run run-ascii test race vet bench open clean

help:
	@echo "make build      compile to $(BIN)"
	@echo "make demo       build + generate sample data + run all 5 analyses (no network needed)"
	@echo "make fetch      download live Open Baltimore data for YEAR=$(YEAR) into $(DATA)"
	@echo "make run        analyze whatever is in $(DATA) (YEAR=$(YEAR), WORKERS=$(WORKERS))"
	@echo "make test       unit + end-to-end tests      make race   tests under the race detector"
	@echo "make bench      1 vs N worker timing          make open   open the HTML report"

build:
	go build -o $(BIN) ./cmd/baltimore-intel

sample: build
	./$(BIN) sample -data $(DATA) -year $(YEAR)

fetch: build
	./$(BIN) fetch -data $(DATA) -year $(YEAR)

run: build
	./$(BIN) analyze -data $(DATA) -out $(OUT) -year $(YEAR) -workers $(WORKERS)

run-ascii: build
	./$(BIN) analyze -data $(DATA) -out $(OUT) -year $(YEAR) -workers $(WORKERS) -ascii

demo: sample run

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

bench: build
	./$(BIN) bench -data $(DATA) -workers $(WORKERS)

open:
	open $(OUT)/report.html 2>/dev/null || xdg-open $(OUT)/report.html

clean:
	rm -rf bin $(OUT)
