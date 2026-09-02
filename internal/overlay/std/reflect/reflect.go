//go:build dart

// Package reflect gives the part of the standard reflect package that the
// surface holds.
//
// The compiler cannot read the real source. That source reads the memory layout
// of a value, and the Dart VM gives no such access. The type descriptor of the
// runtime library holds the name, the kind, and each field of a type, thus the
// package gives a deep comparison, the description of a type, and a read and a
// write of a value.
//
// `docs/parity.md` lists each gap.
package reflect

import (
	"iter"
	"syscall/dart"
)

// DeepEqual reports whether 2 values are deeply equal.
//
// 2 values of 2 types are never equal. A slice, a map, a struct, an array, a
// pointer, and an interface each compare their parts, and each other kind
// compares with the equality of Go.
func DeepEqual(x, y any) bool {
	return dart.DeepEqual(x, y)
}

// A Kind names the class of a type, for example `int` or `struct`.
//
// Go gives an unsigned number, and the overlay gives the text of the kind. The
// type descriptor of the runtime holds that text, and the 2 forms hold the same
// set of values.
type Kind string

// The kinds of Go.
const (
	Invalid       Kind = "invalid"
	Bool          Kind = "bool"
	Int           Kind = "int"
	Int8          Kind = "int8"
	Int16         Kind = "int16"
	Int32         Kind = "int32"
	Int64         Kind = "int64"
	Uint          Kind = "uint"
	Uint8         Kind = "uint8"
	Uint16        Kind = "uint16"
	Uint32        Kind = "uint32"
	Uint64        Kind = "uint64"
	Uintptr       Kind = "uintptr"
	Float32       Kind = "float32"
	Float64       Kind = "float64"
	Complex64     Kind = "complex64"
	Complex128    Kind = "complex128"
	Array         Kind = "array"
	Chan          Kind = "chan"
	Func          Kind = "func"
	Interface     Kind = "interface"
	Map           Kind = "map"
	Pointer       Kind = "ptr"
	Slice         Kind = "slice"
	String        Kind = "string"
	Struct        Kind = "struct"
	UnsafePointer Kind = "unsafe.Pointer"
)

// Ptr is the old name of Pointer.
const Ptr = Pointer

// String gives the name of the kind.
func (k Kind) String() string { return string(k) }

// A Type describes a Go type. It compares with nil, exactly as it does in Go.
//
// The overlay holds 1 implementation, and that implementation is unexported.
// A Dart library holds each private name of its own, thus a method of an
// unexported type reaches no other package as a plain call. An interface
// dispatches through the method table, and each package then reads it.
type Type interface {
	String() string
	Name() string
	Kind() Kind
	Comparable() bool
	Elem() Type
	Key() Type
	Len() int
	NumField() int
	Field(i int) StructField
	FieldByName(name string) (StructField, bool)
	FieldByIndex(index []int) StructField
	Bits() int
	NumIn() int
	In(i int) Type
	NumOut() int
	Out(i int) Type
	IsVariadic() bool
	NumMethod() int
	Method(i int) Method
	Size() uintptr
	Align() int
	FieldAlign() int
	AssignableTo(u Type) bool
	ConvertibleTo(u Type) bool
	Implements(u Type) bool
	ChanDir() ChanDir
	CanSeq() bool
	CanSeq2() bool
	handle() int
	layout() (uintptr, int)
	offsetOf(index int) uintptr
}

// An rtype holds the handle of 1 type descriptor of the runtime.
type rtype struct {
	desc int
}

// types interns 1 rtype for each handle, thus 2 reads of the same type give 2
// equal values, as they do in Go.
var types = map[int]*rtype{}

// typeOf gives the Type of a handle. The handle 0 names no type, and it gives
// nil, exactly as a nil interface does in Go.
func typeOf(handle int) Type {
	if handle == 0 {
		return nil
	}
	if t := types[handle]; t != nil {
		return t
	}
	t := &rtype{desc: handle}
	types[handle] = t
	return t
}

// handle gives the handle of a type. A nil Type gives 0.
func (t *rtype) handle() int {
	if t == nil {
		return 0
	}
	return t.desc
}

// A StructTag is the tag of a struct field.
type StructTag string

// Get gives the value of the key of a tag, and it gives an empty string for a
// key that the tag does not hold.
func (tag StructTag) Get(key string) string {
	v, _ := tag.Lookup(key)
	return v
}

// Lookup gives the value of the key of a tag. The flag is false when the tag
// holds no such key.
func (tag StructTag) Lookup(key string) (string, bool) {
	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		value := string(tag[1:i])
		tag = tag[i+1:]
		if key == name {
			return value, true
		}
	}
	return "", false
}

// A StructField describes 1 field of a struct type.
type StructField struct {
	Name string
	// PkgPath holds the import path of an unexported field, and it is empty
	// for an exported field.
	PkgPath string
	Tag     StructTag
	Type    Type
	// Index holds the path of the field inside the struct. The overlay reads
	// no embedded field of a parent, thus the path holds 1 element.
	Index []int
	// Offset holds the count of bytes from the start of the struct. Read
	// Type.Size.
	Offset uintptr
	// Anonymous marks an embedded field.
	Anonymous bool
}

// TypeOf gives the type of the value inside an interface.
func TypeOf(v any) Type {
	return typeOf(dart.DescOf(v))
}

// String gives the text of the type. It holds the name of the package, and not
// the import path.
func (t *rtype) String() string {
	text, _, _ := dart.DescInfo(t.handle())
	return text
}

// Name gives the name of a named type, and it gives an empty string for each
// other type.
func (t *rtype) Name() string {
	text := t.String()
	if composite(text) {
		return ""
	}
	// The name of an instance of a generic type holds the type arguments, and
	// each one can name its own package. The package of the type itself stands
	// before the first bracket.
	head := text
	for i := 0; i < len(text); i++ {
		if text[i] == '[' {
			head = text[:i]
			break
		}
	}
	for i := len(head) - 1; i >= 0; i-- {
		if head[i] == '.' {
			return text[i+1:]
		}
	}
	return text
}

// composite reports whether the text names a type that Go does not define with
// a name.
func composite(text string) bool {
	for _, prefix := range []string{
		"[", "*", "map[", "chan ", "<-chan ", "func(",
		"struct{", "struct {", "interface{", "interface {",
	} {
		if len(text) >= len(prefix) && text[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// Kind gives the class of the type.
func (t *rtype) Kind() Kind {
	_, kind, _ := dart.DescInfo(t.handle())
	return Kind(kind)
}

// Comparable reports whether Go compares 2 values of the type with ==.
func (t *rtype) Comparable() bool {
	_, _, comparable := dart.DescInfo(t.handle())
	return comparable
}

// Elem gives the element type of a pointer, a slice, an array, a channel, or a
// map. Each other type panics, exactly as in Go.
func (t *rtype) Elem() Type {
	elem := dart.DescElem(t.handle())
	if elem == 0 {
		panic("reflect: Elem of " + t.String())
	}
	return typeOf(elem)
}

// Len gives the count of the elements of an array type. Each other type panics,
// exactly as in Go.
func (t *rtype) Len() int {
	if t.Kind() != Array {
		panic("reflect: Len of " + t.String())
	}
	return dart.DescLength(t.handle())
}

// NumField gives the count of the fields of a struct type.
func (t *rtype) NumField() int {
	if t.Kind() != Struct {
		panic("reflect: NumField of " + t.String())
	}
	return dart.FieldCount(t.handle())
}

// Field gives 1 field of a struct type.
func (t *rtype) Field(i int) StructField {
	if t.Kind() != Struct {
		panic("reflect: Field of " + t.String())
	}
	name, tag, exported, desc := dart.FieldInfo(t.handle(), i)
	pkg := ""
	if !exported {
		pkg = dart.DescPkgPath(t.handle())
	}
	return StructField{
		Name:      name,
		PkgPath:   pkg,
		Tag:       StructTag(tag),
		Type:      typeOf(desc),
		Index:     []int{i},
		Offset:    t.offsetOf(i),
		Anonymous: dart.FieldEmbedded(t.handle(), i),
	}
}

// offsetOf gives the count of bytes from the start of the struct to the field.
// Read Type.Size.
func (t *rtype) offsetOf(index int) uintptr {
	offset := uintptr(0)
	for i := 0; i <= index; i++ {
		_, _, _, desc := dart.FieldInfo(t.handle(), i)
		field := typeOf(desc)
		offset = roundUp(offset, uintptr(field.Align()))
		if i == index {
			return offset
		}
		offset += field.Size()
	}
	return offset
}

// Key gives the key type of a map type. Each other type panics, exactly as in
// Go.
func (t *rtype) Key() Type {
	key := dart.DescKey(t.handle())
	if key == 0 {
		panic("reflect: Key of " + t.String())
	}
	return typeOf(key)
}

// NumIn gives the count of the parameters of a function type.
func (t *rtype) NumIn() int {
	if t.Kind() != Func {
		panic("reflect: NumIn of " + t.String())
	}
	return dart.DescNumIn(t.handle())
}

// In gives the type of 1 parameter of a function type.
func (t *rtype) In(i int) Type {
	if t.Kind() != Func {
		panic("reflect: In of " + t.String())
	}
	return typeOf(dart.DescIn(t.handle(), i))
}

// NumOut gives the count of the results of a function type.
func (t *rtype) NumOut() int {
	if t.Kind() != Func {
		panic("reflect: NumOut of " + t.String())
	}
	return dart.DescNumOut(t.handle())
}

// Out gives the type of 1 result of a function type.
func (t *rtype) Out(i int) Type {
	if t.Kind() != Func {
		panic("reflect: Out of " + t.String())
	}
	return typeOf(dart.DescOut(t.handle(), i))
}

// IsVariadic reports whether the last parameter of a function type holds a
// list. The description of the output holds no such flag, thus the result is
// always false.
func (t *rtype) IsVariadic() bool { return dart.DescVariadic(t.handle()) }

// A Method describes 1 exported method of a type.
type Method struct {
	Name  string
	Index int
}

// NumMethod gives the count of the exported methods of the type.
func (t *rtype) NumMethod() int { return dart.DescNumMethod(t.handle()) }

// Method gives 1 exported method of the type, in the order of the name.
func (t *rtype) Method(i int) Method {
	return Method{Name: dart.DescMethodName(t.handle(), i), Index: i}
}

// Size gives the count of bytes that a value of the type holds in Go.
//
// The Dart output holds no memory layout. The result follows the rules of a
// 64-bit machine, exactly as the compiler of Go computes them.
func (t *rtype) Size() uintptr {
	switch t.Kind() {
	case Bool, Int8, Uint8:
		return 1
	case Int16, Uint16:
		return 2
	case Int32, Uint32, Float32:
		return 4
	case Int, Int64, Uint, Uint64, Uintptr, Float64, Complex64,
		Pointer, UnsafePointer, Map, Chan, Func:
		return 8
	case Complex128, Interface, String:
		return 16
	case Slice:
		return 24
	case Array:
		return uintptr(t.Len()) * t.Elem().Size()
	case Struct:
		size, _ := t.layout()
		return size
	}
	panic("reflect: Size of " + t.String())
}

// Align gives the count of bytes of the alignment of a value of the type.
func (t *rtype) Align() int {
	switch t.Kind() {
	case Bool, Int8, Uint8:
		return 1
	case Int16, Uint16:
		return 2
	case Int32, Uint32, Float32, Complex64:
		return 4
	case Int, Int64, Uint, Uint64, Uintptr, Float64, Complex128,
		Pointer, UnsafePointer, Map, Chan, Func, Interface, String, Slice:
		return 8
	case Array:
		return t.Elem().Align()
	case Struct:
		_, align := t.layout()
		return align
	}
	panic("reflect: Align of " + t.String())
}

// FieldAlign gives the alignment of the type inside a struct.
func (t *rtype) FieldAlign() int { return t.Align() }

// layout gives the size and the alignment of a struct type.
//
// Go puts each field at an offset that its alignment allows, and it rounds the
// size up to the alignment of the struct. A struct whose last field holds 0
// bytes takes 1 more byte, thus a pointer to the end stays inside the value.
func (t *rtype) layout() (uintptr, int) {
	count := dart.FieldCount(t.handle())
	offset := uintptr(0)
	align := 1
	last := uintptr(0)
	for i := 0; i < count; i++ {
		_, _, _, desc := dart.FieldInfo(t.handle(), i)
		field := typeOf(desc)
		a := field.Align()
		if a > align {
			align = a
		}
		offset = roundUp(offset, uintptr(a))
		last = field.Size()
		offset += last
	}
	if count > 0 && last == 0 && offset > 0 {
		offset++
	}
	return roundUp(offset, uintptr(align)), align
}

// roundUp gives the first multiple of the alignment that is not below n.
func roundUp(n, align uintptr) uintptr {
	return (n + align - 1) / align * align
}

// The 4 forms of the location that a Value names. A Value of the form locNone
// holds no location, thus Set panics.
const (
	locNone = iota
	locPointer
	locField
	locIndex
)

// A Value holds a Go value. A Value that Elem, Field, or Index read also holds
// its location, thus Set writes through it.
type Value struct {
	value any
	owner any
	index int
	loc   int
	// typ holds the static type when the value itself is nil, for example
	// the element of a pointer to an interface.
	typ Type
}

// ValueOf gives the value inside an interface.
func ValueOf(v any) Value { return Value{value: v} }

// Indirect gives the value behind a pointer. A value that is no pointer gives
// itself.
func Indirect(v Value) Value {
	if v.Kind() != Pointer {
		return v
	}
	return v.Elem()
}

// Zero gives the zero value of a type.
func Zero(t Type) Value { return Value{value: dart.DescZero(t.handle())} }

// New gives a Value that holds a pointer to a fresh zero value of the type.
func New(t Type) Value { return Value{value: dart.NewValue(t.handle())} }

// MakeSlice gives a slice of the type, with the length and the capacity. The
// overlay gives each slice a capacity that is equal to its length.
func MakeSlice(t Type, length, capacity int) Value {
	if t.Kind() != Slice {
		panic("reflect: MakeSlice of " + t.String())
	}
	if length < 0 || capacity < length {
		panic("reflect: MakeSlice with a wrong length")
	}
	return Value{value: dart.MakeValue(t.handle(), length, capacity)}
}

// MakeMap gives an empty map of the type.
func MakeMap(t Type) Value {
	if t.Kind() != Map {
		panic("reflect: MakeMap of " + t.String())
	}
	return Value{value: dart.MakeMap(t.handle())}
}

// TypeAssert reads the value inside v as the type T. The flag is false when the
// value holds another type.
func TypeAssert[T any](v Value) (T, bool) {
	out, ok := v.read().(T)
	return out, ok
}

// read gives the value at the location that v names. A Set writes through the
// location, thus a read after it gives the new value.
func (v Value) read() any {
	switch v.loc {
	case locPointer:
		return dart.PointerElem(v.owner)
	case locField:
		return dart.FieldRead(v.owner, v.index)
	case locIndex:
		return dart.ValueIndex(v.owner, v.index)
	}
	return v.value
}

// IsValid reports whether v holds a value.
func (v Value) IsValid() bool { return v.read() != nil }

// Interface gives the value of v inside an interface.
func (v Value) Interface() any { return v.read() }

// Type gives the type of the value.
func (v Value) Type() Type {
	if v.read() == nil && v.typ != nil {
		return v.typ
	}
	return TypeOf(v.read())
}

// Kind gives the class of the type of the value.
func (v Value) Kind() Kind {
	if v.read() == nil {
		if v.typ != nil {
			return v.typ.Kind()
		}
		return Invalid
	}
	return v.Type().Kind()
}

// Elem gives the value behind a pointer. The result holds the pointer, thus Set
// writes through it.
func (v Value) Elem() Value {
	p := v.read()
	out := Value{value: dart.PointerElem(p), owner: p, loc: locPointer}
	if v.Kind() == Pointer {
		out.typ = v.Type().Elem()
	}
	return out
}

// Len gives the length of a slice, an array, a string, or a map.
func (v Value) Len() int { return dart.ValueLen(v.read()) }

// Index gives 1 element of a slice, an array, or a string. The result of a
// slice and of an array holds its location, thus Set writes through it.
func (v Value) Index(i int) Value {
	owner := v.read()
	out := Value{value: dart.ValueIndex(owner, i)}
	if v.Kind() != String {
		out.owner, out.index, out.loc = owner, i, locIndex
		out.typ = v.Type().Elem()
	}
	return out
}

// NumField gives the count of the fields of a struct value.
func (v Value) NumField() int { return v.Type().NumField() }

// Field gives 1 field of a struct value. The result holds its location, thus
// Set writes through it.
func (v Value) Field(i int) Value {
	owner := v.read()
	return Value{value: dart.FieldRead(owner, i), owner: owner,
		index: i, loc: locField, typ: v.Type().Field(i).Type}
}

// CanSet reports whether Set writes through v.
func (v Value) CanSet() bool { return v.loc != locNone }

// Set writes x into the location that v names. A Value that holds no location
// panics.
func (v Value) Set(x Value) {
	switch v.loc {
	case locPointer:
		dart.PointerSet(v.owner, x.read())
	case locField:
		dart.FieldWrite(v.owner, v.index, x.read())
	case locIndex:
		dart.ValueSetIndex(v.owner, v.index, x.read())
	default:
		panic("reflect: Set of a value that holds no address")
	}
}

// Int gives the value of a signed integer kind.
func (v Value) Int() int64 { return dart.AsInt(v.read()) }

// Uint gives the value of an unsigned integer kind.
func (v Value) Uint() uint64 { return uint64(dart.AsInt(v.read())) }

// Float gives the value of a float kind.
func (v Value) Float() float64 { return dart.AsFloat(v.read()) }

// Bool gives the value of the kind bool.
func (v Value) Bool() bool { return dart.AsBool(v.read()) }

// String gives the text of a value of the kind string. Each other kind gives a
// description of the value, exactly as in Go.
func (v Value) String() string {
	if v.Kind() == String {
		return dart.AsString(v.read())
	}
	if v.read() == nil {
		return "<invalid Value>"
	}
	return "<" + v.Type().String() + " Value>"
}

// NumMethod gives the count of the exported methods of the type of v.
func (v Value) NumMethod() int { return v.Type().NumMethod() }

// Method gives a func value that binds 1 exported method of the type to v.
func (v Value) Method(i int) Value {
	return Value{value: dart.MethodValue(v.read(), i)}
}

// Call runs the function that v holds and gives each result.
//
// A function that gives up control is outside this call. The Dart form of such
// a function gives a sentinel of the scheduler, and the call reads no
// scheduler.
func (v Value) Call(in []Value) []Value {
	args := make([]any, len(in))
	for i, a := range in {
		args[i] = a.read()
	}
	raw := dart.CallValue(v.read(), args)
	// The static type of each result travels with the value. A nil interface
	// result holds no descriptor, and Kind then reads the static type.
	t := v.Type()
	out := make([]Value, len(raw))
	for i, r := range raw {
		out[i] = Value{value: r, typ: t.Out(i)}
	}
	return out
}

// SetMapIndex writes 1 entry of a map. A key with no value drops the entry.
func (v Value) SetMapIndex(key, value Value) {
	dart.MapSet(v.read(), key.read(), value.read(), !value.IsValid())
}

// IsNil reports whether the value of a kind that can hold nil holds nil. Each
// other kind panics.
func (v Value) IsNil() bool {
	switch v.Kind() {
	case Chan, Func, Interface, Map, Pointer, Slice:
		if v.read() == nil {
			return true
		}
		return dart.ValueIsNil(v.read())
	}
	panic("reflect: IsNil of " + v.Kind().String())
}

// MapKeys gives the keys of a map value. The order follows the insertion.
func (v Value) MapKeys() []Value {
	raw := dart.MapKeys(v.read())
	out := make([]Value, len(raw))
	for i, k := range raw {
		out[i] = Value{value: k}
	}
	return out
}

// MapIndex gives the value of 1 key of a map value. A key that the map does
// not hold gives the zero Value.
func (v Value) MapIndex(key Value) Value {
	return Value{value: dart.MapIndex(v.read(), key.read()), typ: v.Type().Elem()}
}

// SetZero writes the zero value of the type into the location that v names.
func (v Value) SetZero() { v.Set(Zero(v.Type())) }

// SetInt writes a whole number into the location that v names.
func (v Value) SetInt(x int64) { v.Set(Value{value: dart.WithInt(v.descOf(), x)}) }

// SetUint writes a whole number into the location that v names.
func (v Value) SetUint(x uint64) { v.Set(Value{value: dart.WithInt(v.descOf(), int64(x))}) }

// SetFloat writes a float into the location that v names.
func (v Value) SetFloat(x float64) { v.Set(Value{value: dart.WithFloat(v.descOf(), x)}) }

// SetBool writes a flag into the location that v names.
func (v Value) SetBool(x bool) { v.Set(Value{value: dart.WithBool(v.descOf(), x)}) }

// SetString writes text into the location that v names.
func (v Value) SetString(x string) { v.Set(Value{value: dart.WithString(v.descOf(), x)}) }

func (v Value) descOf() int { return dart.DescOf(v.read()) }

// The type constructors.
//
// A constructor gives a Type that no value of the program carries. The name of
// the type goes through the table of the runtime first, thus a type that the
// program already holds keeps its own descriptor and MakeSlice, MakeMap, New,
// and Zero of it each work. A type that the program never holds gives a Type
// for the inspection alone, and each of those 4 calls then panics with a clear
// text. Read docs/parity.md.

// SliceOf gives the slice type of an element type.
func SliceOf(t Type) Type { return typeOf(dart.SliceOf(t.handle())) }

// ArrayOf gives the array type of a length and an element type.
func ArrayOf(length int, t Type) Type {
	if length < 0 {
		panic("reflect: ArrayOf with a negative length")
	}
	return typeOf(dart.ArrayOf(length, t.handle()))
}

// MapOf gives the map type of a key type and an element type.
func MapOf(key, elem Type) Type {
	if !key.Comparable() {
		panic("reflect: invalid key type " + key.String())
	}
	return typeOf(dart.MapOf(key.handle(), elem.handle()))
}

// PointerTo gives the pointer type of an element type.
func PointerTo(t Type) Type { return typeOf(dart.PointerTo(t.handle())) }

// PtrTo gives the pointer type of an element type. Go marks it as older than
// PointerTo.
func PtrTo(t Type) Type { return PointerTo(t) }

// TypeFor gives the type of T.
func TypeFor[T any]() Type {
	var zero T
	if t := TypeOf(zero); t.handle() != 0 {
		return t
	}
	// The zero value of an interface type holds no descriptor, thus the type
	// comes from a pointer to it.
	return TypeOf(&zero).Elem()
}

// Cap gives the capacity of a slice, of an array, or of a channel.
func (v Value) Cap() int { return dart.ValueCap(v.read()) }

// Append gives a slice that holds each element of s and each value of x.
func Append(s Value, x ...Value) Value {
	if s.Kind() != Slice {
		panic("reflect: Append of " + s.Type().String())
	}
	items := make([]any, len(x))
	for i, v := range x {
		items[i] = v.read()
	}
	return Value{value: dart.AppendValue(s.read(), items)}
}

// AppendSlice gives a slice that holds each element of s and each element of t.
func AppendSlice(s, t Value) Value {
	if s.Kind() != Slice {
		panic("reflect: AppendSlice of " + s.Type().String())
	}
	return Value{value: dart.AppendSliceValue(s.read(), t.read())}
}

// Copy writes each element of src into dst, and it gives the count that it
// wrote. src may hold a string when dst holds a byte slice.
func Copy(dst, src Value) int {
	switch dst.Kind() {
	case Slice, Array:
	default:
		panic("reflect: Copy of " + dst.Type().String())
	}
	return dart.CopyValue(dst.read(), src.read())
}

// A ChanDir names the direction of a channel type.
type ChanDir int

// The 3 directions of a channel type.
const (
	RecvDir ChanDir = 1 << iota
	SendDir
	BothDir = RecvDir | SendDir
)

func (d ChanDir) String() string {
	switch d {
	case SendDir:
		return "chan<-"
	case RecvDir:
		return "<-chan"
	case BothDir:
		return "chan"
	}
	return "ChanDir" + itoa(int(d))
}

// itoa writes a small integer. The overlay reads no strconv, because strconv
// reads no reflect and the surface keeps the 2 apart.
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	if neg {
		return "-" + out
	}
	return out
}

// ChanDir gives the direction of a channel type.
//
// The descriptor of the runtime holds no direction, thus each channel type
// reads as a channel of both directions. Read docs/parity.md.
func (t *rtype) ChanDir() ChanDir {
	if t.Kind() != Chan {
		panic("reflect: ChanDir of " + t.String())
	}
	return ChanDir(dart.DescChanDir(t.handle()))
}

// AssignableTo reports whether a value of the type fits a location of u.
func (t *rtype) AssignableTo(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.AssignableTo")
	}
	return dart.DescAssignable(t.handle(), u.handle())
}

// ConvertibleTo reports whether a conversion of the type to u holds.
func (t *rtype) ConvertibleTo(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.ConvertibleTo")
	}
	return dart.DescConvertible(t.handle(), u.handle())
}

// Implements reports whether the type holds each method of the interface u.
func (t *rtype) Implements(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.Implements")
	}
	if u.Kind() != Interface {
		panic("reflect: non-interface type passed to Type.Implements")
	}
	return dart.DescImplements(t.handle(), u.handle())
}

// FieldByName gives the field of the name, and whether the struct holds it.
func (t *rtype) FieldByName(name string) (StructField, bool) {
	if t.Kind() != Struct {
		panic("reflect: FieldByName of " + t.String())
	}
	i := dart.DescFieldByName(t.handle(), name)
	if i < 0 {
		return StructField{}, false
	}
	return t.Field(i), true
}

// CanSeq reports whether a range loop over a value of the type gives 1 value.
func (t *rtype) CanSeq() bool {
	switch t.Kind() {
	case Int, Int8, Int16, Int32, Int64,
		Uint, Uint8, Uint16, Uint32, Uint64, Uintptr,
		Array, Slice, Chan, String, Map:
		return true
	case Func:
		return t.NumIn() == 1 && t.NumOut() == 0
	}
	return false
}

// CanSeq2 reports whether a range loop over a value of the type gives 2 values.
func (t *rtype) CanSeq2() bool {
	switch t.Kind() {
	case Array, Slice, String, Map:
		return true
	case Func:
		return t.NumIn() == 1 && t.NumOut() == 0
	}
	return false
}

// The value operations that a template engine reads.

// CanAddr reports whether Addr gives a pointer to the value.
func (v Value) CanAddr() bool { return v.loc != locNone }

// Addr gives a pointer to the value. The value must hold a location.
func (v Value) Addr() Value {
	if v.loc == locNone {
		panic("reflect: Addr of a value that holds no address")
	}
	return Value{value: dart.ValueAddr(v.read(), v.owner, v.index, v.loc)}
}

// MethodByName gives the method of the name, bound to the value. A name that
// no method holds gives the zero Value.
func (v Value) MethodByName(name string) Value {
	out := dart.MethodByName(v.read(), name)
	if out == nil {
		return Value{}
	}
	return Value{value: out}
}

// Equal reports whether the 2 values hold the same type and the same value.
func (v Value) Equal(u Value) bool { return dart.ValueEqual(v.read(), u.read()) }

// Convert gives the value converted to the type t.
func (v Value) Convert(t Type) Value {
	if t == nil {
		panic("reflect: nil type passed to Value.Convert")
	}
	return Value{value: dart.ValueConvert(v.read(), t.handle())}
}

// CanConvert reports whether a conversion of the value to t holds.
func (v Value) CanConvert(t Type) bool {
	if t == nil {
		return false
	}
	return v.Type().ConvertibleTo(t)
}

// Slice gives the slice v[i:j].
func (v Value) Slice(i, j int) Value {
	max := v.Cap()
	if v.Kind() == String {
		max = v.Len()
	}
	return Value{value: dart.ValueSlice(v.read(), i, j, max)}
}

// Slice3 gives the slice v[i:j:k].
func (v Value) Slice3(i, j, k int) Value {
	return Value{value: dart.ValueSlice(v.read(), i, j, k)}
}

// Complex gives the complex value.
func (v Value) Complex() complex128 { return dart.AsComplex(v.read()) }

// SetComplex writes a complex value through the location.
func (v Value) SetComplex(x complex128) {
	v.Set(Value{value: dart.WithComplex(v.descOf(), x)})
}

// Pointer gives a number that names the object of the value.
//
// Dart gives no machine address, thus the number is the identity of the object
// and not an address. 2 reads of the same object give the same number, and the
// number names no memory. Read docs/parity.md.
func (v Value) Pointer() uintptr { return dart.ValuePointer(v.read()) }

// UnsafePointer gives the same number as Pointer.
func (v Value) UnsafePointer() uintptr { return v.Pointer() }

// IsZero reports whether the value is the zero value of its type.
func (v Value) IsZero() bool {
	t := v.Type()
	if t == nil {
		return true
	}
	return dart.ValueEqual(v.read(), dart.DescZero(t.handle()))
}

// CanInterface reports whether Interface gives the value. Each value of the
// overlay reads, thus the answer is always true.
func (v Value) CanInterface() bool { return v.IsValid() }

// FieldByName gives the field of the name. A name that no field holds gives
// the zero Value.
func (v Value) FieldByName(name string) Value {
	f, ok := v.Type().FieldByName(name)
	if !ok {
		return Value{}
	}
	return v.Field(f.Index[0])
}

// FieldByIndex gives the field that each index of the path names.
func (v Value) FieldByIndex(index []int) Value {
	out := v
	for i, x := range index {
		if i > 0 && out.Kind() == Pointer {
			out = out.Elem()
		}
		out = out.Field(x)
	}
	return out
}

// FieldByIndexErr does the same, and it gives an error for a nil pointer of
// the path in place of a panic.
func (v Value) FieldByIndexErr(index []int) (Value, error) {
	out := v
	for i, x := range index {
		if i > 0 && out.Kind() == Pointer {
			if out.IsNil() {
				return Value{}, errNilField
			}
			out = out.Elem()
		}
		out = out.Field(x)
	}
	return out, nil
}

// errNilField names a nil pointer inside the path of FieldByIndexErr.
var errNilField = errorText("reflect: indirection through a nil pointer to an embedded struct")

// errorText is an error that holds 1 text. The overlay reads no errors
// package, because that package reads no reflect and the surface keeps the 2
// apart.
type errorText string

func (e errorText) Error() string { return string(e) }

// A MapIter walks the pairs of a map.
type MapIter struct {
	keys []Value
	m    Value
	at   int
}

// MapRange gives an iterator of the pairs of a map.
func (v Value) MapRange() *MapIter {
	return &MapIter{keys: v.MapKeys(), m: v, at: -1}
}

// Next steps to the next pair, and it reports whether a pair stands there.
func (it *MapIter) Next() bool {
	it.at++
	return it.at < len(it.keys)
}

// Key gives the key of the pair.
func (it *MapIter) Key() Value { return it.keys[it.at] }

// Value gives the value of the pair.
func (it *MapIter) Value() Value { return it.m.MapIndex(it.keys[it.at]) }

// Reset points the iterator at another map.
func (it *MapIter) Reset(m Value) {
	it.keys = m.MapKeys()
	it.m = m
	it.at = -1
}

// IsExported reports whether the field is exported.
func (f StructField) IsExported() bool { return f.PkgPath == "" }

// Recv reads 1 value of a channel. The flag is false when the channel closed.
func (v Value) Recv() (Value, bool) {
	out, ok := dart.ValueRecv(v.read())
	return Value{value: out}, ok
}

// Seq gives a sequence of 1 value for each element.
func (v Value) Seq() iter.Seq[Value] {
	return func(yield func(Value) bool) {
		switch v.Kind() {
		case Int, Int8, Int16, Int32, Int64,
			Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
			for i := int64(0); i < v.Int(); i++ {
				if !yield(Value{value: dart.WithInt(v.descOf(), i)}) {
					return
				}
			}
		case Slice, Array, String:
			for i := range v.Len() {
				if !yield(v.Index(i)) {
					return
				}
			}
		case Map:
			for _, k := range v.MapKeys() {
				if !yield(k) {
					return
				}
			}
		case Chan:
			for {
				got, ok := v.Recv()
				if !ok {
					return
				}
				if !yield(got) {
					return
				}
			}
		default:
			panic("reflect: Seq of " + v.Type().String())
		}
	}
}

// Seq2 gives a sequence of 2 values for each element.
func (v Value) Seq2() iter.Seq2[Value, Value] {
	return func(yield func(Value, Value) bool) {
		switch v.Kind() {
		case Slice, Array, String:
			for i := range v.Len() {
				if !yield(Value{value: dart.WithInt(intDesc(), int64(i))}, v.Index(i)) {
					return
				}
			}
		case Map:
			for _, k := range v.MapKeys() {
				if !yield(k, v.MapIndex(k)) {
					return
				}
			}
		default:
			panic("reflect: Seq2 of " + v.Type().String())
		}
	}
}

// intDesc gives the handle of the type int.
func intDesc() int { return TypeOf(0).handle() }

// Bits gives the count of bits of a sized arithmetic type. Each other type
// panics, exactly as in Go.
//
// An int, a uint, and a uintptr each hold 64 bits, because a Dart int holds 64
// bits on the VM and with AOT.
func (t *rtype) Bits() int {
	switch t.Kind() {
	case Int8, Uint8:
		return 8
	case Int16, Uint16:
		return 16
	case Int32, Uint32, Float32:
		return 32
	case Int, Int64, Uint, Uint64, Uintptr, Float64, Complex64:
		return 64
	case Complex128:
		return 128
	}
	panic("reflect: Bits of non-arithmetic Type " + t.String())
}

// FieldByIndex gives the nested field that each index of the list names. An
// index that names a pointer to a struct reads the type behind the pointer.
func (t *rtype) FieldByIndex(index []int) StructField {
	if len(index) == 0 {
		panic("reflect: FieldByIndex of an empty index")
	}
	var f StructField
	typ := Type(t)
	for i, x := range index {
		if i > 0 {
			if typ.Kind() == Pointer && typ.Elem().Kind() == Struct {
				typ = typ.Elem()
			}
		}
		f = typ.Field(x)
		typ = f.Type
	}
	f.Index = index
	return f
}

// Bytes gives the byte slice that v holds.
//
// The result reads the same object that v holds, thus a write through it
// reaches v. A value of an array of bytes needs an address, exactly as in Go.
func (v Value) Bytes() []byte {
	switch v.Kind() {
	case Slice:
		if v.Type().Elem().Kind() != Uint8 {
			panic("reflect.Value.Bytes of " + v.Type().String())
		}
		return v.Interface().([]byte)
	case Array:
		if v.Type().Elem().Kind() != Uint8 {
			panic("reflect.Value.Bytes of " + v.Type().String())
		}
		out := make([]byte, v.Len())
		for i := range out {
			out[i] = byte(v.Index(i).Uint())
		}
		return out
	}
	panic("reflect.Value.Bytes of " + v.Type().String())
}

// SetBytes writes a byte slice into the location that v names.
func (v Value) SetBytes(x []byte) {
	if v.Kind() != Slice || v.Type().Elem().Kind() != Uint8 {
		panic("reflect.Value.SetBytes of " + v.Type().String())
	}
	v.Set(ValueOf(x))
}

// SetLen writes the length of the slice that v names. The capacity holds, and
// the elements between the 2 lengths keep their values.
func (v Value) SetLen(n int) {
	if v.Kind() != Slice {
		panic("reflect: SetLen of " + v.Type().String())
	}
	if n < 0 || n > v.Cap() {
		panic("reflect: SetLen: slice length out of range")
	}
	v.Set(v.Slice3(0, n, v.Cap()))
}

// Grow raises the capacity of the slice that v names, thus n more elements fit
// with no further allocation. The length holds.
func (v Value) Grow(n int) {
	if v.Kind() != Slice {
		panic("reflect: Grow of " + v.Type().String())
	}
	if n < 0 {
		panic("reflect.Value.Grow: negative len")
	}
	length := v.Len()
	if length+n <= v.Cap() {
		return
	}
	// Go raises the capacity by more than the request, thus a loop of single
	// steps stays linear.
	capacity := v.Cap()*2 + n
	if capacity < length+n {
		capacity = length + n
	}
	grown := MakeSlice(v.Type(), length, capacity)
	Copy(grown, v)
	v.Set(grown)
}
