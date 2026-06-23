package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// backupRoot duplicates o.Dir (the selected top-most folder) into a sibling
// "backup" folder before the run mutates anything: <parent>/backup/<rootname>.
// A pre-existing backup of the same root is replaced so each run produces a
// fresh snapshot. It is a no-op when Backup is unset or under DryRun, and it
// never writes per-file .bak sidecars. A returned error is engine-level and
// aborts the run.
func backupRoot(o Options, rep *reportAccum, progress func(Event)) error {
	if !o.Backup || o.DryRun {
		return nil
	}
	root := filepath.Clean(o.Dir)
	dst := filepath.Join(filepath.Dir(root), "backup", filepath.Base(root))

	if fileExists(dst) {
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("remove old backup %q: %w", dst, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create backup folder: %w", err)
	}
	if err := copyTree(root, dst); err != nil {
		return fmt.Errorf("backup %q: %w", dst, err)
	}
	rep.info(progress, "backup", dst, "")
	return nil
}

// copyTree recursively copies the src tree into dst, preserving the directory
// structure and file permissions. Symlinks are followed by os.Open.
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target)
	})
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
		return fmt.Errorf("finalize copy: %w", err)
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
