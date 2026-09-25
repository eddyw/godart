package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestBuiltinPrintBytes(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/printbytes\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
 type text string
 func show[T ~string](s T) { print(s); println(s) }
 func main() {
  raw := make([]byte,256)
  for i := range raw { raw[i]=byte(i) }
  s := string(raw)+"€𐀀"
  print(); println()
  print(s); println(text(s))
  show(s); show(text(s))
  println("end")
 }
 `)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
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
			if got != want {
				t.Fatalf("Go: exit=%d bytes=%x\nGart: exit=%d bytes=%x", want.Exit, want.Text, got.Exit, got.Text)
			}
			t.Logf("matching output bytes=%d", len(got.Text))
		})
	}
}
