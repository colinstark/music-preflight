# coverfixer — build, test, and release tasks.

BINARY      := coverfixer
CMD         := ./cmd/coverfixer
GUI_DIR     := ./cmd/coverfixer-gui
FFMPEG_DIR  := internal/ffmpeg/bin
GOOS        := $(shell go env GOOS)
GOARCH      := $(shell go env GOARCH)

.PHONY: check fmt fmt-check vet lint test build run release fetch-ffmpeg clean gui-dev gui-frontend build-gui release-gui

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

## gui-frontend: rebuild the committed GUI bundle (frontend/dist) with bun.
## Requires bun (https://bun.sh). No wails CLI needed. Run this after editing
## the Svelte sources so the embedded assets are up to date.
gui-frontend:
	cd $(GUI_DIR)/frontend && bun install && bun run build

## gui-dev: run the Wails GUI with live reload. Requires the `wails` CLI
## (go install github.com/wailsapp/wails/v2/cmd/wails@latest) and bun.
gui-dev:
	cd $(GUI_DIR) && wails dev

## build-gui: build the GUI app (uses system ffmpeg for transcode). Requires
## the `wails` CLI and bun.
build-gui:
	cd $(GUI_DIR) && wails build

## release-gui: build a self-contained GUI with ffmpeg embedded. Requires `make fetch-ffmpeg` first, plus the `wails` CLI.
release-gui: fetch-ffmpeg
	cd $(GUI_DIR) && wails build -tags embed_ffmpeg

## clean: remove build artifacts (keeps fetched ffmpeg binaries).
clean:
	rm -f $(BINARY)
	rm -rf dist
