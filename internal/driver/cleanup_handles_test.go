package driver_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestCleanupHandles(t *testing.T) {
	t.Run("lifetime", testCleanupPointerLifetime)
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node is unavailable: %v", err)
	}
	out := t.TempDir()
	writeFile(t, filepath.Join(out, "pubspec.yaml"), fmt.Sprintf("name: cleanup_handles\nenvironment:\n  sdk: ^3.13.0\ndependencies:\n  goruntime:\n    path: %q\n", runtimeDir(t)))
	writeFile(t, filepath.Join(out, "main.dart"), `import 'package:goruntime/goruntime.dart' as rt;

void cleanup() {}

void main() {
  final target = rt.Iface(rt.$ptrDesc(rt.$typeInt), rt.Box(rt.$gi(1)));
  for (var i = 0; i < 10000000; i++) {
    rt.$stopCleanup(rt.$addCleanup(target, null, false, cleanup));
  }
  print('completed');
  print(target.value != null);
}
`)
	prepareDart(t, out)
	run(t, out, "dart", "compile", "js", "main.dart", "-o", "main.js")
	writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis; require('./main.js');\n")
	got := programOutcome(t, out, "node", "--max-old-space-size=64", "run.js")
	if got.Exit != 0 || got.Text != "completed\ntrue" {
		t.Fatalf("cleanup handle storage: %+v", got)
	}
}

func testCleanupPointerLifetime(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/cleanuplifetime\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"fmt"
"reflect"
	"runtime"
	"time"
)

type item struct {
	A, B int
	Link *int
	Data [256]byte
}
type view item

var global int

func notify(done chan bool) { done <- true }

func live[T any](name string, p *T, owner any) (runtime.Cleanup, chan bool) {
	done := make(chan bool, 1)
	cleanup := runtime.AddCleanup(p, notify, done)
	p = nil
	for range 5 {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case <-done:
		panic(name + ": premature cleanup")
	default:
	}
	runtime.KeepAlive(owner)
	return cleanup, done
}

func field() (runtime.Cleanup, chan bool) {
	owner := &item{A: 7}
	return live("field", &owner.A, owner)
}

func convertedField() (runtime.Cleanup, chan bool) {
	owner := &item{A: 7}
	v := (*view)(owner)
	return live("converted field", &v.A, owner)
}

func convertedPointer() (runtime.Cleanup, chan bool) {
	owner := &item{A: 7}
	return live("converted pointer", (*view)(owner), owner)
}

func element() (runtime.Cleanup, chan bool) {
	owner := make([]int, 32)
	return live("element", &owner[4], owner)
}

func arrayView() (runtime.Cleanup, chan bool) {
	owner := make([]int, 32)
	return live("array view", (*[8]int)(owner[2:]), owner)
}

func closure() (runtime.Cleanup, chan bool) {
 owner := new(item)
 done := make(chan bool, 1)
 cleanup := runtime.AddCleanup(owner, func(ch chan bool) { ch <- true }, done)
 return cleanup, done
}

func exercise(makeCleanup func() (runtime.Cleanup, chan bool)) {
	cleanup, done := makeCleanup()
	defer cleanup.Stop()
	deadline := time.Now().Add(4 * time.Second)
	for {
		runtime.GC()
		select {
		case <-done:
			fmt.Println("released cleanup owner")
			return
		default:
		}
		if time.Now().After(deadline) {
			panic("cleanup retained owner")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func main() {
	cleanup, _ := live("global", &global, &global)
	cleanup.Stop()
	exercise(field)
	exercise(convertedField)
	exercise(convertedPointer)
	exercise(element)
	exercise(arrayView)
	exercise(closure)
 exerciseInlineStorage()
 independentStorage()
}

type inner struct {
	Value int
	Data  [256]byte
}

func (p *inner) Self() *inner { return p }

type outer struct {
	Value int
	Inner inner
	Array [8]int
	Link  *int
}

type embedded struct {
	Value int
	inner
	Link *int
}

type generic[T any] struct {
	Value int
	Inner T
	Link  *int
}


func nested(done chan bool) any {
	p := new(outer)
	runtime.AddCleanup(&p.Value, notify, done)
	return &p.Inner
}

func nestedField(done chan bool) any {
	p := new(outer)
	runtime.AddCleanup(&p.Value, notify, done)
	return &p.Inner.Value
}

func arrayField(done chan bool) any {
	p := new(outer)
	runtime.AddCleanup(&p.Value, notify, done)
	return &p.Array[1]
}

func arrayElement(done chan bool) any {
	p := new([2]inner)
	runtime.AddCleanup(&p[0].Value, notify, done)
	return &p[1]
}

func sliceElement(done chan bool) any {
	p := make([]inner, 2)
	runtime.AddCleanup(&p[0].Value, notify, done)
	return &p[1]
}

func reflectedField(done chan bool) any {
	p := new(outer)
	runtime.AddCleanup(&p.Value, notify, done)
	return reflect.ValueOf(p).Elem().FieldByName("Inner").Addr().Interface()
}

func reflectedNestedField(done chan bool) any {
	p := new(outer)
	runtime.AddCleanup(&p.Value, notify, done)
	return reflect.ValueOf(p).Elem().FieldByName("Inner").FieldByName("Value").Addr().Interface()
}

func reflectedArray(done chan bool) any {
	p := new(outer)
	runtime.AddCleanup(&p.Value, notify, done)
	return reflect.ValueOf(p).Elem().FieldByName("Array").Index(1).Addr().Interface()
}

func reflectedSlice(done chan bool) any {
	p := make([]inner, 2)
	runtime.AddCleanup(&p[0].Value, notify, done)
	return reflect.ValueOf(p).Index(1).FieldByName("Value").Addr().Interface()
}

func receiver(done chan bool) any {
	p := new(outer)
	runtime.AddCleanup(&p.Value, notify, done)
	return p.Inner.Self()
}

func promotedReceiver(done chan bool) any {
	p := new(embedded)
	runtime.AddCleanup(&p.Value, notify, done)
	var v interface{ Self() *inner } = p
	return v.Self()
}

func receiverValue(done chan bool) any {
 p := new(outer)
 runtime.AddCleanup(&p.Value, notify, done)
 return p.Inner.Self
}

func promotedExpression(done chan bool) any {
 p := new(embedded)
 runtime.AddCleanup(&p.Value, notify, done)
 return (*embedded).Self(p)
}

func genericField[T any](p *generic[T]) *T { return &p.Inner }

func genericReceiver(done chan bool) any {
	p := new(generic[inner])
	runtime.AddCleanup(&p.Value, notify, done)
	return genericField(p)
}

func keep(name string, create func(chan bool) any) (chan bool, bool) {
	done := make(chan bool, 1)
	p := create(done)
	for range 5 {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	alive := true
	select {
	case <-done:
		fmt.Println(name + ": premature")
		alive = false
	default:
		fmt.Println(name + ": alive")
	}
	runtime.KeepAlive(p)
	return done, alive
}

func release(name string, done chan bool) {
	deadline := time.Now().Add(4 * time.Second)
	for {
		runtime.GC()
		select {
		case <-done:
			fmt.Println(name + ": released")
			return
		default:
		}
		if time.Now().After(deadline) {
			fmt.Println(name + ": retained")
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func exerciseInlineStorage() {
	for _, tc := range []struct {
		name   string
		create func(chan bool) any
	}{
		{"nested", nested},
		{"nested-field", nestedField},
		{"array-field", arrayField},
		{"array-element", arrayElement},
		{"slice-element", sliceElement},
		{"reflected-field", reflectedField},
		{"reflected-nested-field", reflectedNestedField},
		{"reflected-array", reflectedArray},
		{"reflected-slice", reflectedSlice},
		{"receiver", receiver},
        {"receiver-value", receiverValue},
        {"promoted-expression", promotedExpression},
		{"promoted-receiver", promotedReceiver},
		{"generic-receiver", genericReceiver},
	} {
		done, alive := keep(tc.name, tc.create)
		if alive {
			release(tc.name, done)
		}
	}
}

func referenceChild(done chan bool) *inner {
 owner := &struct { Value int; Child *inner }{Child: new(inner)}
 runtime.AddCleanup(&owner.Value, notify, done)
 return owner.Child
}

func zeroChild(done chan bool) {
 owner := new(struct { Value int; Zero [0]int; Link *int })
 child := &owner.Zero
 runtime.KeepAlive(child)
 runtime.AddCleanup(&owner.Value, notify, done)
}

func independentStorage() {
 done := make(chan bool, 1)
 child := referenceChild(done)
 release("reference", done)
 runtime.KeepAlive(child)
 done = make(chan bool, 1)
 zeroChild(done)
 release("zero-child", done)
}
`)
	want := runProgram(t, dir, "go", "run", ".")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var got programResult
			switch target {
			case "vm":
				got = programOutcome(t, out, "dart", "bin/main.dart")
			case "aot":
				exe := filepath.Join(out, "cleanup.exe")
				run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
				got = programOutcome(t, out, exe)
			case "js":
				run(t, out, "dart", "compile", "js", "bin/main.dart", "-o", "main.js")
				writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis; require('./main.js');\n")
				got = programOutcome(t, out, "node", "run.js")
			case "wasm":
				run(t, out, "dart", "compile", "wasm", "bin/main.dart", "-o", "main.wasm")
				writeFile(t, filepath.Join(out, "run.mjs"), wasmRunner)
				got = programOutcome(t, out, "node", "run.mjs")
			}
			if got.Exit != 0 || got.Text != want {
				t.Fatalf("%s: exit %d\nGo:\n%s\nGart:\n%s", target, got.Exit, want, got.Text)
			}
		})
	}
}
