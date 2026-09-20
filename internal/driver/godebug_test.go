package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

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
