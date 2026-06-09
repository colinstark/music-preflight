package gui

import "testing"

// --- positiveIntValidator: empty is allowed (falls back to engine default),
// otherwise the value must be an integer within [min, max] (max 0 = no cap). ---

func TestPositiveIntValidator(t *testing.T) {
	tests := []struct {
		name    string
		min     int
		max     int
		input   string
		wantErr bool
	}{
		// Art size: positive, no upper bound.
		{"art empty ok", 1, 0, "", false},
		{"art valid", 1, 0, "500", false},
		{"art large ok", 1, 0, "100000", false},
		{"art zero", 1, 0, "0", true},
		{"art negative", 1, 0, "-5", true},
		{"art non-numeric", 1, 0, "abc", true},
		// JPEG quality: 1–100.
		{"quality empty ok", 1, 100, "", false},
		{"quality min", 1, 100, "1", false},
		{"quality max", 1, 100, "100", false},
		{"quality typical", 1, 100, "85", false},
		{"quality over max", 1, 100, "101", true},
		{"quality zero", 1, 100, "0", true},
		{"quality non-numeric", 1, 100, "xx", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := positiveIntValidator("field", tc.min, tc.max)(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("input %q: expected an error, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("input %q: expected nil, got %v", tc.input, err)
			}
		})
	}
}

func TestEntriesHaveValidators(t *testing.T) {
	ui := newTestUI(t)
	if ui.artSizeEntry.Validator == nil {
		t.Error("art-size entry should have a validator wired")
	}
	if ui.qualityEntry.Validator == nil {
		t.Error("quality entry should have a validator wired")
	}
}

// A quality above 100 is clamped to 100 (max quality) rather than silently
// resetting to the engine default of 85.
func TestQualityClampedToMax(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.qualityEntry.SetText("150")
	opts := ui.options()
	if opts.JPEGQuality != 100 {
		t.Errorf("JPEGQuality = %d, want 100 (clamped from 150)", opts.JPEGQuality)
	}
}
