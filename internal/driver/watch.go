package driver

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// The pause after a file event. An editor writes several files at one time,
// thus the build waits for the last write.
const settle = 120 * time.Millisecond

// Watch builds the packages, then builds them again after each change of a Go
// file. The cache of stage 6 gives the early cutoff, thus only the packages
// that need a second compile take one.
func Watch(opt Options, log io.Writer) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	if err := watchTree(watcher, opt.Dir, opt.Out); err != nil {
		return err
	}
	report(log, buildOnce(opt))
	fmt.Fprintf(log, "gart: watch %s\n", opt.Dir)
	timer := time.NewTimer(settle)
	if !timer.Stop() {
		<-timer.C
	}
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !isGoFile(event.Name) {
				continue
			}
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = watchTree(watcher, event.Name, opt.Out)
				}
			}
			timer.Reset(settle)
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(log, "gart: watch error: %v\n", err)
		case <-timer.C:
			report(log, buildOnce(opt))
		}
	}
}

type outcome struct {
	stats *Stats
	err   error
}

func buildOnce(opt Options) outcome {
	stats, err := BuildWithStats(opt)
	return outcome{stats: stats, err: err}
}

func report(log io.Writer, done outcome) {
	stats, err := done.stats, done.err
	if err != nil {
		fmt.Fprintf(log, "gart: %v\n", err)
		return
	}
	fmt.Fprintf(log, "gart: %d compiled, %d cached\n", len(stats.Compiled), len(stats.Cached))
	for _, path := range stats.Compiled {
		fmt.Fprintf(log, "  %s\n", path)
	}
}

// watchTree adds each directory of the module to the watcher. The output
// directory stays out, because the build writes into it.
func watchTree(watcher *fsnotify.Watcher, root, out string) error {
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		base := entry.Name()
		if strings.HasPrefix(base, ".") && base != "." || abs == absOut {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}

func isGoFile(path string) bool {
	return strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".mod")
}
