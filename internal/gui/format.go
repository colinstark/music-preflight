package gui

import (
	"fmt"

	"github.com/colinstark/coverfixer/internal/core"
)

// formatEvent renders a core.Event as a human-readable log line with
// distinct formatting per EventKind: [ACT] for action, [SKIP] for skip,
// [ERR] for error (surfacing Err text), [INFO] for informational.
func formatEvent(e core.Event) string {
	switch e.Kind {
	case core.EventAction:
		if e.Detail != "" {
			return fmt.Sprintf("[ACT] %s: %s (%s)", e.Op, e.Path, e.Detail)
		}
		return fmt.Sprintf("[ACT] %s: %s", e.Op, e.Path)
	case core.EventSkip:
		if e.Detail != "" {
			return fmt.Sprintf("[SKIP] %s: %s (%s)", e.Op, e.Path, e.Detail)
		}
		return fmt.Sprintf("[SKIP] %s: %s", e.Op, e.Path)
	case core.EventError:
		if e.Err != nil {
			return fmt.Sprintf("[ERR] %s: %s: %v", e.Op, e.Path, e.Err)
		}
		if e.Detail != "" {
			return fmt.Sprintf("[ERR] %s: %s (%s)", e.Op, e.Path, e.Detail)
		}
		return fmt.Sprintf("[ERR] %s: %s", e.Op, e.Path)
	default: // EventInfo
		if e.Detail != "" {
			return fmt.Sprintf("[INFO] %s: %s (%s)", e.Op, e.Path, e.Detail)
		}
		return fmt.Sprintf("[INFO] %s: %s", e.Op, e.Path)
	}
}

// formatReport renders a core.Report as a multi-line summary string
// showing all seven counters.
func formatReport(r core.Report) string {
	return fmt.Sprintf(
		"Renamed: %d\nCovers Resized: %d\nExtracted: %d\nEmbedded Resized: %d\nTranscoded: %d\nSkipped: %d\nFailed: %d",
		r.Renamed, r.CoversResized, r.Extracted, r.EmbeddedResized, r.Transcoded, r.Skipped, r.Failed,
	)
}
