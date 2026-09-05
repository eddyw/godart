package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// app holds the layout of 1 Flutter application that Go drives.
type app struct {
	// dir holds pubspec.yaml.
	dir string
	// module is the directory of the Go module, relative to dir.
	module string
	// out is the directory of the Dart output, relative to dir.
	out string
	// host is the file of the generated host, relative to dir.
	host string
	// root is the directory of the godart repository. nodegen and the runtime
	// library of Dart stand under it.
	root string
	// nodes holds the node types of the last scan, 1 name for each line. The
	// filter of the build writes it, and the host of nodegen reads it.
	nodes string
}

func (a *app) modulePath() string  { return filepath.Join(a.dir, a.module) }
func (a *app) outPath() string     { return filepath.Join(a.dir, a.out) }
func (a *app) hostPath() string    { return filepath.Join(a.dir, a.host) }
func (a *app) runtimePath() string { return filepath.Join(a.root, "runtime", "goruntime") }
func (a *app) nodegenPath() string { return filepath.Join(a.root, "flutter", "nodegen") }

// findApp walks up from the working directory for pubspec.yaml.
func findApp() (*app, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "pubspec.yaml")); err == nil {
			break
		}
		up := filepath.Dir(dir)
		if up == dir {
			return nil, fmt.Errorf("no pubspec.yaml stands here or above")
		}
		dir = up
	}
	a := &app{dir: dir, module: "go", out: "gen", host: "lib/widgets_gen.dart"}
	if err := a.readPubspec(); err != nil {
		return nil, err
	}
	root, err := findRoot(dir)
	if err != nil {
		return nil, err
	}
	a.root = root
	if _, err := os.Stat(a.modulePath()); err != nil {
		return nil, fmt.Errorf("%s holds no Go module. Read the glutter block of pubspec.yaml", a.modulePath())
	}
	return a, nil
}

// readPubspec reads the glutter block. A key of that block takes 2 spaces of
// indent, thus the reader needs no YAML library.
func (a *app) readPubspec() error {
	f, err := os.Open(filepath.Join(a.dir, "pubspec.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()
	inBlock := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "glutter:") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			break
		}
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		switch key {
		case "module":
			a.module = value
		case "out":
			a.out = value
		case "host":
			a.host = value
		}
	}
	return s.Err()
}

// findRoot gives the directory of the godart repository. The generator of the
// host and the Dart runtime library stand under it.
func findRoot(start string) (string, error) {
	if v := os.Getenv("GODART_ROOT"); v != "" {
		if isRoot(v) {
			return v, nil
		}
		return "", fmt.Errorf("GODART_ROOT names %s, and that directory holds no flutter/nodegen", v)
	}
	dir := start
	for {
		if isRoot(dir) {
			return dir, nil
		}
		up := filepath.Dir(dir)
		if up == dir {
			break
		}
		dir = up
	}
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
		for {
			if isRoot(dir) {
				return dir, nil
			}
			up := filepath.Dir(dir)
			if up == dir {
				break
			}
			dir = up
		}
	}
	return "", fmt.Errorf("no godart repository stands above %s. Set GODART_ROOT", start)
}

func isRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "flutter", "nodegen", "bin", "nodegen.dart"))
	return err == nil
}

// flutterPass runs the flutter command with the arguments of the user.
func flutterPass(args []string) error {
	cmd := exec.Command("flutter", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// run1 runs 1 command in dir and it gives the output to the terminal.
func run1(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// output runs 1 command and it gives the standard output.
func output(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	return string(b), err
}
