package driver_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGOROOTEnvironment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/goroot\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
    "fmt"
    "go/build"
    "os"
    "runtime"
)
func main() {
    original := runtime.GOROOT()
    fmt.Printf("%q %q\n", original, build.Default.GOROOT)
    os.Setenv("GOROOT", "/changed")
    fmt.Println(runtime.GOROOT() == original)
    os.Unsetenv("GOROOT")
    fmt.Println(runtime.GOROOT() == original)
}
`)
	out := t.TempDir()
	var base []string
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "GOROOT=") {
			base = append(base, item)
		}
	}
	for round, trimpath := range []bool{false, true, false} {
		t.Run(fmt.Sprintf("trimpath=%t", trimpath), func(t *testing.T) {
			native := filepath.Join(t.TempDir(), "native")
			args := []string{"build", "-o", native}
			if trimpath {
				args = append(args, "-trimpath")
			}
			runProgram(t, dir, "go", append(args, ".")...)
			opt := driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false, TrimPath: trimpath}
			if err := driver.Build(opt); err != nil {
				t.Fatal(err)
			}
			warm, err := driver.BuildWithStats(opt)
			if err != nil || !slices.Contains(warm.Cached, "runtime") {
				t.Fatalf("runtime cache: stats=%+v err=%v", warm, err)
			}
			prepareDart(t, out)
			targets := []string{"vm", "aot", "js", "wasm"}
			if round == 2 {
				targets = []string{"vm"}
			}
			for _, target := range targets {
				t.Run(target, func(t *testing.T) {
					for i, item := range []struct {
						name string
						env  []string
					}{
						{"unset", base},
						{"empty", append(slices.Clone(base), "GOROOT=")},
						{"override", append(slices.Clone(base), "GOROOT="+t.TempDir())},
					} {
						t.Run(item.name, func(t *testing.T) {
							want := programOutcomeWithLimitEnv(t, dir, item.env, runLimit, native)
							var got programResult
							if target == "vm" {
								got = programOutcomeWithLimitEnv(t, dir, item.env, runLimit, "dart", filepath.Join(out, "bin/main.dart"))
							} else if i == 0 {
								got = runUpstreamTargetEnv(t, target, out, "bin/main.dart", dir, []string{}, item.env)
							} else {
								switch target {
								case "aot":
									got = programOutcomeWithLimitEnv(t, dir, item.env, runLimit, filepath.Join(out, "tests.exe"))
								case "js":
									got = programOutcomeWithLimitEnv(t, dir, item.env, runLimit, "node", filepath.Join(out, "run.js"))
								case "wasm":
									got = programOutcomeWithLimitEnv(t, dir, item.env, runLimit, "node", filepath.Join(out, "run.mjs"))
								}
							}
							if want.Exit != 0 || got.Exit != 0 || got.Text != want.Text {
								t.Fatalf("Go:\n%+v\nGart:\n%+v", want, got)
							}
						})
					}
				})
			}
		})
	}
}
