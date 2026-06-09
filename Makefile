# coverfixer — build, test, and release tasks.

BINARY      := coverfixer
CMD         := ./cmd/coverfixer
FFMPEG_DIR  := internal/ffmpeg/bin
GOOS        := $(shell go env GOOS)
GOARCH      := $(shell go env GOARCH)

.PHONY: check fmt fmt-check vet lint test build run release fetch-ffmpeg clean

## check: the project gate — formatting, vet, lint, and tests must all pass.
check: fmt-check vet lint test

## fmt: format all Go code.
fmt:
	gofmt -w .

## fmt-check: fail if any file is not gofmt-clean.
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
	fi

## vet: run go vet.
vet:
	go vet ./...

## lint: run golangci-lint if installed (no-op with a note otherwise).
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed; skipping (install: https://golangci-lint.run)"; \
	fi

## test: run the test suite. ffmpeg integration tests self-skip without ffmpeg.
test:
	go test ./...

## build: development build (uses system ffmpeg for transcoding).
build:
	go build -o $(BINARY) $(CMD)

## run: build and run against DIR (make run DIR=/path/to/music ARGS="--dry-run").
run: build
	./$(BINARY) $(ARGS) $(DIR)

## release: self-contained build with ffmpeg embedded. Requires `make fetch-ffmpeg` first.
release: fetch-ffmpeg
	go build -tags embed_ffmpeg -o dist/$(BINARY)-$(GOOS)-$(GOARCH) $(CMD)

## fetch-ffmpeg: download a static ffmpeg for this host into internal/ffmpeg/bin.
fetch-ffmpeg:
	@mkdir -p $(FFMPEG_DIR)
	@./scripts/fetch-ffmpeg.sh $(GOOS) $(GOARCH) $(FFMPEG_DIR)

## clean: remove build artifacts (keeps fetched ffmpeg binaries).
clean:
	rm -f $(BINARY)
	rm -rf dist
