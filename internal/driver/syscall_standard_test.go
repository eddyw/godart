package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestSyscallStandardDescriptors(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/standarddescriptors\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "syscall")
func main() {
 fmt.Printf("%T %T %T %d %d %d\n", syscall.Stdin, syscall.Stdout, syscall.Stderr, syscall.Stdin, syscall.Stdout, syscall.Stderr)
 descriptors := []*int{&syscall.Stdin, &syscall.Stdout, &syscall.Stderr}
 for i, descriptor := range descriptors { *descriptor = 11+i }
 fmt.Println(syscall.Stdin, syscall.Stdout, syscall.Stderr)
 syscall.Stdin, syscall.Stdout, syscall.Stderr = 0, 1, 2
 fmt.Println(syscall.Stdin, syscall.Stdout, syscall.Stderr)
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
				t.Fatalf("Go: %+v\nGart: %+v", want, got)
			}
		})
	}
}
