package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLargeSliceLiteral(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module largeliteral\n\ngo 1.27\n")
	source := `package main
import "fmt"
func values[T any](value T) []T { return []T{GENERIC} }
func main() {
	value := complex(3.0, -4.0)
	items := []complex128{COMPLEX}
	value = 9
	fmt.Println(len(items), cap(items), items[0], items[127], items[128], items[256], value)
	var b byte = 255
	bytes := []byte{BYTE}
	b = 0
	fmt.Println(len(bytes), cap(bytes), bytes[0], bytes[127], bytes[128], bytes[256], b)
	generic := values("kept")
	fmt.Println(len(generic), cap(generic), generic[0], generic[127], generic[128], generic[256])
}
`
	source = strings.NewReplacer(
		"GENERIC", strings.Repeat("value,", 257),
		"COMPLEX", strings.Repeat("-value,", 257),
		"BYTE", strings.Repeat("b,", 257),
	).Replace(source)
	writeFile(t, filepath.Join(dir, "main.go"), source)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native fixture failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, out, "dart", "bin/main.dart")
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%+v\nDart:\n%+v", want, got)
			}
		})
	}
}
