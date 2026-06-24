// Command coverfixer-gui is the Wails desktop front-end for coverfixer. It is
// a thin adapter: every bound method delegates to internal/ui.Controller or
// the Wails runtime, so all run lifecycle and formatting logic stays in the
// headless-testable internal/ui package.
package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/colinstark/coverfixer/internal/core"
	"github.com/colinstark/coverfixer/internal/ui"
)

// App is the struct whose methods are exposed to the frontend via Wails
// bindings. It holds the Wails runtime context (captured in startup) and a
// ui.Controller that owns the engine run lifecycle.
type App struct {
	ctx context.Context
	c   *ui.Controller

	// dirMu guards pendingDir. A folder dropped on the app/Dock icon can arrive
	// (via Mac.OnFileOpen) before the frontend has connected — e.g. when the app
	// is launched by the drop. It is stashed here and pulled once via
	// InitialFolder() during frontend init.
	dirMu      sync.Mutex
	pendingDir string
}

// NewApp returns an App with no controller yet; the controller is created in
// startup once the Wails runtime context is available, because emitting events
// requires it.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails once the runtime context is ready. The context is
// saved so bound methods can open dialogs and emit events.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.c = ui.NewController(&wailsEmitter{ctx: ctx})
	// Deliver folder drops into the running window. On macOS these are distinct
	// from Dock-icon drops (which come through handleOpenFile / Mac.OnFileOpen).
	runtime.OnFileDrop(ctx, func(_, _ int, paths []string) {
		if len(paths) == 0 {
			return
		}
		if dir := resolveFolder(paths[0]); dir != "" {
			a.applyFolder(dir)
		}
	})
}

// DefaultRequest returns the GUI defaults so the frontend can seed its form
// from a single source of truth (Go). Mirrors the original Fyne UI's defaults,
// including dry-run ON for safety.
func (a *App) DefaultRequest() ui.RunRequest {
	return ui.DefaultRequest()
}

// Run starts an engine run for the given request and returns immediately. The
// request is validated here; an already-running or invalid request returns an
// error (which Wails surfaces as a rejected promise). Progress and completion
// arrive later via the cf:progress / cf:done / cf:error / cf:state events.
func (a *App) Run(req ui.RunRequest) error {
	return a.c.Start(req)
}

// Cancel cancels the in-flight run, if any. No-op when idle.
func (a *App) Cancel() {
	a.c.Cancel()
}

// IsRunning reports whether a run is currently in flight.
func (a *App) IsRunning() bool {
	return a.c.IsRunning()
}

// OpenFolder opens a native directory picker and returns the chosen path, or an
// empty string if the user cancelled.
func (a *App) OpenFolder() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose a music folder",
	})
	if err != nil {
		return ""
	}
	return dir
}

// resolveFolder maps a dropped path to a music folder. A directory is used as
// delivered; a file (e.g. an .mp3) resolves to its containing folder, which is
// the natural unit for coverfixer. An empty/unsatisfiable path returns "".
func resolveFolder(p string) string {
	if p == "" {
		return ""
	}
	info, err := os.Stat(p)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return p
	}
	return filepath.Dir(p)
}

// applyFolder selects a folder the same way the picker does, and notifies the
// frontend. If the frontend is connected (a.ctx set) it streams the path on the
// cf:folder event; otherwise (launch-by-drop, before startup) the path is
// stashed for InitialFolder to pull once init() runs.
func (a *App) applyFolder(dir string) {
	if dir == "" {
		return
	}
	a.dirMu.Lock()
	a.pendingDir = dir
	a.dirMu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "cf:folder", dir)
	}
}

// handleOpenFile is the Mac.OnFileOpen callback: a folder/file was dropped on
// the app/Dock icon, launching it or while running.
func (a *App) handleOpenFile(p string) {
	if dir := resolveFolder(p); dir != "" {
		a.applyFolder(dir)
	}
}

// InitialFolder returns a folder provided at launch (e.g. by dropping a folder
// on the Dock icon before the app was running) and clears it. The frontend
// calls this once during init to seed the selected folder; empty means none.
func (a *App) InitialFolder() string {
	a.dirMu.Lock()
	defer a.dirMu.Unlock()
	dir := a.pendingDir
	a.pendingDir = ""
	return dir
}

// ReadFirstMetadata returns the genre and album artist of the first audio file
// under dir, used to prefill the GUI's metadata fields once a folder is picked.
func (a *App) ReadFirstMetadata(dir string) core.FirstMetadata {
	if dir == "" {
		return core.FirstMetadata{}
	}
	return core.ReadFirstMetadata(dir)
}

// ReadLibrary scans dir for audio files and returns them grouped by album for
// the GUI's idle preview (artwork thumbnails, track titles, durations). It is
// read-only and never mutates files. recursive mirrors the run's scope.
func (a *App) ReadLibrary(dir string, recursive bool) ([]core.Album, error) {
	if dir == "" {
		return nil, nil
	}
	return core.ReadLibrary(dir, recursive)
}

// wailsEmitter adapts ui.Emitter to Wails' runtime.EventsEmit. It is the only
// bridge from internal/ui to Wails, keeping internal/ui dependency-free.
type wailsEmitter struct{ ctx context.Context }

// Emit forwards a named event with its payload to every subscribed frontend
// listener.
func (e *wailsEmitter) Emit(name string, data ...any) {
	runtime.EventsEmit(e.ctx, name, data...)
}
