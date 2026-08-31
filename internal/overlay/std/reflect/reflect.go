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
	// Offset holds the count of bytes from the start of the struct. Read
	// Type.Size.
	Offset uintptr
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
	return StructField{
		Name:   name,
		Tag:    StructTag(tag),
		Type:   Type{desc: desc},
		Offset: t.offsetOf(i),
	}
}

// offsetOf gives the count of bytes from the start of the struct to the field.
// Read Type.Size.
func (t Type) offsetOf(index int) uintptr {
	offset := uintptr(0)
	for i := 0; i <= index; i++ {
		_, _, _, desc := dart.FieldInfo(t.desc, i)
		field := Type{desc: desc}
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

// A Method describes 1 exported method of a type.
type Method struct {
	Name  string
	Index int
}

// NumMethod gives the count of the exported methods of the type.
func (t Type) NumMethod() int { return dart.DescNumMethod(t.desc) }

// Method gives 1 exported method of the type, in the order of the name.
func (t Type) Method(i int) Method {
	return Method{Name: dart.DescMethodName(t.desc, i), Index: i}
}

// Size gives the count of bytes that a value of the type holds in Go.
//
// The Dart output holds no memory layout. The result follows the rules of a
// 64-bit machine, exactly as the compiler of Go computes them.
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
	case Struct:
		size, _ := t.layout()
		return size
	}
	panic("reflect: Size of " + t.String())
}

// Align gives the count of bytes of the alignment of a value of the type.
func (t Type) Align() int {
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
func (t Type) FieldAlign() int { return t.Align() }

// layout gives the size and the alignment of a struct type.
//
// Go puts each field at an offset that its alignment allows, and it rounds the
// size up to the alignment of the struct. A struct whose last field holds 0
// bytes takes 1 more byte, thus a pointer to the end stays inside the value.
func (t Type) layout() (uintptr, int) {
	count := dart.FieldCount(t.desc)
	offset := uintptr(0)
	align := 1
	last := uintptr(0)
	for i := 0; i < count; i++ {
		_, _, _, desc := dart.FieldInfo(t.desc, i)
		field := Type{desc: desc}
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
func (v Value) Type() Type {
	if v.read() == nil && v.typ.desc != 0 {
		return v.typ
	}
	return TypeOf(v.read())
}

// Kind gives the class of the type of the value.
func (v Value) Kind() Kind {
	if v.read() == nil {
		if v.typ.desc != 0 {
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
	return Value{value: dart.MapIndex(v.read(), key.read())}
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
