package load

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestCopyGoSDK(t *testing.T) {
	source, cache := t.TempDir(), t.TempDir()
	for name, content := range map[string]string{"bin/go": "SDK executable", "LICENSE": "SDK license", "src/fmt/print.go": "package fmt"} {
		path := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0555); err != nil {
			t.Fatal(err)
		}
	}
	var group sync.WaitGroup
	paths := make([]string, 4)
	for i := range paths {
		group.Go(func() {
			var err error
			paths[i], err = copyGoSDK(source, "go1.27.0", cache)
			if err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	for _, path := range paths {
		if path == "" || path != paths[0] {
			t.Fatalf("concurrent SDK copies differ: %v", paths)
		}
	}
	for _, name := range []string{"bin/go", "LICENSE", "src/fmt/print.go"} {
		want, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(paths[0], name))
		if err != nil || string(got) != string(want) {
			t.Fatalf("copied %s: %q, %v", name, got, err)
		}
	}
	executable, err := os.Stat(filepath.Join(paths[0], "bin", "go1.27.0"))
	if err != nil || executable.Mode()&0111 == 0 {
		t.Fatalf("toolchain dispatcher lost execute permission: %v", err)
	}
	again, err := copyGoSDK(source, "go1.27.0", cache)
	if err != nil || again != paths[0] {
		t.Fatalf("SDK cache reuse: %q, %v", again, err)
	}
	if _, err := os.Stat(filepath.Join(source, "bin", "go1.27.0")); !os.IsNotExist(err) {
		t.Fatalf("the original SDK changed: %v", err)
	}
}

func TestCopyGoSDKFailure(t *testing.T) {
	cache := t.TempDir()
	if _, err := copyGoSDK(t.TempDir(), "go1.27.0", cache); err == nil {
		t.Fatal("accepted an SDK without a Go executable")
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed SDK copy left published files: %v, %v", entries, err)
	}
	for _, version := range []string{"", "../go1.27.0", "go1.27.0/path"} {
		if _, err := copyGoSDK(t.TempDir(), version, cache); err == nil {
			t.Fatalf("accepted invalid version %q", version)
		}
	}
}

func TestLoadWithSDKInModuleCache(t *testing.T) {
	moduleCache := t.TempDir()
	source := filepath.Join(moduleCache, "toolchain")
	if err := os.CopyFS(source, os.DirFS(runtime.GOROOT())); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOROOT", source)
	t.Setenv("GOMODCACHE", moduleCache)
	t.Setenv("GOTOOLCHAIN", "local")
	dir := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":       "module sdkprobe\n\ngo " + strings.TrimPrefix(runtime.Version(), "go") + "\n",
		"main.go":      "package main\nimport \"fmt\"\nfunc main() { fmt.Println(1) }\n",
		"main_test.go": "package main\nimport \"testing\"\nfunc TestMainCode(t *testing.T) { main() }\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	sdk, err := prepareGoSDK(dir)
	if err != nil {
		t.Fatal(err)
	}
	if sdk.root == source {
		t.Fatal("the SDK remains under the module cache")
	}
	defer os.RemoveAll(sdk.root)
	cmd := exec.Command("go", "env", "GOROOT", "GOVERSION")
	cmd.Dir, cmd.Env = dir, sdk.env
	text, err := cmd.CombinedOutput()
	if err != nil || string(text) != sdk.root+"\n"+runtime.Version()+"\n" {
		t.Fatalf("selected SDK: %s, %v", text, err)
	}
	result, err := LoadWithOptions(Options{Dir: dir, Patterns: []string{"."}, Tests: true, Cover: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Binaries) != 1 || len(result.Binaries[0].Cover) != 1 {
		t.Fatal("SDK test or coverage source is absent")
	}
	if result.GOROOT != source {
		t.Fatalf("build SDK root: %q, want %q", result.GOROOT, source)
	}
	for name := range result.Sources {
		if rel, err := filepath.Rel(moduleCache, name); err == nil && filepath.IsLocal(rel) {
			t.Fatalf("overlay remains in the module cache: %s", name)
		}
	}
	if os.Getenv("GOROOT") != source || os.Getenv("GOTOOLCHAIN") != "local" {
		t.Fatal("the loader changed the process environment")
	}
}
