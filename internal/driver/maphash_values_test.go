package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestMaphashValues(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/maphashvalues\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
 "fmt"
 "hash/maphash"
 "runtime"
)
func main() {
 values := []any{[]byte("s"), map[string]int(nil), (func())(nil), struct{ S []byte }{}, [0][]byte{}, struct{ I any }{struct{ S []byte }{}}, [0]int{}, [1]int{1}, struct{ N int }{1}, nil}
 for operation := 0; operation < 2; operation++ {
  for i, value := range values {
   func() {
    defer func() {
     e := recover()
     if e == nil { fmt.Println(operation, i, "no panic"); return }
     if err, ok := e.(runtime.Error); ok { fmt.Println(operation, i, err.Error()) } else { fmt.Printf("%d %d %T %v\n", operation, i, e, e) }
    }()
    seed := maphash.MakeSeed()
    if operation == 0 { maphash.Comparable[any](seed, value) } else {
     var h maphash.Hash
     h.SetSeed(seed)
     maphash.WriteComparable[any](&h, value)
    }
   }()
  }
 }
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if want.Exit != 0 || got.Exit != 0 || want.Text != got.Text {
				t.Fatalf("Go:\n%+v\nGart:\n%+v", want, got)
			}
		})
	}
}
