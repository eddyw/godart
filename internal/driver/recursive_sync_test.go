package driver_test

import (
	"fmt"
	"go/build"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestSyncRecursiveHostCallbacks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module synchost\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "reflect"
//gart:sync
type Callback func() bool
//gart:dart package:synchost/host.dart invoke
func invoke() bool
func walk(n int) int {
 if n == 0 { return 0 }
 return 1 + walk(n-1)
}
func fail(n int) {
 if n == 0 { panic("deep") }
 fail(n-1)
}
func Exported() Callback {
 return func() bool {
  return walk(10000) == 10000 && invoke()
 }
}
func Inner() Callback { return func() bool { return walk(20000) == 20000 } }
func Reflected(index int) Callback {
 original := reflect.ValueOf(func() bool { return walk(20000) == 20000 })
 var callback Callback
 reflect.ValueOf(&callback).Elem().Set(original)
 holder := struct { Build Callback }{}
 reflect.ValueOf(&holder).Elem().Field(0).Set(original)
 list := make([]Callback, 1)
 reflect.ValueOf(list).Index(0).Set(original)
 values := make(map[int]Callback)
 reflect.ValueOf(values).SetMapIndex(reflect.ValueOf(0), original)
 ch := make(chan Callback, 1)
 reflect.ValueOf(ch).Send(original)
 fromChannel := <-ch
 typ := reflect.TypeOf(callback)
 converted := original.Convert(typ).Interface().(Callback)
 made := reflect.MakeFunc(typ, func([]reflect.Value) []reflect.Value {
  return []reflect.Value{reflect.ValueOf(walk(20000) == 20000)}
 }).Interface().(Callback)
 call := reflect.ValueOf(func(fn Callback) Callback { return fn }).Call([]reflect.Value{original})
 factory := reflect.MakeFunc(reflect.TypeOf(func() Callback { return nil }), func([]reflect.Value) []reflect.Value {
  return []reflect.Value{original}
 }).Interface().(func() Callback)
 appended := reflect.Append(reflect.ValueOf([]Callback{}), original).Interface().([]Callback)
 return []Callback{callback, holder.Build, list[0], values[0], fromChannel, converted, made, call[0].Interface().(Callback), factory(), appended[0]}[index]
}
func Recover() Callback {
 return func() (ok bool) {
  defer func() { ok = recover() == "deep" }()
  fail(12000)
  return false
 }
}
func main() { println(Exported()(), Recover()(), Inner()(), Reflected(0)()) }
`)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"./..."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(out, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(out, "lib", "host.dart"), fmt.Sprintf("import %q as entry;\nbool invoke() => entry.Inner()!();", (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(out, "bin", "main.dart"))}).String()))
	writeFile(t, filepath.Join(out, "host.dart"), `import 'bin/main.dart' as entry;
import 'package:goruntime/goruntime.dart';
void main(List<String> args) {
 entry.$init();
 if ($current != null || $blocked != null) throw StateError('initial state');
 for (var n = 0; n < 3; n++) {
  if (!entry.Exported()!() || !entry.Recover()!()) throw StateError('result');
  for (var i = 0; i < 10; i++) {
   if (!entry.Reflected($gi(i))!()) throw StateError('reflected result $i');
  }
  if ($current != null || $blocked != null) throw StateError('final state');
 }
 print('host passed');
}
`)
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, out, "dart", "host.dart")
			} else {
				got = runUpstreamTarget(t, target, out, "host.dart", out, []string{})
			}
			if got.Exit != 0 || got.Text != "host passed" {
				t.Fatalf("host result: %+v", got)
			}
		})
	}
	if got := runProgram(t, out, "dart", "bin/main.dart"); got != "true true true true" {
		t.Fatalf("goroutine result: %q", got)
	}
}

func TestRecursiveLanguage(t *testing.T) {
	inventory, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, fixture := range inventory.Cases {
		if fixture.Name == "peano" {
			cases = append(cases, fixture)
		}
	}
	if len(cases) != 1 {
		t.Fatal("missing upstream peano fixture")
	}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inventory, cases) })
	}
}
