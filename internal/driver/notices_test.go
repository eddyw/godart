package driver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSourceNotices(t *testing.T) {
	for _, mode := range []string{"module", "vendor"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			dep := filepath.Join(dir, "dep")
			app := filepath.Join(dir, "app")
			for _, path := range []string{app, filepath.Join(dep, "pkg")} {
				if err := os.MkdirAll(path, 0755); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, filepath.Join(dep, "go.mod"), "module example.com/dep\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dep, "LICENSE"), "Dependency license version 1\n")
			writeFile(t, filepath.Join(dep, "pkg", "NOTICE"), "Dependency attribution\n")
			writeFile(t, filepath.Join(dep, "pkg", "value.go"), "// Copyright 2026 Dependency Authors.\npackage pkg\nconst Value = 42\n")
			writeFile(t, filepath.Join(app, "go.mod"), "module example.com/app\n\ngo 1.27\nrequire example.com/dep v0.0.0\nreplace example.com/dep => ../dep\n")
			writeFile(t, filepath.Join(app, "LICENSE"), "Application license\n")
			writeFile(t, filepath.Join(app, "main.go"), "package main\nimport (\"fmt\"; \"example.com/dep/pkg\")\nfunc main() { fmt.Println(pkg.Value) }\n")
			licensePath := filepath.Join(dep, "LICENSE")
			if mode == "vendor" {
				run(t, app, "go", "mod", "vendor")
				licensePath = filepath.Join(app, "vendor", "example.com", "dep", "LICENSE")
			}
			opt := driver.Options{Dir: app, Patterns: []string{"."}, Out: filepath.Join(dir, "out"), TrimPath: true}
			if err := driver.Build(opt); err != nil {
				t.Fatal(err)
			}
			read := func(name string) string {
				t.Helper()
				data, err := os.ReadFile(filepath.Join(opt.Out, name))
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
			for _, want := range []string{"Application license", "Dependency license version 1", "Dependency attribution", "Copyright 2026 Dependency Authors.", "The Go Authors"} {
				if !strings.Contains(read("LICENSE"), want) {
					t.Errorf("output omits %q", want)
				}
			}
			if strings.Contains(read("LICENSE"), dir) {
				t.Fatal("output notice contains an absolute checkout path")
			}
			if !strings.Contains(read("goruntime/LICENSE"), "Copyright (c) 2026, Eddy Wilson") {
				t.Fatal("embedded runtime omits its license")
			}
			if !strings.Contains(read("goruntime/LICENSE"), "Copyright 2009 The Go Authors.") {
				t.Fatal("embedded runtime omits its Go source license")
			}
			writeFile(t, licensePath, "Dependency license version 2\n")
			stats, err := driver.BuildWithStats(opt)
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Compiled) != 0 {
				t.Fatalf("a notice-only change recompiled %v", stats.Compiled)
			}
			if text := read("LICENSE"); !strings.Contains(text, "Dependency license version 2") || strings.Contains(text, "Dependency license version 1") {
				t.Fatal("cached build retained stale notices")
			}
			writeFile(t, filepath.Join(app, "main.go"), "package main\nfunc main() {}\n")
			if err := driver.Build(opt); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(read("LICENSE"), "Dependency license") {
				t.Fatal("removed dependency retained its notices")
			}

		})
	}
}
