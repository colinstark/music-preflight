package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// albumFolder groups the artwork-relevant files found in a single directory.
type albumFolder struct {
	dir      string
	jpgs     []string // every *.jpg / *.jpeg in the folder
	audio    []string // mp3 / m4a / aac files
	hasCover bool     // a file named cover.jpg (any case) exists
}

// audioKind classifies an audio file by how its embedded art is accessed.
type audioKind int

const (
	audioOther audioKind = iota // raw .aac or anything without a tag container
	audioMP3
	audioM4A
)

func classifyAudio(path string) audioKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return audioMP3
	case ".m4a":
		return audioM4A
	default:
		return audioOther
	}
}

// scan walks root and groups files into albumFolders. AppleDouble (._*) sidecar
// files are ignored, matching the original shell script. Results are sorted for
// deterministic processing order.
func scan(root string, recursive bool) ([]*albumFolder, error) {
	folders := map[string]*albumFolder{}
	get := func(dir string) *albumFolder {
		f := folders[dir]
		if f == nil {
			f = &albumFolder{dir: dir}
			folders[dir] = f
		}
		return f
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && path != root {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, "._") {
			return nil
		}
		dir := filepath.Dir(path)
		switch strings.ToLower(filepath.Ext(name)) {
		case ".jpg", ".jpeg":
			f := get(dir)
			f.jpgs = append(f.jpgs, path)
			if strings.EqualFold(name, "cover.jpg") {
				f.hasCover = true
			}
		case ".mp3", ".m4a", ".aac":
			get(dir).audio = append(get(dir).audio, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]*albumFolder, 0, len(folders))
	for _, f := range folders {
		sort.Strings(f.jpgs)
		sort.Strings(f.audio)
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].dir < out[j].dir })
	return out, nil
}

// coverPath returns the canonical cover.jpg path for a folder.
func coverPath(dir string) string {
	return filepath.Join(dir, "cover.jpg")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
