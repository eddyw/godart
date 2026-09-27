package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGoImporter(t *testing.T) {
	t.Setenv("GOROOT", build.Default.GOROOT)
	dir, out := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sample"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module importerprobe\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "sample/sample.go"), "package sample\ntype Number int\nconst Value Number = 42\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("fmt"; "go/importer"; "go/token"; "go/types")
func main() {
 for _, imp := range []types.Importer{importer.Default(), importer.For("gc", nil), importer.ForCompiler(token.NewFileSet(), "gc", nil)} {
  pkg,err:=imp.Import("math/big");if err!=nil {panic(err)}
  fmt.Println(pkg.Path(),pkg.Scope().Lookup("Int").Name(),pkg.Complete())
 }
 pkg,err:=importer.ForCompiler(token.NewFileSet(),"source",nil).Import("./sample")
 if err!=nil {panic(err)}
 fmt.Println(pkg.Name(),pkg.Scope().Lookup("Value").(*types.Const).Val(),pkg.Complete())
 fmt.Println(importer.ForCompiler(token.NewFileSet(),"invalid",nil)==nil)
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("Go:\n%s\nGart:\n%+v", want, got)
			}
		})
	}
}
