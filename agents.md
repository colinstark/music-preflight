# agents.md — working on coverfixer

Guidance for AI agents (and humans) contributing to this repo.

## What this is

`coverfixer` batch-fixes cover art in a music library. It is the Go successor to
`rockbox_covers.sh`. Operations:

- Resize artwork **embedded inside** MP3/M4A files, in place.
- Generate `cover.jpg` in an album folder from embedded art.
- Resize existing `cover.jpg` (and rename a stray `*.jpg` to `cover.jpg`).
- Optionally **transcode** audio to `mp3-320` or `aac-256`.

Design doc: [`plan.md`](plan.md).

## Architecture (keep this intact)

The engine lives in `internal/core` and is **front-end-agnostic** — it must not
import any CLI or GUI package. Front-ends drive it through one entry point:

```go
core.Run(ctx, core.Options{...}, func(core.Event){ ... }) (core.Report, error)
```

- `cmd/coverfixer` — the CLI (stdlib `flag`), one of two front-ends.
- `cmd/coverfixer-gui` + `internal/ui` — the Wails desktop front-end. `internal/ui`
  owns the run lifecycle (single-flight, cancel) and Event/Report formatting and
  **must not import any Wails/GUI type** (so it stays headless-testable). The thin
  Wails adapter lives in `cmd/coverfixer-gui/app.go`; its bound methods delegate to
  `internal/ui`.

When adding behaviour, put logic in `internal/core` and surface it as an
`Options` field + an `Event`/`Report` counter, not in the CLI.

### Engine split (important)
- **Pure Go**, no external deps, for all artwork work:
  - images: `image/jpeg` + `golang.org/x/image/draw` (`image.go`) — baseline JPEG.
  - MP3 art: `github.com/bogem/id3v2/v2` (`mp3.go`).
  - M4A art: `github.com/Sorrow446/go-mp4tag` (`m4a.go`).
- **ffmpeg** is used **only** for transcoding (`transcode.go`). Artwork-only runs
  never touch it.

## ffmpeg delivery

ffmpeg is resolved lazily by `internal/ffmpeg` in two build variants:

- **Default build** (`go build`): `system.go` finds `ffmpeg` on `PATH`. Tests and
  normal dev builds never need the large static binary. Transcoding works if the
  user has ffmpeg installed.
- **Release build** (`go build -tags embed_ffmpeg`): a static ffmpeg is compiled
  into the binary via `//go:embed` (`embed.go` + `embed_<goos>_<goarch>.go`) and
  extracted to `os.UserCacheDir()/coverfixer/` on first transcode. This is the
  self-contained single-file distribution.

Build a release:

```sh
make fetch-ffmpeg     # downloads a static ffmpeg into internal/ffmpeg/bin/ (gitignored)
make release          # go build -tags embed_ffmpeg → dist/coverfixer-<os>-<arch>
make release-gui      # wails build -tags embed_ffmpeg → cmd/coverfixer-gui/build/bin/
```

Bump `ffmpegVersion` in `internal/ffmpeg/version.go` whenever you refresh the
bundled binary so cached copies are invalidated.

> **License note:** the static ffmpeg builds include libmp3lame and are GPL. A
> binary built with `-tags embed_ffmpeg` is therefore covered by the GPL. The
> default (non-embedded) build has no such constraint.

## Build / run / test

```sh
make build                       # dev CLI binary (system ffmpeg)
make run DIR=/path ARGS=--dry-run
make build-gui                   # GUI app via Wails (needs the `wails` CLI)
make gui-dev                     # GUI live-reload dev session
make check                       # the gate: fmt-check + vet + lint + test
go test ./...
```

`make check` is the green bar before declaring work done. `lint` runs
`golangci-lint` if installed and is skipped otherwise. The Go gate needs **no
node and no Wails CLI** — the GUI frontend is committed static files, and
`internal/ui` is tested headless.

### Front-end conventions (Wails GUI)
- The GUI is vanilla HTML/CSS/JS served from `cmd/coverfixer-gui/frontend/dist/`
  — there is **no bundler and no node build step**. Edit those files directly.
- All run lifecycle and formatting logic lives in `internal/ui` (testable,
  Wails-free). The bound methods in `cmd/coverfixer-gui/app.go` are a thin
  adapter; do not put engine logic there.
- Engine → frontend streaming uses `runtime.EventsEmit` over four events:
  `cf:progress` (a formatted log line), `cf:done` (summary), `cf:error` (msg),
  `cf:state` (bool running). Keep these names stable; they are part of the
  `internal/ui` contract (`emitter.go`).
- Regenerate TypeScript bindings after changing a bound method or `RunRequest`:
  run `wails build` (or `wails dev`) once; the `frontend/wailsjs/` output is
  gitignored. The committed `app.js` calls the injected `window.go.main.App` /
  `window.runtime` globals directly, so it does not import the generated files.

### Tests
- Pure-Go tests synthesise fixtures in memory (`testhelpers_test.go`) — no binary
  test assets are committed.
- `internal/ui` tests exercise the run lifecycle (single-flight, cancel, event
  streaming) with a fake emitter + fake engine; they import no Wails type and run
  headless as part of the standard Go gate.
- ffmpeg-dependent tests (`ffmpeg_integration_test.go`) **self-skip** when ffmpeg
  is not on `PATH`, so the suite stays green everywhere.

## Conventions
- In-place edits; `--backup` writes a one-time `<file>.bak` before mutating.
- `--dry-run` must mutate nothing while still reporting intended actions and
  counters — preserve this when adding passes.
- Default artwork target: fit within **500×500**, baseline JPEG quality **85**,
  never upscaled.
- Skip work that is already correct (within size + baseline) rather than
  re-encoding blindly.

## Roadmap
1. ✅ Core engine + CLI.
2. ✅ Wails desktop front-end (`cmd/coverfixer-gui` + `internal/ui`) on the same `core.Run`.
3. ⬜ Optional: custom minimal ffmpeg build to shrink the embedded binary.
