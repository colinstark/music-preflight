package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupRootDuplicatesFolder(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.mp3"), []byte("aaa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.m4a"), []byte("bbb"), 0o644); err != nil {
		t.Fatal(err)
	}

	var rep reportAccum
	o := Options{Dir: root, Backup: true}
	if err := backupRoot(o, &rep, func(Event) {}); err != nil {
		t.Fatalf("backupRoot: %v", err)
	}

	dst := filepath.Join(filepath.Dir(root), "backup", filepath.Base(root))
	check := func(rel string, wantDir bool) {
		t.Helper()
		info, err := os.Stat(filepath.Join(dst, rel))
		if err != nil {
			t.Errorf("missing %q in backup: %v", rel, err)
			return
		}
		if info.IsDir() != wantDir {
			t.Errorf("%q: isDir=%v, want %v", rel, info.IsDir(), wantDir)
		}
	}
	check("a.mp3", false)
	check("sub", true)
	check(filepath.Join("sub", "b.m4a"), false)

	// Re-running refreshes the backup: the old copy is replaced with the current
	// state (no stale leftover, no error on the pre-existing target).
	if err := os.WriteFile(filepath.Join(root, "a.mp3"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := backupRoot(o, &rep, func(Event) {}); err != nil {
		t.Fatalf("backupRoot refresh: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "a.mp3"))
	if err != nil || string(got) != "changed" {
		t.Errorf("backup not refreshed; got %q, want \"changed\"", string(got))
	}

	// No .bak sidecars are produced anywhere.
	if fileExists(filepath.Join(root, "a.mp3.bak")) {
		t.Error("backup must not create .bak sidecars")
	}
}

func TestBackupRootNoOpWhenDryRunOrOff(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var rep reportAccum

	// DryRun: nothing is copied.
	if err := backupRoot(Options{Dir: root, Backup: true, DryRun: true}, &rep, func(Event) {}); err != nil {
		t.Fatalf("dry-run backupRoot: %v", err)
	}
	if fileExists(filepath.Join(filepath.Dir(root), "backup")) {
		t.Error("dry-run should not create a backup folder")
	}

	// Backup off: nothing is copied.
	if err := backupRoot(Options{Dir: root, Backup: false}, &rep, func(Event) {}); err != nil {
		t.Fatalf("backup-off backupRoot: %v", err)
	}
	if fileExists(filepath.Join(filepath.Dir(root), "backup")) {
		t.Error("backup off should not create a backup folder")
	}
}
