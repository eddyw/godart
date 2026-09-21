package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestFrameTupleLifetime(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/framelifetime\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "runtime"; "time"; "weak")
type item struct { data [256]byte }
func makeItem() (*item, error) {
 runtime.Gosched()
 return &item{}, nil
}
func main() {
 p, err := makeItem()
 if err != nil { panic(err) }
 w := weak.Make(p)
 runtime.GC()
 if w.Value() != p { panic("lost live pointer") }
 runtime.KeepAlive(p)
 p = nil
 deadline := time.Now().Add(4*time.Second)
 for w.Value() != nil {
  if time.Now().After(deadline) { panic("tuple retained pointer") }
  runtime.GC()
  runtime.Gosched()
 }
 fmt.Println("released tuple pointer")
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	if got := runProgram(t, dir, "dart", filepath.Join(out, "bin/main.dart")); got != want {
		t.Fatalf("Go:\n%s\nGart:\n%s", want, got)
	}
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runLifetimeTarget(t, target, out, "bin/main.dart")
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
