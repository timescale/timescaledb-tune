.PHONY: build clean test

BINARY_NAME=timescaledb-tune
BUILD_DIR=build

build:
	@mkdir -p $(BUILD_DIR)
	go build -v -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/timescaledb-tune/

clean:
	@rm -rf $(BUILD_DIR)

test:
	go test -v ./...
