BINARY_HARNESS=baltimore-engine
DATA_DIR=data

.PHONY: all build run clean setup mock

all: build run 

setup: 
	mkdir -p $(DATA_DIR)

mock: 
	go run scripts/gen_mock.go

build: 
	go build -o $(BINARY_HARNESS) . 

run: build 
	./$(BINARY_HARNESS)

clean: 
	rm -f $(BINARY_HARNESS)
