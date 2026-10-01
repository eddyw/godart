package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"path/filepath"
	"testing"
)

func TestFunctionReceiverSuspension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/functionreceiver\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("flag"; "fmt")
type callback func() int
func (f callback) Run() int { return f() }
type runner interface { Run() int }
func invoke(r runner) int { return r.Run() }
func receive(value int) int {
 ch := make(chan int)
 go func() { ch <- value }()
 return <-ch
}
func main() {
 f := callback(func() int { return receive(7) })
 fmt.Println("interface", invoke(f))
 var known runner = f
 fmt.Println("known interface", known.Run())
 fs := flag.NewFlagSet("callbacks", flag.ContinueOnError)
 fs.BoolFunc("b", "", func(s string) error { fmt.Println("bool", s, receive(11)); return nil })
 fs.Func("f", "", func(s string) error { fmt.Println("func", s, receive(13)); return nil })
 if err := fs.Parse([]string{"-b", "-b=false", "-f=value"}); err != nil { panic(err) }
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
