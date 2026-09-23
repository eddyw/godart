package driver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
)

func TestTrimPathAndRelocatedCache(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	for _, dir := range []string{first, second} {
		writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/paths\n\ngo 1.27\n")
		writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() { type Local int; var value any = Local(1); println(value) }\n//line virtual.go:70\nfunc other() { println(2) }\n")
	}
	opt := driver.Options{Dir: first, Patterns: []string{"./..."}, Out: t.TempDir()}
	build := func() (*driver.Stats, string) {
		t.Helper()
		stats, err := driver.BuildWithStats(opt)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(opt.Out, "bin", "main.map.dart"))
		if err != nil {
			t.Fatal(err)
		}
		return stats, string(data)
	}
	_, absolute := build()
	if !strings.Contains(absolute, filepath.Join(first, "main.go")) || !strings.Contains(absolute, "virtual.go") {
		t.Fatalf("source paths missing: %s", absolute)
	}
	warm, _ := build()
	if len(warm.Compiled) != 0 {
		t.Fatalf("warm build compiled %v", warm.Compiled)
	}
	opt.Dir = second
	moved, movedMap := build()
	if !slices.Contains(moved.Compiled, "example.com/paths") || !strings.Contains(movedMap, filepath.Join(second, "main.go")) {
		t.Fatalf("relocated cache retained old paths: %s", movedMap)
	}
	opt.TrimPath = true
	trimmed, short := build()
	if len(trimmed.Compiled) == 0 || strings.Contains(short, second) || !strings.Contains(short, "example.com/paths/main.go") {
		t.Fatalf("trimpath retained absolute paths: %s", short)
	}
	trimmedCode, err := os.ReadFile(filepath.Join(opt.Out, "bin", "main.dart"))
	if err != nil {
		t.Fatal(err)
	}
	opt.Dir = first
	cached, same := build()
	if len(cached.Compiled) != 0 || same != short {
		t.Fatal("trimmed cache depends on the checkout")
	}
	opt.Out = t.TempDir()
	_, rebuilt := build()
	if rebuilt != short {
		t.Fatal("trimmed source maps differ across checkouts")
	}
	rebuiltCode, err := os.ReadFile(filepath.Join(opt.Out, "bin", "main.dart"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(trimmedCode, rebuiltCode) {
		t.Fatal("trimmed code differs across checkouts")
	}
	manifest, err := os.ReadFile(filepath.Join(opt.Out, "pubspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "path: goruntime\n") {
		t.Fatalf("runtime path is not portable: %s", manifest)
	}
	if _, err := os.Stat(filepath.Join(opt.Out, "goruntime", "lib", "goruntime.dart")); err != nil {
		t.Fatal(err)
	}
	opt.TrimPath = false
	_, restored := build()
	if !strings.Contains(restored, filepath.Join(first, "main.go")) {
		t.Fatal("default paths did not return")
	}
}

func TestRunKernelPaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/kernelpaths\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() {\npanic(\"failure\")\n}\n")
	compiles := 0
	opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: t.TempDir(), Trace: func(phase string, _ time.Duration) {
		if phase == "kernel-compile" {
			compiles++
		}
	}}
	for i, trim := range []bool{false, false, true} {
		opt.TrimPath = trim
		var output bytes.Buffer
		code, err := driver.Run(opt, nil, &output)
		if err != nil || code != 2 {
			t.Fatalf("run %d: exit %d, error %v, output %s", i, code, err, output.String())
		}
		path := filepath.Join(dir, "main.go")
		if trim {
			path = "example.com/kernelpaths/main.go"
		}
		if !strings.Contains(output.String(), path+":3") {
			t.Fatalf("missing source path %s: %s", path, output.String())
		}
		if trim && strings.Contains(output.String(), dir) {
			t.Fatal("trimmed kernel retained an absolute Go path")
		}
		want := 1
		if trim {
			want = 2
		}
		if compiles != want {
			t.Fatalf("run %d: compiled %d kernels, want %d", i, compiles, want)
		}
	}
}
