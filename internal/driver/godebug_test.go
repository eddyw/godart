package driver_test

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"golang.org/x/tools/txtar"
)

func TestGODEBUGUpstreamDirectiveErrors(t *testing.T) {
	for _, name := range []string{"godebug_unknown.txt", "godebug_default.txt", "godebug_removed.txt", "build_repeated_godebug_issue62346.txt"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/cmd/go/testdata/script", name))
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(source)
			t.Logf("upstream=%s Go=%s SHA256=%x", name, runtime.Version(), sha256.Sum256(source))
			dir := t.TempDir()
			for _, file := range archive.Files {
				if !filepath.IsLocal(file.Name) {
					t.Fatalf("nonlocal upstream fixture: %s", file.Name)
				}
				path := filepath.Join(dir, file.Name)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, string(file.Data))
			}
			license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			var tags []string
			patterns := []string{"."}
			if name == "build_repeated_godebug_issue62346.txt" {
				patterns = []string{"file.go"}
			}
			if name == "godebug_default.txt" || name == "godebug_removed.txt" {
				module := "go.mod.20"
				if name == "godebug_default.txt" {
					module = "go.mod.21"
					tags = []string{"godebugbad"}
				} else {
					patterns = []string{"godebugold.go"}
				}
				data, err := os.ReadFile(filepath.Join(dir, module))
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(dir, "go.mod"), string(data))
			}
			var assertion string
			for _, line := range strings.Split(string(archive.Comment), "\n") {
				if strings.HasPrefix(line, "stderr '") && (strings.Contains(line, "//go:debug") || strings.Contains(line, "removed GODEBUG")) {
					assertion = strings.TrimSuffix(strings.TrimPrefix(line, "stderr '"), "'")
					break
				}
			}
			if assertion == "" {
				t.Fatal("missing original diagnostic assertion")
			}
			want := regexp.MustCompile(assertion)
			args := []string{"build", "-o", filepath.Join(t.TempDir(), "native")}
			if len(tags) != 0 {
				args = append(args, "-tags="+strings.Join(tags, ","))
			}
			native := programOutcome(t, dir, "go", append(args, patterns...)...)
			if native.Exit == 0 || !want.MatchString(native.Text) {
				t.Fatalf("native result does not satisfy original assertion %q: %+v", assertion, native)
			}
			err = driver.Build(driver.Options{Dir: dir, Patterns: patterns, Out: t.TempDir(), Runtime: runtimeDir(t), BuildTags: tags})
			if err == nil || !want.MatchString(err.Error()) {
				t.Fatalf("Gart result does not satisfy original assertion %q: %v", assertion, err)
			}
			if name == "build_repeated_godebug_issue62346.txt" {
				for _, line := range strings.Split(string(archive.Comment), "\n") {
					if strings.HasPrefix(line, "! stderr '") {
						pattern := strings.TrimSuffix(strings.TrimPrefix(line, "! stderr '"), "'")
						if forbidden := regexp.MustCompile(pattern); forbidden.MatchString(native.Text) || forbidden.MatchString(err.Error()) {
							t.Fatalf("a compiler violates original negative assertion %q", pattern)
						}
					}
				}
			}
			t.Logf("both compilers satisfy %q", assertion)
		})
	}
}

func TestUnsupportedFIPSMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fipsmode\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"crypto/fips140"
	"fmt"
)
func main() { fmt.Println(fips140.Enabled(), fips140.Enforced()) }
`)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, setting := range []string{"", "off", "on", "only", "debug"} {
		t.Run(setting, func(t *testing.T) {
			t.Setenv("GODEBUG", "fips140="+setting)
			result := programOutcome(t, out, "dart", "bin/main.dart")
			if setting == "" || setting == "off" {
				if result.Exit != 0 || result.Text != "false false" {
					t.Fatalf("default crypto failed: %+v", result)
				}
			} else if result.Exit == 0 || !strings.Contains(result.Text, "Dart does not support FIPS 140-3 mode") {
				t.Fatalf("unsupported FIPS mode did not fail explicitly: %+v", result)
			}
		})
	}
}

func TestGODEBUGUpstreamDefaults(t *testing.T) {
	t.Setenv("GODEBUG", "")
	t.Setenv("GOWORK", "auto")
	source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/cmd/go/testdata/script/godebug_default.txt"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files := map[string]string{}
	for _, file := range txtar.Parse(source).Files {
		if !filepath.IsLocal(file.Name) {
			t.Fatalf("nonlocal upstream fixture: %s", file.Name)
		}
		files[file.Name] = string(file.Data)
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, file.Name)), 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, file.Name), string(file.Data))
	}
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	out := t.TempDir()
	native := filepath.Join(t.TempDir(), "native")
	for _, c := range []struct {
		name, module, workspace string
		tags                    []string
	}{
		{name: "module20", module: "go.mod.20"},
		{name: "module21", module: "go.mod.21"},
		{name: "workspace21", module: "go.mod.20", workspace: "go.work.21"},
		{name: "directive20", module: "go.mod.21", tags: []string{"godebug"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			writeFile(t, filepath.Join(dir, "go.mod"), files[c.module])
			if c.workspace != "" {
				writeFile(t, filepath.Join(dir, "go.work"), files[c.workspace])
			} else if err := os.Remove(filepath.Join(dir, "go.work")); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			args := []string{"build", "-o", native}
			if len(c.tags) > 0 {
				args = append(args, "-tags="+strings.Join(c.tags, ","))
			}
			run(t, dir, "go", append(args, ".")...)
			want := programOutcome(t, dir, native)
			opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), BuildTags: c.tags}
			if err := driver.Build(opt); err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			got := programOutcome(t, out, "dart", "bin/main.dart")
			if want.Exit != 2 || got != want {
				t.Fatalf("upstream default behavior: Gart=%+v Go=%+v", got, want)
			}
			if c.name == "directive20" {
				app := filepath.Join(out, "app")
				run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", app)
				if got := programOutcome(t, out, app); got != want {
					t.Fatalf("AOT default behavior: Gart=%+v Go=%+v", got, want)
				}
			}
		})
	}
}

func TestGODEBUGDefaultCache(t *testing.T) {
	t.Setenv("GODEBUG", "")
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "math/rand"; "os")
func show() {
 defer func() { fmt.Printf("%T\n", recover()) }()
 panic(nil)
}
func main() {
 show()
 for _, setting := range []string{"panicnil=0", "panicnil=1", "", "panicnil=invalid", ""} {
  os.Setenv("GODEBUG", setting)
  show()
 }
 rand.Seed(17)
 fmt.Println(rand.Int63())
}
`)
	opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}
	native := filepath.Join(t.TempDir(), "native")
	for _, value := range []string{"1", "0", "1"} {
		writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/defaultcache\n\ngo 1.27\n\ngodebug (\n panicnil="+value+"\n randseednop=0\n randautoseed=0\n)\n")
		run(t, dir, "go", "build", "-o", native, ".")
		want := programOutcome(t, dir, native)
		stats, err := driver.BuildWithStats(opt)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(stats.Compiled, "example.com/defaultcache") {
			t.Fatal("changed GODEBUG defaults reused the main package")
		}
		prepareDart(t, out)
		got := programOutcome(t, out, "dart", "bin/main.dart")
		if want.Exit != 0 || got != want {
			t.Fatalf("default override %s: Gart=%+v Go=%+v", value, got, want)
		}
	}
}

func TestGODEBUGTestMainDefault(t *testing.T) {
	t.Setenv("GODEBUG", "")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/testdefault\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main_test.go"), `//go:debug panicnil=1
package main_test
import "testing"
func TestDefault(t *testing.T) {
 defer func() { if value := recover(); value != nil { t.Fatal(value) } }()
 panic(nil)
}
`)
	run(t, dir, "go", "test", "-count=1", ".")
	got := gartTest(t, dir, driver.TestOptions{Verbose: true})
	if !strings.Contains(got, "--- PASS: TestDefault") {
		t.Fatal(got)
	}
}

func TestPanicNilEnvironment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/panicnil\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"fmt"
	"os"
)
func show() {
	defer func() { fmt.Printf("%T\n", recover()) }()
	panic(nil)
}
func main() {
	show()
	for _, value := range []string{"panicnil=1", "panicnil=invalid", "panicnil=0", "panicnil=2147483648", "panicnil=+1", "panicnil= 1", ""} {
		os.Setenv("GODEBUG", value)
		show()
	}
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, value := range []string{"", "panicnil=0", "panicnil=1", "panicnil=1,panicnil=0", "panicnil=0,panicnil=1", "panicnil=1,panicnil=invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("GODEBUG", value)
			want := programOutcome(t, dir, native)
			got := programOutcome(t, out, "dart", "bin/main.dart")
			if want.Exit != 0 || got != want {
				t.Fatalf("panic(nil): Gart=%+v Go=%+v", got, want)
			}
		})
	}
}
