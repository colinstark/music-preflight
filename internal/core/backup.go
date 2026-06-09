package core

import (
	"fmt"
	"io"
	"os"
)

// maybeBackup writes a <path>.bak copy of the file when o.Backup is set and a
// backup does not already exist. It is a no-op under DryRun. Call it immediately
// before the first in-place mutation of a file.
func maybeBackup(o Options, path string) error {
	if !o.Backup || o.DryRun {
		return nil
	}
	bak := path + ".bak"
	if fileExists(bak) {
		return nil
	}
	return copyFile(path, bak)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("finalize backup: %w", err)
	}
	return nil
}

// writeFileAtomic writes data to path via a temp file + rename, preserving the
// existing file's permissions when present.
func writeFileAtomic(path string, data []byte) error {
	perm := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	tmp := path + ".coverfixer.tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
