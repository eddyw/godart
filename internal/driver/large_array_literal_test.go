package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLargeArrayLiteral(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module largearray\n\ngo 1.27\n")
	source := `package main
import "fmt"
type item struct { n int; a [2]int }
func values[T any](value T) [257]T { return [257]T{GENERIC} }
var table = [...]byte{TABLE}
var counter int
func next() int { counter++; return counter }
func main() {
 var b byte = 255
 bytes := [300]byte{BYTE}
 b = 0
 fmt.Println(len(bytes), bytes[0], bytes[127], bytes[128], bytes[256], bytes[257], bytes[299], b)
 value := item{7, [2]int{11, 13}}
 items := [300]item{STRUCT}
 value.n = 9
 copy := items
 items[128].n = 17
 items[128].a[1] = 19
 fmt.Println(items[0], items[128], copy[128], items[257], value)
 generic := values("kept")
 genericCopy := generic
 generic[128] = "changed"
 fmt.Println(len(generic), generic[0], generic[127], generic[128], generic[256], genericCopy[128])
 calls := [257]int{CALL}
 fmt.Println(counter, calls[0], calls[127], calls[128], calls[256])
 keyed := [300]int{0: 3, 128: 5, 299: 7}
 fmt.Println(keyed[0], keyed[127], keyed[128], keyed[299])
 fmt.Println(len(table), table[0], table[127], table[128], table[1024])
}
`
	source = strings.NewReplacer("GENERIC", strings.Repeat("value,", 257), "BYTE", strings.Repeat("b,", 257), "STRUCT", strings.Repeat("value,", 257), "CALL", strings.Repeat("next(),", 257), "TABLE", strings.Repeat("173,", 1025)).Replace(source)
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
