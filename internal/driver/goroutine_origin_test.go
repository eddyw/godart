package driver_test

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRuntimeGoroutineOrigins(t *testing.T) {
	const program = `package main
import ("fmt"; "runtime"; "strings")
func snapshot(all bool) string {
 b := make([]byte, 65536)
 return string(b[:runtime.Stack(b, all)])
}
func id() string { return strings.Fields(snapshot(false))[1] }
func inspect(stack, parent, file string, line int) {
 want := fmt.Sprintf("created by main.spawn in goroutine %s\n\t%s:%d", parent, file, line)
 if !strings.Contains(stack, want) { panic(fmt.Sprintf("missing origin %q in %s", want, stack)) }
}
func wait(resume <-chan bool, ready chan<- string, child string) { ready <- child; <-resume }
func spawn() {
 parent := id()
 ready := make(chan string)
 resume := make(chan bool)
 done := make(chan bool)
 _, file, line, _ := runtime.Caller(0)
 go func() {
  child := id()
  if child == parent { panic("child has parent identity") }
  inspect(snapshot(false), parent, file, line+1)
  wait(resume, ready, child)
  inspect(snapshot(false), parent, file, line+1)
  close(done)
 }()
 child := <-ready
 all := snapshot(true)
 start := strings.Index(all, "goroutine "+child+" [")
 if start < 0 { panic("missing child goroutine") }
 block, _, _ := strings.Cut(all[start:], "\n\n")
 inspect(block, parent, file, line+1)
 inner, outer := strings.Index(block, "main.wait("), strings.Index(block, "main.spawn.func1(")
 if inner < 0 || outer < 0 || inner >= outer { panic("incorrect parked frame order: "+block) }
 resume <- true
 <-done
}
func main() {
 spawn()
 done := make(chan bool)
 go func() { spawn(); close(done) }()
 <-done
 fmt.Println("origin records match")
}
`
	for _, mode := range []struct {
		name string
		trim bool
	}{{"absolute", false}, {"trimmed", true}} {
		t.Run(mode.name, func(t *testing.T) {
			dir, out := t.TempDir(), t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/goroutineorigins\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "main.go"), program)
			native := filepath.Join(t.TempDir(), "native")
			run(t, dir, "go", "build", "-trimpath="+strconv.FormatBool(mode.trim), "-o", native, ".")
			want := programReference(t, dir, native)
			if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: mode.trim}); err != nil {
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
						t.Fatalf("goroutine origin output: got %+v, want %+v", got, want)
					}
				})
			}
		})
	}
}
