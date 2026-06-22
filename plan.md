# Coverfixer — Go cover-art batch utility

## Context

The user maintains a music library (Rockbox-oriented) and currently fixes cover art with
`rockbox_covers.sh`, which uses `vips` + `ffmpeg` to: rename stray `*.jpg` → `cover.jpg`,
resize `cover.jpg` to 360×360 baseline JPEG, and extract embedded art to `cover.jpg` when
missing. The user wants this reimplemented as a **Go application** that does more:

- Resize cover art **embedded inside** mp3/AAC(m4a) files (and re-embed it).
- Generate `cover.jpg` in each album folder from embedded art.
- Resize existing `cover.jpg` files (script parity).
- Optionally **transcode** audio to **mp3 320k** or **AAC 256k**.
- Configurable artwork size; a boolean to extract cover art or not.

Goal: a clean **core library** first, then a **CLI**, then a **Fyne GUI** later. The end
product must ship as a **single self-contained binary** — users must not have to install
ffmpeg.

### Locked decisions (from user)
- **Engine:** Pure Go for everything possible; ffmpeg only for transcode.
  - Image resize → pure Go (`image/jpeg` + `golang.org/x/image/draw`), baseline JPEG.
  - MP3 embedded art → pure Go (`bogem/id3v2`).
  - M4A/AAC embedded art → pure Go (`Sorrow446/go-mp4tag`, fallback `abema/go-mp4`).
  - Transcode (mp3-320 / aac-256) → **embedded ffmpeg** (only real need).
- **ffmpeg delivery:** `//go:embed` a static ffmpeg into the binary; extract to the user
  cache dir on first transcode. Single self-contained file. (Accept ~70–100 MB/platform
  size and GPL implication from libmp3lame.)
- **Output mode:** modify **in-place**, with optional `--backup` sidecar. `cover.jpg` is
  always written into the album folder.
- **v1 scope:** full core now (embedded resize + extract + cover.jpg resize + transcode),
  then CLI. GUI is a later milestone.
- **Art default:** fit within **500×500** preserving aspect, **baseline** JPEG quality 85,
  no upscaling. Size/quality overridable by flags.

## Architecture

Keep the core decoupled from CLI/GUI so both front-ends call the same library.

```
coverfixer/
  go.mod                      # module path e.g. github.com/colinstark/coverfixer
  plan.md                     # copy of this plan (deliverable)
  agents.md                   # contributor/agent guide (deliverable)
  Makefile                    # fetch-ffmpeg, build, check (fmt+vet+lint+test)
  cmd/coverfixer/main.go      # CLI front-end (v1)
  internal/core/              # the library — NO cli/gui imports
    options.go                # Options + TranscodeMode enum
    runner.go                 # orchestration: scan → passes → Report
    scan.go                   # walk tree, group by folder, classify files
    image.go                  # resize → baseline JPEG (pure Go), "already ok?" check
    mp3.go                    # id3v2 read/resize/re-embed APIC
    m4a.go                    # mp4tag read/resize/re-embed covr (+ ffmpeg fallback)
    cover.go                  # rename stray jpg, resize cover.jpg, extract → cover.jpg
    transcode.go              # ffmpeg transcode (mp3-320 / aac-256)
    backup.go                 # sidecar .bak before in-place edits
    report.go                 # counters/results (renamed/resized/extracted/…)
  internal/ffmpeg/
    runner.go                 # extract embedded binary to cache, chmod, exec, probe
    embed_darwin_arm64.go     # //go:build tag + //go:embed bin/ffmpeg-darwin-arm64
    embed_darwin_amd64.go     # one file per target platform
    embed_linux_amd64.go
    embed_windows_amd64.go
    bin/                      # static ffmpeg binaries (gitignored; fetched by Makefile)
   internal/ui/               # Wails front-end glue: run lifecycle + formatting
                              # (front-end-agnostic; no Wails import). Bound methods
                              # in cmd/coverfixer-gui/app.go adapt it to Wails.
```

### Core API (shared by CLI + future GUI)

```go
type TranscodeMode int // None, MP3_320, AAC_256

type Options struct {
    Dir            string
    ArtSize        int  // 500
    JPEGQuality    int  // 85
    Recursive      bool // true
    RenameStrayJPG bool // true  (script parity)
    ResizeCoverJPG bool // true  (script parity)
    ExtractCover   bool // true  ("extract cover art or not")
    ResizeEmbedded bool // false (new: resize art inside audio files)
    Transcode      TranscodeMode // None
    Backup         bool // false
    DryRun         bool // false
}

func Run(ctx context.Context, o Options, progress func(Event)) (Report, error)
```

`progress`/`Event` gives the GUI live updates later; the CLI prints them as lines.

### Processing pipeline (`runner.go`)
Walk `Dir`, classify each file, then run passes (each gated by its Option):

1. **JPG pass** (`cover.go`): rename stray `*.jpg` → `cover.jpg`; resize `cover.jpg` to
   fit `ArtSize` baseline JPEG. Skip if already ≤ size & baseline.
2. **Extract pass** (`cover.go`): for folders with audio but no `cover.jpg`, read embedded
   art (pure Go), resize, write `cover.jpg`.
3. **Embedded-art pass** (`mp3.go`/`m4a.go`): for each audio file, decode embedded art;
   if larger than `ArtSize`, resize and re-embed in place. MP3 = id3v2 APIC, M4A = covr.
4. **Transcode pass** (`transcode.go`): if `Transcode != None`, run embedded ffmpeg to
   produce mp3-320 / aac-256, carrying metadata + art over; back up original first; then
   re-run the embedded-art pass on the output so art ends up correctly sized.

`--backup` writes `<file>.bak` once before the first in-place mutation of a file.
`--dry-run` logs intended actions and mutates nothing.

### Image handling (`image.go`)
- Decode jpeg/png; compute fit box within `ArtSize×ArtSize` preserving aspect, **no
  upscale**; resample with `draw.CatmullRom`; encode `image/jpeg` (baseline) at quality.
- "Already correct" check (skip): max dimension ≤ `ArtSize` and source is baseline JPEG.

### Embedded ffmpeg (`internal/ffmpeg`)
- Per-platform `//go:build` + `//go:embed` of a **static** ffmpeg (one file per
  GOOS/GOARCH); each cross-build embeds only its own.
- `runner.go`: on first use, write bytes to `os.UserCacheDir()/coverfixer/ffmpeg-<ver>`,
  `chmod 0755`, cache by version so it isn't rewritten each run; exec for transcode.
- ffmpeg is **never touched** unless a transcode is requested → artwork-only runs stay
  fully native and fast.
- Static binaries are **gitignored**; `make fetch-ffmpeg` downloads them per platform
  (macOS: evermeet.cx/osxexperts; Linux: johnvansickle; Windows: gyan.dev). Document
  sources + the GPL note in `agents.md`.

### CLI (`cmd/coverfixer/main.go`)
Stdlib `flag` (no extra dep). `coverfixer [flags] <dir>`:
`--size`(500) `--quality`(85) `--resize-embedded` `--extract-cover` `--resize-covers`
`--rename-jpg` `--transcode none|mp3-320|aac-256` `--backup` `--dry-run` `--no-recursive`.
Maps flags → `core.Options`, prints progress events, prints `Report` summary (matching the
old script's Renamed/Resized/Extracted/Skipped/Failed counters).

## Dependencies
- `github.com/bogem/id3v2/v2` — MP3 APIC read/write.
- `github.com/Sorrow446/go-mp4tag` (validate covr round-trip; fallback `github.com/abema/go-mp4`,
  and ultimate fallback = bundled ffmpeg for M4A art if pure-Go write proves unreliable).
- `golang.org/x/image/draw` — high-quality resampling.
- `github.com/wailsapp/wails/v2` — GUI only (desktop front-end).

## Risks / fallbacks
- **M4A covr write in pure Go** is the least certain piece. If `go-mp4tag` can't reliably
  rewrite art, fall back to the already-bundled ffmpeg for the M4A art path (keeps single
  binary; just uses ffmpeg slightly more).
- **Binary size** (~70–100 MB/platform from static ffmpeg) is accepted; could later shrink
  via a custom minimal ffmpeg build (mjpeg + libmp3lame + aac only).
- **GPL**: libmp3lame makes the shipped binary GPL — fine for personal use; noted in docs.

## Deliverables (implementation order)
1. `plan.md` (this plan) + `agents.md` (build/run/test/deps, ffmpeg fetch, GPL note, layout).
2. `go.mod` + `internal/core` skeleton (Options, Runner, Report) + pure-Go `image.go`.
3. `mp3.go`, `m4a.go`, `cover.go` (artwork features, no ffmpeg) + tests.
4. `internal/ffmpeg` embed/extract + `transcode.go`.
5. `cmd/coverfixer` CLI wiring all options.
6. `Makefile` with `fetch-ffmpeg`, `build`, and `check` (fmt + vet + lint + test) as the
   project gate.
7. (Later milestone) `internal/gui` Fyne front-end on the same core.

## Verification
- **Gate:** `make check` = `gofmt -l` clean + `go vet ./...` + `golangci-lint run` +
  `go test ./...`. This is the green bar before "done".
- **Unit tests:** image resize golden test (dims + baseline marker); id3v2 APIC round-trip;
  m4a covr round-trip — using tiny fixture files in `testdata/`.
- **Integration:** run against a sample album folder (copied to a temp dir):
  - `cover.jpg` written and decodes to ≤ 500×500 baseline.
  - embedded art in an mp3 and an m4a re-decodes to ≤ 500×500.
  - `--transcode mp3-320` output verified via bundled `ffprobe` (codec=mp3, ~320k) and art
    still ≤ 500×500; original `.bak` present when `--backup`.
  - `--dry-run` mutates nothing (checksum the dir before/after).
- **Self-contained check:** build, move binary to a clean `PATH` without system ffmpeg,
  run a transcode, confirm it extracts and uses the embedded ffmpeg.
