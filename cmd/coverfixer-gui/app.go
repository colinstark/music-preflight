// Command coverfixer-gui is the Wails desktop front-end for coverfixer. It is
// a thin adapter: every bound method delegates to internal/ui.Controller or
// the Wails runtime, so all run lifecycle and formatting logic stays in the
// headless-testable internal/ui package.
package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/colinstark/coverfixer/internal/ui"
)

// App is the struct whose methods are exposed to the frontend via Wails
// bindings. It holds the Wails runtime context (captured in startup) and a
// ui.Controller that owns the engine run lifecycle.
type App struct {
	ctx context.Context
	c   *ui.Controller
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

// wailsEmitter adapts ui.Emitter to Wails' runtime.EventsEmit. It is the only
// bridge from internal/ui to Wails, keeping internal/ui dependency-free.
type wailsEmitter struct{ ctx context.Context }

// Emit forwards a named event with its payload to every subscribed frontend
// listener.
func (e *wailsEmitter) Emit(name string, data ...any) {
	runtime.EventsEmit(e.ctx, name, data...)
}
