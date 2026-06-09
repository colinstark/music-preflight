// Command coverfixer-gui provides a native desktop GUI for batch-fixing
// cover art in a music library. It is a thin entry point that creates
// the Fyne app and delegates all UI logic to internal/gui.
package main

import (
	"fyne.io/fyne/v2/app"
	"github.com/colinstark/coverfixer/internal/gui"
)

func main() {
	a := app.New()
	gui.New(a).ShowAndRun()
}
