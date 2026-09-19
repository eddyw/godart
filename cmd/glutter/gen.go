package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
	"github.com/eddyw/godart/internal/nodefilter"
)

// gen compiles the Go module into Dart, and it writes the host when the set of
// node types of the application changed.
func (a *app) gen() error {
	if err := a.build(true); err != nil {
		return err
	}
	changed, err := a.writeHost()
	if err != nil {
		return err
	}
	if changed {
		fmt.Printf("glutter: %s holds the host\n", a.host)
	}
	return nil
}

func (a *app) build(format bool) error {
	stats, err := driver.BuildWithStats(a.options(format))
	if err != nil {
		return err
	}
	fmt.Printf("glutter: %d compiled, %d cached\n",
		len(stats.Compiled), len(stats.Cached))
	return nil
}

func (a *app) options(format bool) driver.Options {
	return driver.Options{
		Dir:      a.modulePath(),
		Patterns: []string{"./..."},
		Out:      a.outPath(),
		Runtime:  a.runtimePath(),
		Format:   format,
		Filter:   a.filter,
	}
}

// filter drops each node type that the application does not name from the
// generated file of each pack. The compiler then writes no Dart class and no
// type descriptor of a widget that no code builds. Read internal/nodefilter.
//
// The driver calls it in front of each load, thus a watch cycle reads the
// names of that moment and a new node type reaches the build at once. The scan
// runs 1 time for each build, and writeHost reads the same answer.
func (a *app) filter() (load.Filter, error) {
	names, err := a.scan()
	if err != nil {
		// The module holds an error, thus the scan reads no type information.
		// The build then reads the whole node set, and the compiler reports
		// the error of the source itself. A build with no filter is correct,
		// and it writes more bytes.
		a.nodes = ""
		return nil, nil
	}
	a.nodes = names
	root, err := a.uiModule()
	if err != nil {
		// The application reads no node module, thus no pack stands here.
		return nil, nil
	}
	return nodefilter.Sources(root, typeNames(names))
}

// typeNames keeps the node types of the answer of the scan. A line with a
// quotation mark in front holds a string constant of the module, and nodegen
// alone reads such a line.
func typeNames(names string) []string {
	out := make([]string, 0, 64)
	for _, line := range strings.Fields(names) {
		if strings.HasPrefix(line, `"`) {
			continue
		}
		out = append(out, line)
	}
	return out
}

// cacheDir holds the state of glutter for 1 application.
func (a *app) cacheDir() string {
	return filepath.Join(a.dir, ".dart_tool", "glutter")
}

// writeHost refreshes the host after a node or dependency input changes.
func (a *app) writeHost() (bool, error) {
	if _, err := os.Stat(filepath.Join(a.dir, ".dart_tool", "package_config.json")); os.IsNotExist(err) {
		if err := run1(a.dir, "flutter", "pub", "get"); err != nil {
			return false, err
		}
	}
	names, err := a.nodeNames()
	if err != nil {
		return false, err
	}
	inputs, err := a.hostInputs()
	if err != nil {
		return false, err
	}
	key := names + "\n" + inputs
	cache := filepath.Join(a.cacheDir(), "nodes.txt")
	old, _ := os.ReadFile(cache)
	_, missing := os.Stat(a.hostPath())
	if string(old) == key && missing == nil {
		return false, nil
	}
	if err := a.nodegen(names); err != nil {
		return false, err
	}
	if err := os.MkdirAll(a.cacheDir(), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(cache, []byte(key), 0o644)
}

// nodeNames gives the node types of the last build of this process. The filter
// of the build already ran the scan, thus the host needs no second run of it.
func (a *app) nodeNames() (string, error) {
	if a.nodes != "" {
		return a.nodes, nil
	}
	return a.scan()
}

// scan names each node type that the application uses.
func (a *app) scan() (string, error) {
	bin, err := a.scanner()
	if err != nil {
		return "", err
	}
	return output(a.dir, bin, a.modulePath())
}

// scanner builds the scanner 1 time and it gives the path of the binary. A go
// run of each cycle costs about 1 second more.
//
// A binary that stands older than the source of the scanner gives the answer
// of the version before, thus the build runs again in that case.
func (a *app) scanner() (string, error) {
	bin := filepath.Join(a.cacheDir(), "nodescan")
	if info, err := os.Stat(bin); err == nil && !a.scannerIsOld(info.ModTime()) {
		return bin, nil
	}
	if err := os.MkdirAll(a.cacheDir(), 0o755); err != nil {
		return "", err
	}
	if err := run1(a.root, "go", "build", "-o", bin, "./flutter/nodescan"); err != nil {
		return "", err
	}
	return bin, nil
}

// scannerIsOld reports whether a source file of the scanner is newer than the
// binary that the cache holds.
func (a *app) scannerIsOld(built time.Time) bool {
	dir := filepath.Join(a.root, "flutter", "nodescan")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.ModTime().After(built) {
			return true
		}
	}
	return false
}

// nodegen writes the host of the names.
func (a *app) nodegen(names string) error {
	dir := a.nodegenPath()
	if _, err := os.Stat(filepath.Join(dir, ".dart_tool")); err != nil {
		if err := run1(dir, "dart", "pub", "get"); err != nil {
			return err
		}
	}
	f, err := os.CreateTemp("", "glutter-nodes-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(names); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return run1(dir, "dart", "run", "bin/nodegen.dart", "host",
		a.dir, a.hostPath(), f.Name())
}

// after compiles the Go module and it then runs the flutter command.
func (a *app) after(cmd string, args []string) error {
	if err := a.gen(); err != nil {
		return err
	}
	full := append([]string{cmd}, args...)
	c := exec1(a.dir, "flutter", full)
	return c.Run()
}
