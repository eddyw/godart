package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestNativeSourceCallFrames(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/sourcepositionsequence\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"path/filepath";"runtime")
func site(label string, depth int) string {
 _,file,line,ok:=runtime.Caller(depth)
 return fmt.Sprintf("%s %s:%d %t",label,filepath.Base(file),line,ok)
}
func wrap(value string) string { return value+" | "+site("outer",2) }
func stackDepth(label string, depth int) string {
 pc,file,line,ok := runtime.Caller(depth)
 name := ""
 if ok { name = runtime.FuncForPC(pc).Name() }
 if filepath.Base(file)=="main.go" { return fmt.Sprintf("%s %s %s:%d %t",label,name,filepath.Base(file),line,ok) }
 return fmt.Sprintf("%s %s %t",label,name,ok)
}
func hold(text string) string { return text }
var step int
func condition() bool {
 fmt.Println(site("condition",2))
 step++
 return step<3
}
func main() {
 value := 0; value++; fmt.Println(value, site("same-line",1))
 for i := 0; i < 2; i++ { value++; fmt.Println(value, site("loop-line",1)) }
 if value == 3 { value++; fmt.Println(value, site("branch-line",1)) }
 ch := make(chan int, 1); ch <- value; value = <-ch; fmt.Println(value, site("resume-line",1))
 fmt.Println(stackDepth("depth-single",2))
 fmt.Println(
  hold(
   stackDepth("depth-nested",2)))
 fmt.Println(site("single",1))
 fmt.Println(
  wrap(site("nested",1)))
 values:=struct{A,B string}{
  site("first",1),
  site("second",1),
 }
 fmt.Println(values)
 for condition() {
  fmt.Println(site("body",1))
 }
 goto label
label:
 fmt.Println(site("label",1))
//line virtual.go:100
 fmt.Println(site("directive",1))
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
