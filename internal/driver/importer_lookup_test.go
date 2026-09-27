package driver_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestImporterLookup(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "export.go"), `package sample
 type Pair[T any] struct { First, Second T }
 type Counter int
 func (c Counter) Add(n Counter) Counter { return c+n }
 const Answer = 42
 `)
	runProgram(t, dir, "go", "tool", "compile", "-p", "example.com/sample", "-o", "sample.a", "export.go")
	raw, err := os.ReadFile(filepath.Join(dir, "sample.a"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "export.go")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/lookup\n\ngo 1.27\n")
	source := `package main
 import ("bytes"; "encoding/base64"; "fmt"; "go/importer"; "go/token"; "io")
 type reader struct { *bytes.Reader; closed *int }
 func (r reader) Close() error { *r.closed++; return nil }
 func main() {
  data,err:=base64.StdEncoding.DecodeString(ARCHIVE); if err!=nil {panic(err)}
  calls,closed:=0,0
  lookup:=func(path string)(io.ReadCloser,error) {
   calls++;fmt.Println("lookup",path)
   return reader{bytes.NewReader(data),&closed},nil
  }
  imp:=importer.ForCompiler(token.NewFileSet(),"gc",lookup)
  pkg,err:=imp.Import("example.com/sample");if err!=nil {panic(err)}
  again,err:=imp.Import("example.com/sample");if err!=nil {panic(err)}
  fmt.Println(pkg.Path(),pkg.Complete(),pkg==again,calls,closed)
  for _,name:=range pkg.Scope().Names() { fmt.Println(pkg.Scope().Lookup(name)) }
  deny:=importer.ForCompiler(token.NewFileSet(),"gc",func(string)(io.ReadCloser,error){return nil,fmt.Errorf("lookup denied")})
  _,err=deny.Import("example.com/denied");fmt.Println("denied",err)
  broken:=importer.ForCompiler(token.NewFileSet(),"gc",func(string)(io.ReadCloser,error){return reader{bytes.NewReader([]byte("bad archive")),&closed},nil})
  _,err=broken.Import("example.com/broken");fmt.Println("broken",err!=nil,closed)
 }
 `
	source = strings.Replace(source, "ARCHIVE", strconv.Quote(base64.StdEncoding.EncodeToString(raw)), 1)
	writeFile(t, filepath.Join(dir, "main.go"), source)
	want := runProgram(t, dir, "go", "run", ".")
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
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("Go:\n%s\nGart:\n%+v", want, got)
			}
		})
	}
}
