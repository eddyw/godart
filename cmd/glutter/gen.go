package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eddyw/godart/internal/driver"
)

// checks names the tests of the Go module that a command runs first.
type checks struct{ vet, test bool }

// gen compiles the Go module into Dart, and it writes the host when the set of
// node types of the application changed.
func (a *app) gen(full bool) error {
	c := checks{vet: true, test: full}
	if err := a.build(c, true); err != nil {
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

// build runs the checks and it compiles the module into the output directory.
func (a *app) build(c checks, format bool) error {
	if c.vet {
		if err := run1(a.modulePath(), "go", "vet", "./..."); err != nil {
			return err
		}
	}
	if c.test {
		if err := run1(a.modulePath(), "go", "test", "./..."); err != nil {
			return err
		}
	}
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
	}
}

// cacheDir holds the state of glutter for 1 application.
func (a *app) cacheDir() string {
	return filepath.Join(a.dir, ".dart_tool", "glutter")
}

// writeHost writes the generated host when the set of node types changed. It
// reports whether it wrote the file.
//
// nodegen reads the whole public API of Flutter, thus a run takes about 8
// seconds. The set of node types changes only when the application names a new
// widget, and the cache below holds the set of the run before.
func (a *app) writeHost() (bool, error) {
	names, err := a.scan()
	if err != nil {
		return false, err
	}
	cache := filepath.Join(a.cacheDir(), "nodes.txt")
	old, _ := os.ReadFile(cache)
	_, missing := os.Stat(a.hostPath())
	if string(old) == names && missing == nil {
		return false, nil
	}
	if err := a.nodegen(names); err != nil {
		return false, err
	}
	if err := os.MkdirAll(a.cacheDir(), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(cache, []byte(names), 0o644)
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
func (a *app) scanner() (string, error) {
	bin := filepath.Join(a.cacheDir(), "nodescan")
	if _, err := os.Stat(bin); err == nil {
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
		filepath.Join(a.root, "flutter", "goflutter"), a.hostPath(), f.Name())
}

// after compiles the Go module and it then runs the flutter command.
func (a *app) after(cmd string, args []string) error {
	if err := a.gen(true); err != nil {
		return err
	}
	full := append([]string{cmd}, args...)
	c := exec1(a.dir, "flutter", full)
	return c.Run()
}
