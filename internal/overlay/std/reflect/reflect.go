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

import "syscall/dart"

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

// A Type describes a Go type. Go gives an interface, and the overlay gives a
// struct that holds a handle of the type descriptor of the runtime.
type Type struct {
	desc int
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
	Tag  StructTag
	Type Type
	// Anonymous marks an embedded field.
	Anonymous bool
}

// TypeOf gives the type of the value inside an interface.
func TypeOf(v any) Type {
	return Type{desc: dart.DescOf(v)}
}

// String gives the text of the type. It holds the name of the package, and not
// the import path.
func (t Type) String() string {
	text, _, _ := dart.DescInfo(t.desc)
	return text
}

// Name gives the name of a named type, and it gives an empty string for each
// other type.
func (t Type) Name() string {
	text := t.String()
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == '.' {
			return text[i+1:]
		}
		if text[i] == ']' || text[i] == '*' || text[i] == ' ' {
			return ""
		}
	}
	switch t.Kind() {
	case Array, Chan, Func, Interface, Map, Pointer, Slice:
		return ""
	}
	return text
}

// Kind gives the class of the type.
func (t Type) Kind() Kind {
	_, kind, _ := dart.DescInfo(t.desc)
	return Kind(kind)
}

// Comparable reports whether Go compares 2 values of the type with ==.
func (t Type) Comparable() bool {
	_, _, comparable := dart.DescInfo(t.desc)
	return comparable
}

// Elem gives the element type of a pointer, a slice, an array, a channel, or a
// map. Each other type panics, exactly as in Go.
func (t Type) Elem() Type {
	elem := dart.DescElem(t.desc)
	if elem == 0 {
		panic("reflect: Elem of " + t.String())
	}
	return Type{desc: elem}
}

// Len gives the count of the elements of an array type. Each other type panics,
// exactly as in Go.
func (t Type) Len() int {
	if t.Kind() != Array {
		panic("reflect: Len of " + t.String())
	}
	return dart.DescLength(t.desc)
}

// NumField gives the count of the fields of a struct type.
func (t Type) NumField() int {
	if t.Kind() != Struct {
		panic("reflect: NumField of " + t.String())
	}
	return dart.FieldCount(t.desc)
}

// Field gives 1 field of a struct type.
func (t Type) Field(i int) StructField {
	if t.Kind() != Struct {
		panic("reflect: Field of " + t.String())
	}
	name, tag, _, desc := dart.FieldInfo(t.desc, i)
	return StructField{Name: name, Tag: StructTag(tag), Type: Type{desc: desc}}
}

// Key gives the key type of a map type. Each other type panics, exactly as in
// Go.
func (t Type) Key() Type {
	key := dart.DescKey(t.desc)
	if key == 0 {
		panic("reflect: Key of " + t.String())
	}
	return Type{desc: key}
}

// NumIn gives the count of the parameters of a function type.
func (t Type) NumIn() int {
	if t.Kind() != Func {
		panic("reflect: NumIn of " + t.String())
	}
	return dart.DescNumIn(t.desc)
}

// In gives the type of 1 parameter of a function type.
func (t Type) In(i int) Type {
	if t.Kind() != Func {
		panic("reflect: In of " + t.String())
	}
	return Type{desc: dart.DescIn(t.desc, i)}
}

// NumOut gives the count of the results of a function type.
func (t Type) NumOut() int {
	if t.Kind() != Func {
		panic("reflect: NumOut of " + t.String())
	}
	return dart.DescNumOut(t.desc)
}

// Out gives the type of 1 result of a function type.
func (t Type) Out(i int) Type {
	if t.Kind() != Func {
		panic("reflect: Out of " + t.String())
	}
	return Type{desc: dart.DescOut(t.desc, i)}
}

// IsVariadic reports whether the last parameter of a function type holds a
// list. The description of the output holds no such flag, thus the result is
// always false.
func (t Type) IsVariadic() bool { return false }

// Size gives the count of bytes that a value of the type holds in Go.
//
// A struct of Go holds a padding that follows the alignment of the machine. The
// Dart output holds no memory layout, thus the size of a struct panics.
func (t Type) Size() uintptr {
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
	}
	panic("reflect: Size of " + t.String())
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
func Zero(t Type) Value { return Value{value: dart.DescZero(t.desc)} }

// New gives a Value that holds a pointer to a fresh zero value of the type.
func New(t Type) Value { return Value{value: dart.NewValue(t.desc)} }

// MakeSlice gives a slice of the type, with the length and the capacity. The
// overlay gives each slice a capacity that is equal to its length.
func MakeSlice(t Type, length, capacity int) Value {
	if t.Kind() != Slice {
		panic("reflect: MakeSlice of " + t.String())
	}
	if length < 0 || capacity < length {
		panic("reflect: MakeSlice with a wrong length")
	}
	return Value{value: dart.MakeValue(t.desc, length)}
}

// MakeMap gives an empty map of the type.
func MakeMap(t Type) Value {
	if t.Kind() != Map {
		panic("reflect: MakeMap of " + t.String())
	}
	return Value{value: dart.MakeMap(t.desc)}
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
func (v Value) Type() Type { return TypeOf(v.read()) }

// Kind gives the class of the type of the value.
func (v Value) Kind() Kind {
	if v.read() == nil {
		return Invalid
	}
	return v.Type().Kind()
}

// Elem gives the value behind a pointer. The result holds the pointer, thus Set
// writes through it.
func (v Value) Elem() Value {
	p := v.read()
	return Value{value: dart.PointerElem(p), owner: p, loc: locPointer}
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
		index: i, loc: locField}
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
	out := make([]Value, len(raw))
	for i, r := range raw {
		out[i] = Value{value: r}
	}
	return out
}

// SetMapIndex writes 1 entry of a map. A key with no value drops the entry.
func (v Value) SetMapIndex(key, value Value) {
	dart.MapSet(v.read(), key.read(), value.read(), !value.IsValid())
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
