package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/colinstark/coverfixer/internal/core"
)

func TestFormatEventAction(t *testing.T) {
	e := core.Event{Kind: core.EventAction, Op: "resize-cover", Path: "cover.jpg"}
	got := formatEvent(e)
	if !strings.HasPrefix(got, "[ACT]") {
		t.Errorf("action event should start with [ACT], got: %q", got)
	}
	if !strings.Contains(got, "resize-cover") || !strings.Contains(got, "cover.jpg") {
		t.Errorf("action event should contain op and path, got: %q", got)
	}
}

func TestFormatEventActionWithDetail(t *testing.T) {
	e := core.Event{Kind: core.EventAction, Op: "rename", Path: "front.jpg", Detail: "→ cover.jpg"}
	got := formatEvent(e)
	if !strings.Contains(got, "(→ cover.jpg)") {
		t.Errorf("action with detail should include detail in parens, got: %q", got)
	}
}

func TestFormatEventSkip(t *testing.T) {
	e := core.Event{Kind: core.EventSkip, Op: "resize-cover", Path: "cover.jpg", Detail: "already ok"}
	got := formatEvent(e)
	if !strings.HasPrefix(got, "[SKIP]") {
		t.Errorf("skip event should start with [SKIP], got: %q", got)
	}
	if !strings.Contains(got, "already ok") {
		t.Errorf("skip event should contain detail, got: %q", got)
	}
}

func TestFormatEventError(t *testing.T) {
	e := core.Event{Kind: core.EventError, Op: "extract", Path: "song.mp3", Err: errors.New("read failed")}
	got := formatEvent(e)
	if !strings.HasPrefix(got, "[ERR]") {
		t.Errorf("error event should start with [ERR], got: %q", got)
	}
	if !strings.Contains(got, "read failed") {
		t.Errorf("error event should contain error text, got: %q", got)
	}
}

func TestFormatEventErrorNilErr(t *testing.T) {
	t.Run("no_detail", func(t *testing.T) {
		e := core.Event{Kind: core.EventError, Op: "extract", Path: "song.mp3"}
		got := formatEvent(e)
		if strings.Contains(got, "<nil>") {
			t.Errorf("error event with nil Err should not contain '<nil>', got: %q", got)
		}
		if !strings.HasPrefix(got, "[ERR]") {
			t.Errorf("error event should start with [ERR], got: %q", got)
		}
		if !strings.Contains(got, "extract") || !strings.Contains(got, "song.mp3") {
			t.Errorf("error event with nil Err should still contain op and path, got: %q", got)
		}
	})
	t.Run("with_detail", func(t *testing.T) {
		e := core.Event{Kind: core.EventError, Op: "resize-cover", Path: "cover.jpg", Detail: "corrupt header"}
		got := formatEvent(e)
		if strings.Contains(got, "<nil>") {
			t.Errorf("error event with nil Err should not contain '<nil>', got: %q", got)
		}
		if !strings.Contains(got, "corrupt header") {
			t.Errorf("error event with nil Err and Detail should include detail, got: %q", got)
		}
	})
}

func TestFormatEventInfo(t *testing.T) {
	e := core.Event{Kind: core.EventInfo, Op: "scan", Path: "/music", Detail: "3 files"}
	got := formatEvent(e)
	if !strings.HasPrefix(got, "[INFO]") {
		t.Errorf("info event should start with [INFO], got: %q", got)
	}
}

func TestFormatEventDistinctKinds(t *testing.T) {
	prefixes := map[core.EventKind]string{
		core.EventAction: "[ACT]",
		core.EventSkip:   "[SKIP]",
		core.EventError:  "[ERR]",
		core.EventInfo:   "[INFO]",
	}
	for kind, prefix := range prefixes {
		e := core.Event{Kind: kind, Op: "op", Path: "path"}
		got := formatEvent(e)
		if !strings.HasPrefix(got, prefix) {
			t.Errorf("EventKind %d: expected prefix %q, got: %q", kind, prefix, got)
		}
	}
}

func TestFormatReportAllCounters(t *testing.T) {
	r := core.Report{Renamed: 1, CoversResized: 2, Extracted: 3, EmbeddedResized: 4, Transcoded: 5, Skipped: 6, Failed: 7}
	got := formatReport(r, false)
	expected := []string{
		"Renamed: 1",
		"Covers Resized: 2",
		"Extracted: 3",
		"Embedded Resized: 4",
		"Transcoded: 5",
		"Skipped: 6",
		"Failed: 7",
	}
	for _, exp := range expected {
		if !strings.Contains(got, exp) {
			t.Errorf("formatReport missing %q, got: %q", exp, got)
		}
	}
}

func TestFormatReportZero(t *testing.T) {
	r := core.Report{}
	got := formatReport(r, false)
	expected := []string{
		"Renamed: 0",
		"Covers Resized: 0",
		"Extracted: 0",
		"Embedded Resized: 0",
		"Transcoded: 0",
		"Skipped: 0",
		"Failed: 0",
	}
	for _, exp := range expected {
		if !strings.Contains(got, exp) {
			t.Errorf("zero formatReport missing %q, got: %q", exp, got)
		}
	}
}

func TestFormatReportDryRunBanner(t *testing.T) {
	r := core.Report{CoversResized: 3}

	dry := formatReport(r, true)
	if !strings.Contains(strings.ToLower(dry), "dry-run") {
		t.Errorf("dry-run report should mention dry-run, got: %q", dry)
	}
	if !strings.Contains(dry, "Covers Resized: 3") {
		t.Errorf("dry-run report should still show counters, got: %q", dry)
	}

	real := formatReport(r, false)
	if strings.Contains(strings.ToLower(real), "dry-run") {
		t.Errorf("non-dry-run report should not mention dry-run, got: %q", real)
	}
}
