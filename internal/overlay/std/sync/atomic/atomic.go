//go:build dart

// Package atomic gives the atomic types that the surface needs.
//
// All Go code runs in 1 Dart isolate, and a goroutine gives up control only at
// a block operation. Thus a plain read and a plain write are atomic, and each
// operation below needs no lock.
//
// LoadPointer, StorePointer, SwapPointer, and CompareAndSwapPointer stay out.
// Each 1 names unsafe.Pointer in its signature, and stage 2 rejects an import
// of the package unsafe. Pointer[T] holds the same 4 operations with a typed
// pointer, and Go names it the safer form of each 1.
package atomic

import "syscall/dart"

// Value holds a value of any type.
type Value struct {
	v any
}

func (v *Value) Load() any { return v.v }

func (v *Value) Store(val any) {
	if val == nil {
		panic("sync/atomic: store of nil value into Value")
	}
	if v.v != nil && dart.DescOf(v.v) != dart.DescOf(val) {
		panic("sync/atomic: store of inconsistently typed value into Value")
	}
	v.v = val
}

func (v *Value) Swap(new any) any {
	if new == nil {
		panic("sync/atomic: swap of nil value into Value")
	}
	if v.v != nil && dart.DescOf(v.v) != dart.DescOf(new) {
		panic("sync/atomic: swap of inconsistently typed value into Value")
	}
	old := v.v
	v.v = new
	return old
}

func (v *Value) CompareAndSwap(old, new any) bool {
	if new == nil {
		panic("sync/atomic: compare and swap of nil value into Value")
	}
	if old != nil && dart.DescOf(old) != dart.DescOf(new) {
		panic("sync/atomic: compare and swap of inconsistently typed values")
	}
	if v.v != nil && dart.DescOf(v.v) != dart.DescOf(new) {
		panic("sync/atomic: compare and swap of inconsistently typed value into Value")
	}
	if v.v != old {
		return false
	}
	v.v = new
	return true
}

// Bool holds a bool.
type Bool struct {
	v bool
}

func (x *Bool) Load() bool { return x.v }

func (x *Bool) Store(val bool) { x.v = val }

func (x *Bool) Swap(new bool) bool {
	old := x.v
	x.v = new
	return old
}

func (x *Bool) CompareAndSwap(old, new bool) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
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

func (x *Int32) Swap(new int32) int32 {
	old := x.v
	x.v = new
	return old
}

func (x *Int32) And(mask int32) int32 {
	old := x.v
	x.v &= mask
	return old
}

func (x *Int32) Or(mask int32) int32 {
	old := x.v
	x.v |= mask
	return old
}

func (x *Int32) CompareAndSwap(old, new int32) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

// Int64 holds an int64.
type Int64 struct {
	v int64
}

func (x *Int64) Load() int64 { return x.v }

func (x *Int64) Store(val int64) { x.v = val }

func (x *Int64) Add(delta int64) int64 {
	x.v += delta
	return x.v
}

func (x *Int64) Swap(new int64) int64 {
	old := x.v
	x.v = new
	return old
}

func (x *Int64) And(mask int64) int64 {
	old := x.v
	x.v &= mask
	return old
}

func (x *Int64) Or(mask int64) int64 {
	old := x.v
	x.v |= mask
	return old
}

func (x *Int64) CompareAndSwap(old, new int64) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

// Uint32 holds a uint32.
type Uint32 struct {
	v uint32
}

func (x *Uint32) Load() uint32 { return x.v }

func (x *Uint32) Store(val uint32) { x.v = val }

func (x *Uint32) Add(delta uint32) uint32 {
	x.v += delta
	return x.v
}

func (x *Uint32) Swap(new uint32) uint32 {
	old := x.v
	x.v = new
	return old
}

func (x *Uint32) And(mask uint32) uint32 {
	old := x.v
	x.v &= mask
	return old
}

func (x *Uint32) Or(mask uint32) uint32 {
	old := x.v
	x.v |= mask
	return old
}

func (x *Uint32) CompareAndSwap(old, new uint32) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

// Uint64 holds a uint64.
type Uint64 struct {
	v uint64
}

func (x *Uint64) Load() uint64 { return x.v }

func (x *Uint64) Store(val uint64) { x.v = val }

func (x *Uint64) Add(delta uint64) uint64 {
	x.v += delta
	return x.v
}

func (x *Uint64) Swap(new uint64) uint64 {
	old := x.v
	x.v = new
	return old
}

func (x *Uint64) And(mask uint64) uint64 {
	old := x.v
	x.v &= mask
	return old
}

func (x *Uint64) Or(mask uint64) uint64 {
	old := x.v
	x.v |= mask
	return old
}

func (x *Uint64) CompareAndSwap(old, new uint64) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

// Uintptr holds a uintptr.
type Uintptr struct {
	v uintptr
}

func (x *Uintptr) Load() uintptr { return x.v }

func (x *Uintptr) Store(val uintptr) { x.v = val }

func (x *Uintptr) Add(delta uintptr) uintptr {
	x.v += delta
	return x.v
}

func (x *Uintptr) And(mask uintptr) uintptr {
	old := x.v
	x.v &= mask
	return old
}

func (x *Uintptr) Or(mask uintptr) uintptr {
	old := x.v
	x.v |= mask
	return old
}

func (x *Uintptr) Swap(new uintptr) uintptr {
	old := x.v
	x.v = new
	return old
}

func (x *Uintptr) CompareAndSwap(old, new uintptr) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

// Pointer holds a pointer to a value of the type parameter.
type Pointer[T any] struct {
	v *T
}

func (x *Pointer[T]) Load() *T { return x.v }

func (x *Pointer[T]) Store(val *T) { x.v = val }

func (x *Pointer[T]) Swap(new *T) *T {
	old := x.v
	x.v = new
	return old
}

func (x *Pointer[T]) CompareAndSwap(old, new *T) bool {
	if x.v != old {
		return false
	}
	x.v = new
	return true
}

// The function forms. Go holds 1 function for each width, and each one takes
// the address of the value.

func AddInt32(addr *int32, delta int32) int32 {
	*addr += delta
	return *addr
}

func AddInt64(addr *int64, delta int64) int64 {
	*addr += delta
	return *addr
}

func AddUint32(addr *uint32, delta uint32) uint32 {
	*addr += delta
	return *addr
}

func AddUint64(addr *uint64, delta uint64) uint64 {
	*addr += delta
	return *addr
}

func LoadInt32(addr *int32) int32 { return *addr }

func LoadInt64(addr *int64) int64 { return *addr }

func LoadUint32(addr *uint32) uint32 { return *addr }

func LoadUint64(addr *uint64) uint64 { return *addr }

func StoreInt32(addr *int32, val int32) { *addr = val }

func StoreInt64(addr *int64, val int64) { *addr = val }

func StoreUint32(addr *uint32, val uint32) { *addr = val }

func StoreUint64(addr *uint64, val uint64) { *addr = val }

func SwapInt32(addr *int32, new int32) int32 {
	old := *addr
	*addr = new
	return old
}

func SwapInt64(addr *int64, new int64) int64 {
	old := *addr
	*addr = new
	return old
}

func CompareAndSwapInt32(addr *int32, old, new int32) bool {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapInt64(addr *int64, old, new int64) bool {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func AddUintptr(addr *uintptr, delta uintptr) uintptr {
	*addr += delta
	return *addr
}

func LoadUintptr(addr *uintptr) uintptr { return *addr }

func StoreUintptr(addr *uintptr, val uintptr) { *addr = val }

func SwapUint32(addr *uint32, new uint32) uint32 {
	old := *addr
	*addr = new
	return old
}

func SwapUint64(addr *uint64, new uint64) uint64 {
	old := *addr
	*addr = new
	return old
}

func SwapUintptr(addr *uintptr, new uintptr) uintptr {
	old := *addr
	*addr = new
	return old
}

func CompareAndSwapUint32(addr *uint32, old, new uint32) bool {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapUint64(addr *uint64, old, new uint64) bool {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapUintptr(addr *uintptr, old, new uintptr) bool {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func AndInt32(addr *int32, mask int32) int32 {
	old := *addr
	*addr &= mask
	return old
}

func OrInt32(addr *int32, mask int32) int32 {
	old := *addr
	*addr |= mask
	return old
}

func AndInt64(addr *int64, mask int64) int64 {
	old := *addr
	*addr &= mask
	return old
}

func OrInt64(addr *int64, mask int64) int64 {
	old := *addr
	*addr |= mask
	return old
}

func AndUint32(addr *uint32, mask uint32) uint32 {
	old := *addr
	*addr &= mask
	return old
}

func OrUint32(addr *uint32, mask uint32) uint32 {
	old := *addr
	*addr |= mask
	return old
}

func AndUint64(addr *uint64, mask uint64) uint64 {
	old := *addr
	*addr &= mask
	return old
}

func OrUint64(addr *uint64, mask uint64) uint64 {
	old := *addr
	*addr |= mask
	return old
}

func AndUintptr(addr *uintptr, mask uintptr) uintptr {
	old := *addr
	*addr &= mask
	return old
}

func OrUintptr(addr *uintptr, mask uintptr) uintptr {
	old := *addr
	*addr |= mask
	return old
}
