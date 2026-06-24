# coverfixer

`coverfixer` batch-fixes cover art in a music library. It is the Go successor to
`rockbox_covers.sh`. In a single pass over a folder tree it can:

- Resize artwork **embedded inside** MP3/M4A files, in place.
- Generate `cover.jpg` in an album folder from embedded art when one is missing.
- Resize an existing `cover.jpg` (and rename a stray `*.jpg` to `cover.jpg`).
- Optionally **transcode** audio to `mp3-320` or `aac-256`.
- Optionally **set the genre tag** on every audio file to a single value.

All artwork work is **pure Go** (no external dependencies). `ffmpeg` is used **only**
for transcoding, so artwork-only runs never touch it.

By default, artwork is fit within **500×500**, encoded as **baseline JPEG at quality 85**,
and **never upscaled**. Edits are made in place; pass `--backup` to keep a one-time
`<file>.bak` copy, or `--dry-run` to report intended actions without changing anything.

## Architecture

The engine lives in `internal/core` and is **front-end-agnostic** — it does not import
any CLI or GUI package. Both front-ends drive it through one entry point:

```go
core.Run(ctx, core.Options{...}, func(core.Event){ ... }) (core.Report, error)
```

- `cmd/coverfixer` — the command-line front-end (stdlib `flag`).
- `cmd/coverfixer-gui` + `internal/ui` — a native desktop front-end built with
  [Wails](https://wails.io) (HTML/CSS/JS UI over a Go backend). `internal/ui`
  owns the run lifecycle and formatting; it depends on no GUI or Wails type, so
  it is fully unit-testable headless.

## CLI

```sh
make build                                   # builds ./coverfixer (uses system ffmpeg)
./coverfixer [flags] <music-folder>
```

| Flag                 | Default | Description                                                  |
| -------------------- | ------- | ------------------------------------------------------------ |
| `--size`             | `500`   | Max artwork dimension; images fit within size×size.          |
| `--quality`          | `85`    | Baseline JPEG quality (1-100).                               |
| `--rename-jpg`       | `true`  | Rename a lone non-cover `*.jpg` to `cover.jpg`.              |
| `--resize-covers`    | `true`  | Resize existing `cover.jpg` files.                          |
| `--extract-cover`    | `true`  | Write `cover.jpg` from embedded art when missing.           |
| `--resize-embedded`  | `false` | Resize artwork embedded inside audio files, in place.        |
| `--transcode`        | `none`  | Audio conversion: `none` \| `mp3-320` \| `aac-256`.          |
| `--genre`            | _(off)_ | Set the genre tag on every audio file to this value (empty = off). |
| `--backup`           | `false` | Write a `<file>.bak` copy before modifying a file.          |
| `--dry-run`          | `false` | Report intended actions without changing anything.          |
| `--no-recursive`     | `false` | Do not descend into subfolders.                             |

Example:

```sh
# Preview a full artwork + transcode pass without touching any files
./coverfixer --resize-embedded --transcode mp3-320 --dry-run ~/Music

# Or via the Makefile
make run DIR=~/Music ARGS="--dry-run"
```

On completion the CLI prints a summary of counters (renamed, covers resized, extracted,
embedded resized, transcoded, skipped, failed).

## GUI

`cmd/coverfixer-gui` is a native desktop app built with
[Wails](https://wails.io) (v2): the UI is plain HTML/CSS/JS served from
`frontend/dist/`, and the backend is Go driving the same `core.Run` engine. It
exposes full `core.Options` parity: a folder picker, toggles for each pass
(recursive, rename stray jpg, resize `cover.jpg`, extract cover, resize
embedded), art-size and JPEG-quality entries, a transcode dropdown
(`none` / `mp3-320` / `aac-256`), and a backup toggle. It shows a live progress
log, a summary of result counters, and Run / Cancel buttons.

For safety, the GUI **defaults to dry-run** — untick the Dry-run checkbox to apply changes.

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest   # one-time Wails CLI install
make build-gui                                             # → cmd/coverfixer-gui/build/bin/
make gui-dev                                               # live-reload dev session
```

The GUI frontend is **Svelte 5 + Vite** (bun-managed), built from
`cmd/coverfixer-gui/frontend/src/` into the committed
`cmd/coverfixer-gui/frontend/dist/` bundle that Wails embeds. `make check`
(Go fmt/vet/lint/test) needs neither bun nor the Wails CLI — the built bundle
is checked in. The Wails CLI and bun are only required for `gui-dev`,
`build-gui`, `release-gui`, and `make gui-frontend` (the standalone bundle
rebuild).

Wails v2 renders via the OS webview, so **CGO is not required** on macOS
(WKWebView) or Windows (WebView2). Building the GUI still needs the platform
toolchain Wails links against:

- **macOS:** Xcode Command Line Tools (`xcode-select --install`).
- **Linux:** `libwebkit2gtk-4.1-dev` and `libgtk-3-dev` (replaces Fyne's X11/OpenGL headers).
- **Windows:** WebView2 runtime (built into Windows 11; a small runtime on Windows 10).

## Build / run / test

```sh
make build                       # dev CLI binary (uses system ffmpeg for transcode)
make run DIR=/path/to/music ARGS="--dry-run"
make build-gui                   # GUI app (Wails; requires the wails CLI)
make check                       # the project gate: fmt-check + vet + lint + test
go test ./...
```

`make check` is the green bar before declaring work done. `lint` runs `golangci-lint`
if it is installed and is skipped otherwise. ffmpeg-dependent tests self-skip when
ffmpeg is not on `PATH`, so the suite stays green everywhere.

## Transcoding and ffmpeg

Transcoding requires ffmpeg, which is resolved in two build variants:

- **Default build** (`make build` / `go build`): ffmpeg is found on `PATH`. Transcoding
  works if the user has ffmpeg installed; artwork-only runs need nothing extra.
- **Release build** (`-tags embed_ffmpeg`): a static ffmpeg is embedded into the binary
  and extracted to the user cache dir on first transcode, producing a self-contained
  single-file distribution.

```sh
make fetch-ffmpeg     # downloads a static ffmpeg into internal/ffmpeg/bin/ (gitignored)
make release          # go build -tags embed_ffmpeg → dist/coverfixer-<os>-<arch>
make release-gui      # wails build -tags embed_ffmpeg → cmd/coverfixer-gui/build/bin/
```

> **License note:** the static ffmpeg builds include libmp3lame and are GPL. A binary
> built with `-tags embed_ffmpeg` is therefore covered by the GPL. The default
> (non-embedded) build has no such constraint.

See [`plan.md`](plan.md) for the full design and [`AGENTS.md`](AGENTS.md) for contributor
guidance.
