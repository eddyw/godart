package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLegacyTestingMain(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/legacymain\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "testing")
func match(pattern, name string) (bool, error) {
 return pattern == "" || pattern == "chosen" && name == "TestKeep" || pattern == "failure" && name == "TestFailure", nil
}
func main() {
 defer fmt.Println("unexpected defer")
 testing.Main(match, []testing.InternalTest{
  {"TestKeep", func(t *testing.T) { fmt.Println("kept") }},
  {"TestReject", func(t *testing.T) { panic("filter ignored") }},
  {"TestFailure", func(t *testing.T) { t.Fail() }},
 }, nil, nil)
 fmt.Println("unexpected return")
}
`)
	native := filepath.Join(dir, "native")
	run(t, dir, "go", "build", "-o", native, ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, filter := range []string{"chosen", "failure"} {
		t.Run(filter, func(t *testing.T) {
			args := []string{"-test.run=" + filter}
			want := programOutcome(t, dir, native, args...)
			for _, target := range []string{"vm", "aot", "js", "wasm"} {
				t.Run(target, func(t *testing.T) {
					var got programResult
					if target == "vm" {
						got = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, "bin/main.dart")}, args...)...)
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, args)
					}
					if got.Exit != want.Exit || normalize(got.Text) != normalize(want.Text) {
						t.Fatalf("Go (%d):\n%s\nGart (%d):\n%s", want.Exit, want.Text, got.Exit, got.Text)
					}
				})
			}
		})
	}
}
