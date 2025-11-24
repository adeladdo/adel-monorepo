.PHONY: build run test test-unit test-integration test-all test-race clean


# in Makefile
GO          ?= go
BUILD_DIR   ?= bin
MAIN_PKG    ?= ./app/cmd


START_URL   ?= https://crawlme.monzo.com/
MAX_DEPTH   ?= 0
WORKERS     ?= 10
HTTP_TIMEOUT ?= 10s
TASK_QUEUE_BUFFER ?= 100

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/web-crawler $(MAIN_PKG)

run:
	START_URL=$(START_URL) \
	MAX_DEPTH=$(MAX_DEPTH) \
	WORKERS=$(WORKERS) \
	HTTP_TIMEOUT=$(HTTP_TIMEOUT) \
	TASK_QUEUE_BUFFER=$(TASK_QUEUE_BUFFER) \
	$(GO) run $(MAIN_PKG)

run-stress:
	START_URL=$(START_URL) \
	MAX_DEPTH=0 \
	WORKERS=50 \
	TASK_QUEUE_BUFFER=1000 \
	go run ./app/cmd

test-unit:
	$(GO) test ./app/internal/... -short -v

test-integration:
	$(GO) test ./app/test/integration/... -v

test-all:
	$(GO) test ./... -v

test-coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

test-race:
	$(GO) test -race ./... -v

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html
