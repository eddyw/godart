//go:build dart

// Package atomic gives the atomic types that the surface needs.
//
// All Go code runs in 1 Dart isolate, and a goroutine gives up control only at
// a block operation. Thus a plain read and a plain write are atomic, and each
// operation below needs no lock.
package atomic

// Value holds a value of any type.
type Value struct {
	v any
}

func (v *Value) Load() any { return v.v }

func (v *Value) Store(val any) { v.v = val }

func (v *Value) Swap(new any) any {
	old := v.v
	v.v = new
	return old
}

// Int32 holds an int32.
type Int32 struct {
	v int32
}

func (x *Int32) Load() int32 { return x.v }

func (x *Int32) Store(val int32) { x.v = val }

func (x *Int32) Add(delta int32) int32 {
	x.v += delta
	return x.v
}
