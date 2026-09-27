package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNamedResultsAcrossSuspension(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/namedresults\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "runtime")

func number(wait bool) (n int) {
 defer func() { fmt.Println("number", n) }()
 if wait { runtime.Gosched() }
 return 42
}

func values() (int,string) { return 7,"value" }
func tuple() (n int,text string) {
 defer func() { fmt.Println("tuple", n, text) }()
 runtime.Gosched()
 return values()
}

type counter struct { value int }
func (c counter) read() (n int) {
 defer func() { fmt.Println("method", n) }()
 runtime.Gosched()
 return c.value
}

func retained() (n int, observe func() int) {
 read:=func() int { return n }
 runtime.Gosched()
 return 9,read
}

func main() {
 fmt.Println("plain",number(false))
 fmt.Println("resumed",number(true))
 fmt.Println(tuple())
 fmt.Println(counter{11}.read())
 literal:=func() (text string) {
  defer func() { fmt.Println("literal",text) }()
  runtime.Gosched()
  return "result"
 }
 fmt.Println(literal())
 n,observe:=retained()
 fmt.Println("retained",n,observe())
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, out, "dart", "bin/main.dart")
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result.Exit != 0 || result.Text != want {
				t.Fatalf("Go:\n%s\nGart (%s, exit %d):\n%s", want, target, result.Exit, result.Text)
			}
		})
	}
}
