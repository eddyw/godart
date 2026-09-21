package driver_test

import (
	"strings"
	"testing"
)

func TestSyncRejectsReflectedReplacement(t *testing.T) {
	err := buildSync(t, `package main
import ("reflect"; "runtime")
//gart:sync
type Callback func(int16, string) (int16, string)
func initial(int16, string) (int16, string) { return 0, "" }
func main() {
 f := initial
 reflect.ValueOf(&f).Elem().Set(reflect.MakeFunc(reflect.TypeOf(f), func(args []reflect.Value) []reflect.Value {
  runtime.Gosched()
  return args
 }))
 callback := Callback(f)
 callback(1, "value")
}
`)
	if err == nil || !strings.Contains(err.Error(), "Callback is a synchronous position") {
		t.Fatalf("the build must reject the reflected callback: %v", err)
	}
}

func TestSyncPreservesUnmodifiedFunctionValue(t *testing.T) {
	err := buildSync(t, `package main
import ("reflect"; "runtime")
//gart:sync
type Callback func(int16, string) (int16, string)
func initial(int16, string) (int16, string) { return 0, "" }
func main() {
 callback := Callback(initial)
 f := initial
 reflect.ValueOf(&f).Elem().Set(reflect.MakeFunc(reflect.TypeOf(f), func(args []reflect.Value) []reflect.Value {
  runtime.Gosched()
  return args
 }))
 callback(1, "value")
 f(2, "replacement")
}
`)
	if err != nil {
		t.Fatalf("the build rejected the unmodified function value: %v", err)
	}
}
