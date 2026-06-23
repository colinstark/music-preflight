// Command coverfixer batch-fixes cover art in a music library: it resizes
// artwork embedded in MP3/M4A files, writes and resizes folder cover.jpg files,
// and can optionally transcode audio to mp3-320 or aac-256.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/colinstark/coverfixer/internal/core"
)

func main() {
	o := core.DefaultOptions()

	flag.IntVar(&o.ArtSize, "size", o.ArtSize, "max artwork dimension; images fit within size×size")
	flag.IntVar(&o.JPEGQuality, "quality", o.JPEGQuality, "baseline JPEG quality (1-100)")
	flag.BoolVar(&o.RenameStrayJPG, "rename-jpg", o.RenameStrayJPG, "rename a lone non-cover *.jpg to cover.jpg")
	flag.BoolVar(&o.ResizeCoverJPG, "resize-covers", o.ResizeCoverJPG, "resize existing cover.jpg files")
	flag.BoolVar(&o.ExtractCover, "extract-cover", o.ExtractCover, "write cover.jpg from embedded art when missing")
	flag.BoolVar(&o.ResizeEmbedded, "resize-embedded", o.ResizeEmbedded, "resize artwork embedded inside audio files, in place")
	transcode := flag.String("transcode", "none", "audio conversion: none|mp3-320|aac-256")
	genre := flag.String("genre", "", "set the genre tag on every audio file to this value (empty = off)")
	flag.BoolVar(&o.Backup, "backup", o.Backup, "write a <file>.bak copy before modifying a file")
	flag.BoolVar(&o.DryRun, "dry-run", o.DryRun, "report intended actions without changing anything")
	noRecursive := flag.Bool("no-recursive", false, "do not descend into subfolders")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	dir := flag.Arg(0)

	mode, err := core.ParseTranscodeMode(*transcode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverfixer:", err)
		os.Exit(2)
	}
	o.Transcode = mode
	o.Recursive = !*noRecursive
	o.Dir = dir
	if *genre != "" {
		o.SetGenre = true
		o.Genre = *genre
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "coverfixer: %q is not a directory\n", dir)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Printf("Scanning: %s\n", dir)
	if o.DryRun {
		fmt.Println("(dry-run — no files will be modified)")
	}

	// Resolve the working directory once; printEvent uses it to shorten paths.
	wd, _ := os.Getwd()

	rep, err := core.Run(ctx, o, func(e core.Event) { printEvent(wd, e) })
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverfixer:", err)
		os.Exit(1)
	}

	printSummary(rep, o.DryRun)
	if rep.Failed > 0 {
		os.Exit(1)
	}
}

func printEvent(wd string, e core.Event) {
	rel := e.Path
	if wd != "" {
		if r, err := filepath.Rel(wd, e.Path); err == nil && len(r) < len(rel) {
			rel = r
		}
	}
	switch e.Kind {
	case core.EventInfo:
		// Folder header: a blank line then the path, grouping the events below it.
		fmt.Printf("\n%s\n", rel)
	case core.EventAction:
		if e.Detail != "" {
			fmt.Printf("  %-16s %s  %s\n", e.Op, rel, e.Detail)
		} else {
			fmt.Printf("  %-16s %s\n", e.Op, rel)
		}
	case core.EventSkip:
		fmt.Printf("  %-16s %s  (%s)\n", "skip", rel, e.Detail)
	case core.EventError:
		fmt.Printf("  %-16s %s  ERROR: %v\n", e.Op, rel, e.Err)
	}
}

func printSummary(r core.Report, dryRun bool) {
	fmt.Println()
	fmt.Println("Done.")
	if dryRun {
		fmt.Println("  (dry-run — no changes made)")
	}
	fmt.Printf("  Renamed          : %d\n", r.Renamed)
	fmt.Printf("  Covers resized   : %d\n", r.CoversResized)
	fmt.Printf("  Extracted        : %d\n", r.Extracted)
	fmt.Printf("  Embedded resized : %d\n", r.EmbeddedResized)
	fmt.Printf("  Transcoded       : %d\n", r.Transcoded)
	fmt.Printf("  Genres set       : %d\n", r.GenresSet)
	fmt.Printf("  Skipped          : %d\n", r.Skipped)
	if r.Failed > 0 {
		fmt.Printf("  Failed           : %d\n", r.Failed)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: coverfixer [flags] <music-folder>\n\nFlags:\n")
	flag.PrintDefaults()
}
