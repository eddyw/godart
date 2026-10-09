package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestAtomicOpaquePointerFunctions(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/sync/atomic")
	fixture, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(fixture, "go.mod"), "module atomic_test\n\ngo 1.27\n")
	writeFile(t, filepath.Join(fixture, "000_package.go"), "package atomic_test\n")
	source := selectedUpstreamSource(t, filepath.Join(dir, "atomic_test.go"), []string{"global", "magic64", "testPointers", "TestLoadPointer", "TestStorePointer", "TestSwapPointer", "TestCompareAndSwapPointer"})
	writeFile(t, filepath.Join(fixture, "atomic_test.go"), source)

	stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^(TestLoadPointer|TestStorePointer|TestSwapPointer|TestCompareAndSwapPointer)$"
	args := upstreamTestArgs("-test.run=" + filter)
	result := upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
	compareUpstream(t, "atomic-opaque-pointer-functions", "sync/atomic", filter, "vm", dir, result.Text, result.Exit, true)
	for _, target := range []string{"aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			result := runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			compareUpstream(t, "atomic-opaque-pointer-functions", "sync/atomic", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}

func TestAtomicOpaquePointerContracts(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/atomicopaque\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import("fmt";"sync";"sync/atomic";"unsafe")
type pair struct { First, Last int }
func panics(f func()) (yes bool) { defer func(){ yes = recover()!=nil }(); f(); return }
func main() {
 var value unsafe.Pointer
 fmt.Println("zero",atomic.LoadPointer(&value)==nil)
 x:=new(pair);x.First=7;x.Last=9
 first,last:=unsafe.Pointer(&x.First),unsafe.Pointer(&x.Last)
 atomic.StorePointer(&value,unsafe.Pointer(x))
 fmt.Println("alias",atomic.LoadPointer(&value)==first,atomic.CompareAndSwapPointer(&value,first,last),atomic.LoadPointer(&value)==last)
 fmt.Println("mismatch",atomic.CompareAndSwapPointer(&value,first,nil),atomic.LoadPointer(&value)==last)
 fmt.Println("swap nil",atomic.SwapPointer(&value,nil)==last,atomic.LoadPointer(&value)==nil)
 fmt.Println("nil compare",atomic.CompareAndSwapPointer(&value,nil,first),atomic.LoadPointer(&value)==first)
 fmt.Println("nil addresses",panics(func(){atomic.LoadPointer(nil)}),panics(func(){atomic.StorePointer(nil,first)}),panics(func(){atomic.SwapPointer(nil,first)}),panics(func(){atomic.CompareAndSwapPointer(nil,nil,first)}))
 var count atomic.Int32
 var group sync.WaitGroup
 for i:=0;i<4;i++ { group.Add(1);go func(){defer group.Done();for j:=0;j<100;j++{ for { old:=atomic.LoadPointer(&value);next:=first;if old==first{next=last};if atomic.CompareAndSwapPointer(&value,old,next){count.Add(1);break} } }}() }
 group.Wait()
 fmt.Println("goroutines",count.Load(),atomic.LoadPointer(&value)==first,x.First,x.Last)
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
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("Go: %+v\nGart: %+v", want, result)
			}
		})
	}
}
