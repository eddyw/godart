//go:build dart

// This file includes code derived from the Go reflect package.
// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2016 The Go Authors. All rights reserved.
// Use of the Go-derived portions is governed by the BSD-style
// license in ../LICENSE.

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
	"strconv"
	"syscall/dart"
	"unicode"
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
// The type is an unsigned number, exactly as in Go, and each constant holds
// the same number as the constant of Go. Thus a table that an index of a Kind
// reads holds the same layout, and `encoding/gob` compiles.
type Kind uint

// The kinds of Go.
const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

// Ptr is the old name of Pointer.
const Ptr = Pointer

// kindNames holds the text of each kind, in the order of the constants.
var kindNames = []string{
	"invalid",
	"bool",
	"int",
	"int8",
	"int16",
	"int32",
	"int64",
	"uint",
	"uint8",
	"uint16",
	"uint32",
	"uint64",
	"uintptr",
	"float32",
	"float64",
	"complex64",
	"complex128",
	"array",
	"chan",
	"func",
	"interface",
	"map",
	"ptr",
	"slice",
	"string",
	"struct",
	"unsafe.Pointer",
}

// String gives the name of the kind.
func (k Kind) String() string {
	if k < Kind(len(kindNames)) {
		return kindNames[k]
	}
	return "kind" + itoa(int(k))
}

// kindByName gives the Kind of the text that the type descriptor of the
// runtime holds. A text that names no kind gives Invalid.
func kindByName(text string) Kind {
	switch text {
	case "bool":
		return Bool
	case "int":
		return Int
	case "int8":
		return Int8
	case "int16":
		return Int16
	case "int32":
		return Int32
	case "int64":
		return Int64
	case "uint":
		return Uint
	case "uint8":
		return Uint8
	case "uint16":
		return Uint16
	case "uint32":
		return Uint32
	case "uint64":
		return Uint64
	case "uintptr":
		return Uintptr
	case "float32":
		return Float32
	case "float64":
		return Float64
	case "complex64":
		return Complex64
	case "complex128":
		return Complex128
	case "array":
		return Array
	case "chan":
		return Chan
	case "func":
		return Func
	case "interface":
		return Interface
	case "map":
		return Map
	case "ptr":
		return Pointer
	case "slice":
		return Slice
	case "string":
		return String
	case "struct":
		return Struct
	case "unsafe.Pointer":
		return UnsafePointer
	}
	return Invalid
}

// A Type describes a Go type. It compares with nil, exactly as it does in Go.
//
// The overlay holds 1 implementation, and that implementation is unexported.
// A Dart library holds each private name of its own, thus a method of an
// unexported type reaches no other package as a plain call. An interface
// dispatches through the method table, and each package then reads it.
type Type interface {
	String() string
	Name() string
	PkgPath() string
	Kind() Kind
	Comparable() bool
	Elem() Type
	Key() Type
	Len() int
	NumField() int
	Field(i int) StructField
	Fields() iter.Seq[StructField]
	FieldByName(name string) (StructField, bool)
	FieldByNameFunc(func(string) bool) (StructField, bool)
	FieldByIndex(index []int) StructField
	Bits() int
	NumIn() int
	In(i int) Type
	Ins() iter.Seq[Type]
	NumOut() int
	Out(i int) Type
	Outs() iter.Seq[Type]
	IsVariadic() bool
	NumMethod() int
	Method(i int) Method
	Methods() iter.Seq[Method]
	MethodByName(name string) (Method, bool)
	Size() uintptr
	Align() int
	FieldAlign() int
	AssignableTo(u Type) bool
	ConvertibleTo(u Type) bool
	Implements(u Type) bool
	ChanDir() ChanDir
	CanSeq() bool
	CanSeq2() bool
	OverflowComplex(complex128) bool
	OverflowFloat(float64) bool
	OverflowInt(int64) bool
	OverflowUint(uint64) bool
	handle() int
	offsetOf(index int) uintptr
}

// An rtype holds the handle of 1 type descriptor of the runtime.
type rtype struct {
	desc int
	// kindPlus1 holds the Kind of the type plus 1, and 0 marks a field that no
	// read filled yet. The name of a handle never changes, thus the kind of a
	// handle never changes, and 1 read answers each later read.
	//
	// The package interns 1 rtype for each handle, thus the cache holds for
	// the whole program.
	kindPlus1 Kind
}

// types interns 1 Type for each handle, thus 2 reads of the same type give 2
// equal values, as they do in Go.
//
// The index of the table is the handle minus 1. A handle is a small whole
// number that counts up, thus the table needs no hash. The element holds the
// interface value, and not the pointer, thus 1 read builds it and each later
// read gives the same value with no allocation.
var types []Type

// typeOf gives the Type of a handle. The handle 0 names no type, and it gives
// nil, exactly as a nil interface does in Go.
func typeOf(handle int) Type {
	if handle == 0 {
		return nil
	}
	if handle <= len(types) {
		if t := types[handle-1]; t != nil {
			return t
		}
	}
	for len(types) < handle {
		types = append(types, nil)
	}
	t := &rtype{desc: handle}
	types[handle-1] = t
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
		quoted := string(tag[:i+1])
		tag = tag[i+1:]
		if key == name {
			value, err := strconv.Unquote(quoted)
			if err != nil {
				break
			}
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
	if t.kindPlus1 != 0 {
		return t.kindPlus1 - 1
	}
	_, kind, _ := dart.DescInfo(t.handle())
	k := kindByName(kind)
	t.kindPlus1 = k + 1
	return k
}

// PkgPath gives the import path of the package of a named type, and it gives
// an empty text for each other type. A type of the package main gives `main`,
// exactly as in Go.
func (t *rtype) PkgPath() string {
	if t.Name() == "" {
		return ""
	}
	return dart.DescPkgPath(t.handle())
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
		// An unnamed struct type holds no package of its own, thus the field
		// carries the path of the package that declares it.
		pkg = dart.FieldPkgPath(t.handle(), i)
		if pkg == "" {
			pkg = dart.DescPkgPath(t.handle())
		}
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

func (t *rtype) offsetOf(index int) uintptr {
	return dart.DescOffset(t.handle(), index)
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
//
// Go also holds the field Func, and that field stays out. It gives a func value
// whose first parameter is the receiver, and the method table of the runtime
// holds no such value. Read docs/parity.md.
type Method struct {
	Name string
	// PkgPath holds the import path of an unexported method, and it is empty
	// for an exported method. The overlay reads no unexported method, thus the
	// field is always empty.
	PkgPath string
	// Type holds the function type of the method. The type of a method of a
	// concrete type carries the receiver in front of each parameter, and the
	// type of a method of an interface type carries no receiver.
	Type Type
	// Func holds a func value of the method, with the receiver in front of
	// each parameter. A method of an interface type holds the zero Value.
	//
	// Call of the value runs the method. The signature carries no factory of a
	// Dart type, thus Interface of the value gives a func value that no type
	// assertion reads.
	Func  Value
	Index int
}

// IsExported reports whether the method is exported.
func (m Method) IsExported() bool { return m.PkgPath == "" }

// NumMethod gives the method count. Interface types include unexported methods.
func (t *rtype) NumMethod() int { return dart.DescNumMethod(t.handle()) }

// Method gives method i in name order.
func (t *rtype) Method(i int) Method {
	if i < 0 || i >= t.NumMethod() {
		panic("reflect: Method index out of range")
	}
	typ := typeOf(dart.DescMethodType(t.handle(), i))
	m := Method{
		Name:    dart.DescMethodName(t.handle(), i),
		PkgPath: dart.DescMethodPkgPath(t.handle(), i),
		Type:    typ,
		Index:   i,
	}
	if fn := dart.DescMethodFunc(t.handle(), i); fn != nil {
		m.Func = Value{value: fn, typ: typ, ok: true}
	}
	return m
}

// MethodByName gives the method with this name, or the zero Method and false.
func (t *rtype) MethodByName(name string) (Method, bool) {
	for i := range t.NumMethod() {
		if m := t.Method(i); m.Name == name {
			return m, true
		}
	}
	return Method{}, false
}

// Size gives the count of bytes that a value of the type holds in Go.
//
// The Dart output holds no memory layout. The result follows the rules of a
// 64-bit machine, exactly as the compiler of Go computes them.
func (t *rtype) Size() uintptr { return dart.DescSize(t.handle()) }

// Align gives the count of bytes of the alignment of a value of the type.
func (t *rtype) Align() int { return dart.DescAlign(t.handle()) }

// FieldAlign gives the alignment of the type inside a struct.
func (t *rtype) FieldAlign() int { return t.Align() }

// The 4 forms of the location that a Value names. A Value of the form locNone
// holds no location, thus Set panics.
const (
	locNone = iota
	locPointer
	locField
	locIndex
)

// The 2 read only marks of a Value. Go refuses a write and a read through an
// interface for a value that an unexported field gave.
//
// The 2 marks differ in the way that they reach a field of the value. A field
// of a struct that an unexported and embedded field gave is writable, and a
// field of each other unexported field is not. Go holds the same rule, and
// the test TestUnmarshalEmbeddedUnexported of encoding/json reads it.
const (
	roSticky = 1 << iota
	roEmbed
)

// A Value holds a Go value. A Value that Elem, Field, or Index read also holds
// its location, thus Set writes through it.
type Value struct {
	value any
	owner any
	index int
	loc   int
	// ro holds the read only marks. A value of the mark 0 accepts a write and
	// a read through an interface.
	ro uint8
	// typ holds the static type when the value itself is nil, for example
	// the element of a pointer to an interface.
	typ Type
	// ok marks a Value that holds a value. Only the zero Value holds false.
	//
	// A nil interface, a nil pointer, and a nil map each read as nil in Dart,
	// thus the value alone cannot answer IsValid. Go answers true for an entry
	// of a map that holds a nil interface, and false for a key that the map
	// does not hold.
	ok bool

	unexportedMethod bool
}

// ValueOf gives the value inside an interface. A nil interface gives the zero
// Value, exactly as Go does.
func ValueOf(v any) Value {
	if v == nil {
		return Value{}
	}
	return Value{value: v, ok: true}
}

// Indirect gives the value behind a pointer. A value that is no pointer gives
// itself.
func Indirect(v Value) Value {
	if v.Kind() != Pointer {
		return v
	}
	return v.Elem()
}

// Zero gives the zero value of a type.
// The static type travels with the value, because the zero value of an
// interface type holds no descriptor of its own. A Value that holds neither a
// value nor a type is the zero Value, and MakeFunc rejects such a result.
func Zero(t Type) Value {
	return Value{value: dart.DescZero(t.handle()), typ: t, ok: true}
}

// New gives a Value that holds a pointer to a fresh zero value of the type.
func New(t Type) Value { return Value{value: dart.NewValue(t.handle()), ok: true} }

// MakeSlice gives a slice of the type, with the length and the capacity. The
// overlay gives each slice a capacity that is equal to its length.
func MakeSlice(t Type, length, capacity int) Value {
	if t.Kind() != Slice {
		panic("reflect: MakeSlice of " + t.String())
	}
	if length < 0 || capacity < length {
		panic("reflect: MakeSlice with a wrong length")
	}
	return Value{value: dart.MakeValue(t.handle(), length, capacity), ok: true}
}

// MakeMap gives an empty map of the type.
func MakeMap(t Type) Value {
	if t.Kind() != Map {
		panic("reflect: MakeMap of " + t.String())
	}
	return Value{value: dart.MakeMap(t.handle()), ok: true}
}

// TypeAssert reads the value inside v as the type T. The flag is false when the
// value holds another type.
func TypeAssert[T any](v Value) (T, bool) {
	if !v.IsValid() {
		panic(&ValueError{Method: "reflect.TypeAssert", Kind: Invalid})
	}
	if v.ro != 0 {
		panic("reflect.TypeAssert: cannot return value obtained from unexported field or method")
	}
	if v.unexportedMethod {
		panic("reflect: TypeAssert of unexported method")
	}
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

// IsValid reports whether v holds a value. Only the zero Value gives false.
//
// A read of the location cannot answer it. A nil interface, a nil pointer, and
// a nil map each read as nil, and Go holds each 1 valid. An entry of a map that
// holds a nil interface is valid, and a key that the map does not hold gives
// the zero Value.
func (v Value) IsValid() bool { return v.ok }

func init() { dart.SetValueHook(readValue) }

// readValue gives the value that a Value holds. The package fmt writes that
// value, and not the text of the String method of the Value.
func readValue(v any) (any, bool) {
	value, ok := v.(Value)
	if !ok || !value.ok {
		return nil, false
	}
	return value.read(), true
}

// Interface gives the value of v inside an interface.
func (v Value) Interface() any {
	if !v.IsValid() {
		panic(&ValueError{Method: "reflect.Value.Interface", Kind: Invalid})
	}
	if v.ro != 0 {
		panic("reflect.Value.Interface: cannot return value obtained from unexported field or method")
	}
	if v.unexportedMethod {
		panic("reflect: Interface of unexported method")
	}
	return dart.CopyInterface(v.read())
}

// Type gives the type of the value.
func (v Value) Type() Type {
	if v.typ != nil && v.staticType() {
		return v.typ
	}
	return TypeOf(v.read())
}

// staticType reports whether the type of the location wins over the descriptor
// of the value inside it.
//
// A location of an interface type holds a value of another type, and Go reads
// the type of the location. A location of each other type holds a value of that
// type alone, thus the 2 answers are equal there. A nil value carries no
// descriptor, and only the location gives the type.
func (v Value) staticType() bool {
	return v.value == nil || v.typ.Kind() == Interface
}

// Kind gives the class of the type of the value.
//
// The body reads the location 1 time and it reads the static kind 1 time.
// Each read of a location behind a pointer builds an interface value, thus a
// second read costs an allocation.
func (v Value) Kind() Kind {
	if v.typ != nil {
		k := v.typ.Kind()
		if v.value == nil || k == Interface {
			return k
		}
	}
	return v.kindAt(v.read())
}

// kindAt gives the same answer as Kind for a value that a read already gave.
// A method that holds the value of the location reads this one, thus the
// location gives its value 1 time.
func (v Value) kindAt(got any) Kind {
	if v.typ != nil {
		k := v.typ.Kind()
		if v.value == nil || k == Interface {
			return k
		}
	}
	if got == nil {
		return Invalid
	}
	return TypeOf(got).Kind()
}

// typeAt gives the same answer as Type for a value that a read already gave.
func (v Value) typeAt(got any) Type {
	if v.typ != nil && v.staticType() {
		return v.typ
	}
	return TypeOf(got)
}

// Elem gives the value behind a pointer. The result holds the pointer, thus Set
// writes through it.
//
// Elem of an interface value gives the value inside it, with the type of that
// value. Go gives no location there, thus Set of the result reports an error.
func (v Value) Elem() Value {
	v.mustBe("Elem", Interface, Pointer)
	p := v.read()
	kind := v.kindAt(p)
	if kind == Interface {
		// Go gives the zero Value for the value inside a nil interface.
		if p == nil {
			return Value{}
		}
		return Value{value: p, ro: v.ro, ok: true}
	}
	if kind == Pointer && dart.ValueIsNil(p) {
		// Go gives the zero Value for the element of a nil pointer. A nil
		// pointer inside an interface carries the type, thus the value that
		// the read gives is not nil and only ValueIsNil answers.
		return Value{}
	}
	out := Value{value: dart.PointerElem(p), owner: p, loc: locPointer, ro: v.ro, ok: true}
	if kind == Pointer {
		out.typ = v.typeAt(p).Elem()
	}
	return out
}

// Len gives the length of a slice, an array, a string, or a map.
func (v Value) Len() int {
	v.mustBe("Len", Array, Chan, Map, Slice, String, Pointer)
	if v.Kind() == Pointer {
		if v.Type().Elem().Kind() != Array {
			panic("reflect: call of reflect.Value.Len on ptr to non-array Value")
		}
		return v.Type().Elem().Len()
	}
	return dart.ValueLen(v.read())
}

// Index gives 1 element of a slice, an array, or a string.
// Elements of slices and addressable arrays retain their location.
func (v Value) Index(i int) Value {
	v.mustBe("Index", Array, Slice, String)
	owner := v.read()
	ro := v.ro
	if ro != 0 {
		ro = roSticky
	}
	out := Value{value: dart.ValueIndex(owner, i), ro: ro, ok: true}
	if v.kindAt(owner) != String {
		if v.kindAt(owner) == Slice || v.CanAddr() {
			out.owner, out.index, out.loc = owner, i, locIndex
		}
		out.typ = v.typeAt(owner).Elem()
	}
	return out
}

// NumField gives the count of the fields of a struct value.
func (v Value) NumField() int {
	v.mustBe("NumField", Struct)
	return v.Type().NumField()
}

// Field gives 1 field of a struct value. It retains the addressability of v.
func (v Value) Field(i int) Value {
	v.mustBe("Field", Struct)
	owner := v.read()
	f := v.typeAt(owner).Field(i)
	// Only the sticky mark reaches a field. Thus a field of a struct that an
	// unexported and embedded field gave accepts a write.
	ro := v.ro & roSticky
	if !f.IsExported() {
		if f.Anonymous {
			ro |= roEmbed
		} else {
			ro |= roSticky
		}
	}
	out := Value{value: dart.FieldRead(owner, i), ro: ro, typ: f.Type, ok: true}
	if v.CanAddr() {
		out.owner, out.index, out.loc = owner, i, locField
	}
	return out
}

// CanSet reports whether Set writes through v.
func (v Value) CanSet() bool { return v.loc != locNone && v.ro == 0 }

// Set writes x into the location that v names. A Value that holds no location
// panics.
func (v Value) Set(x Value) {
	v.mustBeAssignable("reflect.Value.Set")
	if x.ro != 0 {
		panic("reflect: reflect.Value.Set using value obtained using unexported field")
	}
	if !x.Type().AssignableTo(v.Type()) {
		panic("reflect.Set: value of type " + x.Type().String() + " is not assignable to type " + v.Type().String())
	}
	v.write(x.read())
}

func (v Value) write(value any) {
	switch v.loc {
	case locPointer:
		dart.PointerSet(v.owner, value)
	case locField:
		dart.FieldWrite(v.owner, v.index, value)
	case locIndex:
		dart.ValueSetIndex(v.owner, v.index, value)
	}
}

func (v Value) mustBe(method string, kinds ...Kind) {
	got := v.Kind()
	for _, kind := range kinds {
		if got == kind {
			return
		}
	}
	panic(&ValueError{Method: "reflect.Value." + method, Kind: got})
}

func (v Value) mustBeAssignable(method string) {
	if !v.IsValid() {
		panic(&ValueError{Method: method, Kind: Invalid})
	}
	if v.ro != 0 {
		panic("reflect: " + method + " using value obtained using unexported field")
	}
	if v.loc == locNone {
		panic("reflect: " + method + " using unaddressable value")
	}
}

// Int gives the value of a signed integer kind.
func (v Value) Int() int64 {
	v.mustBe("Int", Int, Int8, Int16, Int32, Int64)
	return dart.AsInt(v.read())
}

// Uint gives the value of an unsigned integer kind.
func (v Value) Uint() uint64 {
	v.mustBe("Uint", Uint, Uint8, Uint16, Uint32, Uint64, Uintptr)
	return uint64(dart.AsInt(v.read()))
}

// Float gives the value of a float kind.
func (v Value) Float() float64 {
	v.mustBe("Float", Float32, Float64)
	return dart.AsFloat(v.read())
}

// Bool gives the value of the kind bool.
func (v Value) Bool() bool {
	v.mustBe("Bool", Bool)
	return dart.AsBool(v.read())
}

// String gives the text of a value of the kind string. Each other kind gives a
// description of the value, exactly as in Go.
func (v Value) String() string {
	got := v.read()
	if v.kindAt(got) == String {
		return dart.AsString(got)
	}
	if got == nil {
		return "<invalid Value>"
	}
	return "<" + v.typeAt(got).String() + " Value>"
}

// NumMethod gives the method count. Interface values include unexported methods.
func (v Value) NumMethod() int {
	if !v.IsValid() {
		panic(&ValueError{Method: "reflect.Value.NumMethod", Kind: Invalid})
	}
	return v.Type().NumMethod()
}

// Method binds method i from the static type of v.
func (v Value) Method(i int) Value {
	if !v.IsValid() {
		panic(&ValueError{Method: "reflect.Value.Method", Kind: Invalid})
	}
	t := v.Type()
	if i < 0 || i >= t.NumMethod() {
		panic("reflect: Method index out of range")
	}
	return Value{
		value: dart.MethodValue(v.read(), t.handle(), i), ro: v.ro, ok: true,
		unexportedMethod: dart.DescMethodPkgPath(t.handle(), i) != "",
	}
}

// Call runs the function that v holds and gives each result.
// The scheduler retains the call while its function waits.
func (v Value) Call(in []Value) []Value {
	v.mustBe("Call", Func)
	v.mustAllowCall("Call")
	v.mustAllowArguments("Call", in)
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
		out[i] = Value{value: r, typ: t.Out(i), ok: true}
	}
	return out
}

func (v Value) mustAllowArguments(method string, in []Value) {
	if v.IsNil() {
		panic("reflect.Value.Call: call of nil function")
	}
	t := v.Type()
	n := t.NumIn()
	variadic := t.IsVariadic()
	if method == "CallSlice" {
		if !variadic {
			panic("reflect: CallSlice of non-variadic function")
		}
	} else if variadic {
		n--
	}
	if len(in) < n {
		panic("reflect: " + method + " with too few input arguments")
	}
	if (method == "CallSlice" || !variadic) && len(in) > n {
		panic("reflect: " + method + " with too many input arguments")
	}
	for _, arg := range in {
		if !arg.IsValid() {
			panic("reflect: " + method + " using zero Value argument")
		}
	}
	for i := 0; i < n; i++ {
		if !in[i].Type().AssignableTo(t.In(i)) {
			panic("reflect: " + method + " using " + in[i].Type().String() + " as type " + t.In(i).String())
		}
	}
	if method == "Call" && variadic {
		elem := t.In(n).Elem()
		for _, arg := range in[n:] {
			if !arg.Type().AssignableTo(elem) {
				panic("reflect: cannot use " + arg.Type().String() + " as type " + elem.String() + " in Call")
			}
			if arg.ro != 0 {
				panic("reflect: reflect.Value.Set using value obtained using unexported field")
			}
			if arg.unexportedMethod {
				panic("reflect: reflect.Set of unexported method")
			}
		}
	}
	for _, arg := range in[:n] {
		if arg.ro != 0 {
			panic("reflect: reflect.Value." + method + " using value obtained using unexported field")
		}
		if arg.unexportedMethod {
			panic("reflect: reflect.Value.Call of unexported method")
		}
	}
}

func (v Value) mustAllowCall(method string) {
	if v.ro != 0 {
		panic("reflect: reflect.Value." + method + " using value obtained using unexported field")
	}
	if v.unexportedMethod {
		panic("reflect: " + method + " of unexported method")
	}
}

// SetMapIndex writes 1 entry of a map. A key with no value drops the entry.
func (v Value) SetMapIndex(key, value Value) {
	v.mustBe("SetMapIndex", Map)
	if !key.Type().AssignableTo(v.Type().Key()) {
		panic("reflect.Value.SetMapIndex: value of type " + key.Type().String() + " is not assignable to type " + v.Type().Key().String())
	}
	got := value.read()
	// Go removes the entry only for the zero Value. A valid value that holds a
	// nil interface writes the entry, and the key stays. The read of such a
	// value also gives nil, thus the test reads the mark of the Value.
	// `{"T":null}` into a map[string]any needs that rule.
	del := !value.ok
	if got != nil && (value.Kind() == Struct || value.Kind() == Array) {
		// Go copies the value into the map. The Dart form of a struct and of an
		// array is an object, thus a caller that reuses 1 cell for each entry
		// would give each key the same object. The map decoder of
		// encoding/json/v2 reuses exactly 1 cell, and 2 keys then held 1 value.
		//
		// Set walks each level of the value now, thus a field of a struct type
		// and an element of an array each hold a copy of their own.
		cell := New(value.Type()).Elem()
		cell.Set(value)
		got = cell.read()
	}
	dart.MapSet(v.read(), key.read(), got, del)
}

// IsNil reports whether the value of a kind that can hold nil holds nil. Each
// other kind panics.
func (v Value) IsNil() bool {
	got := v.read()
	kind := v.kindAt(got)
	switch kind {
	case Interface:
		// A nil interface holds no type and no value, and it reads as nil.
		// An interface that holds a nil pointer is not nil, thus the value
		// inside decides nothing here.
		return got == nil
	case Chan, Func, Map, Pointer, Slice, UnsafePointer:
		if got == nil {
			return true
		}
		return dart.ValueIsNil(got)
	}
	panic(&ValueError{Method: "reflect.Value.IsNil", Kind: kind})
}

// MapKeys gives the keys of a map value. The order follows the insertion.
func (v Value) MapKeys() []Value {
	v.mustBe("MapKeys", Map)
	raw := dart.MapKeys(v.read())
	out := make([]Value, len(raw))
	for i, k := range raw {
		out[i] = Value{value: k, ok: true}
	}
	return out
}

// MapIndex gives the value of 1 key of a map value. A key that the map does
// not hold gives the zero Value.
//
// An entry that holds a nil interface reads as nil, exactly as a key that the
// map does not hold. The test of the key thus stands in front of the read.
func (v Value) MapIndex(key Value) Value {
	v.mustBe("MapIndex", Map)
	if !key.Type().AssignableTo(v.Type().Key()) {
		panic("reflect.Value.MapIndex: value of type " + key.Type().String() + " is not assignable to type " + v.Type().Key().String())
	}
	m := v.read()
	k := key.read()
	if !dart.MapHas(m, k) {
		return Value{}
	}
	return Value{value: dart.MapIndex(m, k), typ: v.typeAt(m).Elem(), ok: true}
}

// SetZero writes the zero value of the type into the location that v names.
func (v Value) SetZero() { v.Set(Zero(v.Type())) }

// SetInt writes a whole number into the location that v names.
func (v Value) SetInt(x int64) {
	v.mustBeAssignable("reflect.Value.SetInt")
	v.mustBe("SetInt", Int, Int8, Int16, Int32, Int64)
	v.write(dart.WithInt(v.descOf(), x))
}

// SetUint writes a whole number into the location that v names.
func (v Value) SetUint(x uint64) {
	v.mustBeAssignable("reflect.Value.SetUint")
	v.mustBe("SetUint", Uint, Uint8, Uint16, Uint32, Uint64, Uintptr)
	v.write(dart.WithInt(v.descOf(), int64(x)))
}

// SetFloat writes a float into the location that v names.
func (v Value) SetFloat(x float64) {
	v.mustBeAssignable("reflect.Value.SetFloat")
	v.mustBe("SetFloat", Float32, Float64)
	v.write(dart.WithFloat(v.descOf(), x))
}

// SetBool writes a flag into the location that v names.
func (v Value) SetBool(x bool) {
	v.mustBeAssignable("reflect.Value.SetBool")
	v.mustBe("SetBool", Bool)
	v.write(dart.WithBool(v.descOf(), x))
}

// SetString writes text into the location that v names.
func (v Value) SetString(x string) {
	v.mustBeAssignable("reflect.Value.SetString")
	v.mustBe("SetString", String)
	v.write(dart.WithString(v.descOf(), x))
}

func (v Value) descOf() int { return dart.DescOf(v.read()) }

// SliceOf gives the slice type of an element type.
func SliceOf(t Type) Type { return typeOf(dart.SliceOf(t.handle())) }

// FuncOf gives the function type of a parameter list, a result list, and the
// variadic flag.
func FuncOf(in, out []Type, variadic bool) Type {
	if variadic && (len(in) == 0 || in[len(in)-1].Kind() != Slice) {
		panic("reflect.FuncOf: last arg of variadic func must be slice")
	}
	if len(in)+len(out) > 128 {
		panic("reflect.FuncOf: too many arguments")
	}
	params := make([]int, len(in))
	for i, t := range in {
		params[i] = t.handle()
	}
	results := make([]int, len(out))
	for i, t := range out {
		results[i] = t.handle()
	}
	return typeOf(dart.FuncOf(params, results, variadic))
}

// ArrayOf gives the array type of a length and an element type.
func ArrayOf(length int, t Type) Type {
	if length < 0 {
		panic("reflect: negative length passed to ArrayOf")
	}
	if size := t.Size(); size > 0 && uintptr(length) > ^uintptr(0)/size {
		panic("reflect.ArrayOf: array size would exceed virtual address space")
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

// MakeMapWithSize gives an empty map of the type. The size is a hint of Go, and
// the map of Dart needs none.
func MakeMapWithSize(t Type, n int) Value { return MakeMap(t) }

// ChanOf gives the channel type of a direction and an element type.
func ChanOf(dir ChanDir, t Type) Type {
	return typeOf(dart.ChanOf(int(dir), t.handle()))
}

// MakeChan gives a channel of the type, with the size of the buffer.
func MakeChan(t Type, buffer int) Value {
	if t.Kind() != Chan {
		panic("reflect: MakeChan of " + t.String())
	}
	if t.ChanDir() != BothDir {
		panic("reflect: MakeChan of a channel of 1 direction")
	}
	if buffer < 0 {
		panic("reflect: MakeChan with a negative buffer")
	}
	return Value{value: dart.MakeChan(t.handle(), buffer), ok: true}
}

// MakeFunc gives a func value of the function type. Each call of that value
// runs fn with the arguments, and the results of fn become the results of the
// call.
//
// The scheduler retains the call while its body waits.
func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	if typ == nil {
		panic("reflect: MakeFunc with a nil type")
	}
	if typ.Kind() != Func {
		panic("reflect: MakeFunc of " + typ.String())
	}
	return Value{value: dart.MakeFunc(typ.handle(), func(args []any) []any {
		in := make([]Value, len(args))
		for i, a := range args {
			in[i] = Value{value: a, typ: typ.In(i), ok: true}
		}
		out := fn(in)
		if len(out) != typ.NumOut() {
			panic("reflect: wrong return count from function created by MakeFunc")
		}
		res := make([]any, len(out))
		for i, o := range out {
			// Go rejects a result that holds the zero Value. Such a Value
			// carries no mark, and reflect.Zero of an interface type does.
			if !o.ok {
				panic("reflect: function created by MakeFunc using closure returned zero Value")
			}
			if o.ro != 0 {
				panic("reflect: function created by MakeFunc using closure returned value obtained from unexported field")
			}
			if !o.Type().AssignableTo(typ.Out(i)) {
				panic("reflect.MakeFunc: value of type " + o.Type().String() + " is not assignable to type " + typ.Out(i).String())
			}
			res[i] = o.read()
		}
		return res
	}), ok: true}
}

func validFieldName(name string) bool {
	for i, r := range name {
		if r == '_' || unicode.IsLetter(r) || i > 0 && unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return name != ""
}

// StructOf returns the struct type with the supplied fields.
func StructOf(fields []StructField) Type {
	names := make([]string, len(fields))
	packages := make([]string, len(fields))
	packagePath := ""
	tags := make([]string, len(fields))
	handles := make([]int, len(fields))
	anon := make([]bool, len(fields))
	seen := make(map[string]bool, len(fields))
	var size, lastZero uintptr
	align := uintptr(1)
	for i, f := range fields {
		if f.Name == "" {
			panic("reflect.StructOf: field " + itoa(i) + " has no name")
		}
		if !validFieldName(f.Name) {
			panic("reflect.StructOf: field " + itoa(i) + " has invalid name")
		}
		if f.Type == nil {
			panic("reflect.StructOf: field " + itoa(i) + " has no type")
		}
		if f.Anonymous && f.PkgPath != "" {
			panic("reflect.StructOf: field \"" + f.Name + "\" is anonymous but has PkgPath set")
		}
		if f.PkgPath == "" {
			c := f.Name[0]
			if 'a' <= c && c <= 'z' || c == '_' {
				panic("reflect.StructOf: field \"" + f.Name + "\" is unexported but missing PkgPath")
			}
		} else if packagePath == "" {
			packagePath = f.PkgPath
		} else if packagePath != f.PkgPath {
			panic("reflect.StructOf: fields with different PkgPath " + packagePath + " and " + f.PkgPath)
		}
		if f.Anonymous {
			if err := dart.EmbeddedFieldError(f.Type.handle(), i, len(fields)); err != "" {
				panic(err)
			}
		}
		if f.Name != "_" && seen[f.Name] {
			panic("reflect.StructOf: duplicate field " + f.Name)
		}
		seen[f.Name] = true
		fieldAlign := uintptr(f.Type.Align())
		offset := (size + fieldAlign - 1) &^ (fieldAlign - 1)
		if offset < size {
			panic("reflect.StructOf: struct size would exceed virtual address space")
		}
		if fieldAlign > align {
			align = fieldAlign
		}
		fieldSize := f.Type.Size()
		size = offset + fieldSize
		if size < offset {
			panic("reflect.StructOf: struct size would exceed virtual address space")
		}
		if fieldSize == 0 {
			lastZero = size
		}
		names[i] = f.Name
		packages[i] = f.PkgPath
		tags[i] = string(f.Tag)
		handles[i] = f.Type.handle()
		anon[i] = f.Anonymous
	}
	if size > 0 && lastZero == size {
		size++
		if size == 0 {
			panic("reflect.StructOf: struct size would exceed virtual address space")
		}
	}
	if (size+align-1)&^(align-1) < size {
		panic("reflect.StructOf: struct size would exceed virtual address space")
	}
	return typeOf(dart.StructOf(names, packages, tags, handles, anon))
}

// PointerTo gives the pointer type of an element type.
func PointerTo(t Type) Type { return typeOf(dart.PointerTo(t.handle())) }

// PtrTo gives the pointer type of an element type. Go marks it as older than
// PointerTo.
func PtrTo(t Type) Type { return PointerTo(t) }

// TypeFor gives the type of T.
func TypeFor[T any]() Type {
	var zero T
	if t := TypeOf(zero); t != nil {
		return t
	}
	// The zero value of an interface type holds no descriptor, thus the type
	// comes from a pointer to it.
	return TypeOf(&zero).Elem()
}

// Cap gives the capacity of a slice, of an array, or of a channel.
func (v Value) Cap() int {
	if v.Kind() == Pointer {
		if v.Type().Elem().Kind() != Array {
			panic("reflect: call of reflect.Value.Cap on ptr to non-array Value")
		}
		return v.Type().Elem().Len()
	}
	return dart.ValueCap(v.read())
}

// Append gives a slice that holds each element of s and each value of x.
func Append(s Value, x ...Value) Value {
	if s.Kind() != Slice {
		panic("reflect: Append of " + s.Type().String())
	}
	if s.ro != 0 {
		panic("reflect: reflect.Append using value obtained using unexported field")
	}
	if s.Len()+len(x) < 0 {
		panic("reflect.Value.Grow: slice overflow")
	}
	items := make([]any, len(x))
	for i, v := range x {
		if v.ro != 0 {
			panic("reflect: reflect.Value.Set using value obtained using unexported field")
		}
		if !v.IsValid() || !v.Type().AssignableTo(s.Type().Elem()) {
			panic("reflect: reflect.Value.Set with incompatible value")
		}
		items[i] = v.read()
	}
	return Value{value: dart.AppendValue(s.read(), items), ok: true}
}

// AppendSlice gives a slice that holds each element of s and each element of t.
func AppendSlice(s, t Value) Value {
	if s.Kind() != Slice {
		panic("reflect: AppendSlice of " + s.Type().String())
	}
	if t.Kind() != Slice || s.Type().Elem() != t.Type().Elem() {
		panic("reflect: AppendSlice of incompatible slices")
	}
	if s.ro != 0 || t.ro != 0 {
		panic("reflect: reflect.AppendSlice using value obtained using unexported field")
	}
	if s.Len()+t.Len() < 0 {
		panic("reflect.Value.Grow: slice overflow")
	}
	return Value{value: dart.AppendSliceValue(s.read(), t.read()), ok: true}
}

// Copy writes each element of src into dst, and it gives the count that it
// wrote. src may hold a string when dst holds a byte slice.
func Copy(dst, src Value) int {
	dk := dst.Kind()
	if dk != Array && dk != Slice {
		panic(&ValueError{Method: "reflect.Copy", Kind: dk})
	}
	// Copy is a package function, thus Go names no Value method.
	if dk == Array {
		dst.mustBeAssignable("unknown method")
	}
	if dst.ro != 0 {
		panic("reflect: unknown method using value obtained using unexported field")
	}
	sk := src.Kind()
	stringCopy := sk == String && dst.Type().Elem().Kind() == Uint8
	if sk != Array && sk != Slice && !stringCopy {
		panic(&ValueError{Method: "reflect.Copy", Kind: sk})
	}
	if src.ro != 0 {
		panic("reflect: unknown method using value obtained using unexported field")
	}
	if !stringCopy && dst.Type().Elem() != src.Type().Elem() {
		panic("reflect.Copy: " + dst.Type().Elem().String() + " != " + src.Type().Elem().String())
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
	for _, field := range VisibleFields(t) {
		if field.Name == name {
			return field, true
		}
	}
	return StructField{}, false
}

// CanSeq reports whether a range loop over a value of the type gives 1 value.
func (t *rtype) CanSeq() bool {
	switch t.Kind() {
	case Int, Int8, Int16, Int32, Int64,
		Uint, Uint8, Uint16, Uint32, Uint64, Uintptr,
		Array, Slice, Chan, String, Map:
		return true
	case Pointer:
		return t.Elem().Kind() == Array
	case Func:
		return rangeFuncType(t, 1)
	}
	return false
}

// CanSeq2 reports whether a range loop over a value of the type gives 2 values.
func (t *rtype) CanSeq2() bool {
	switch t.Kind() {
	case Array, Slice, String, Map:
		return true
	case Pointer:
		return t.Elem().Kind() == Array
	case Func:
		return rangeFuncType(t, 2)
	}
	return false
}

// The value operations that a template engine reads.

// CanAddr reports whether Addr gives a pointer to the value.
func (v Value) CanAddr() bool { return v.loc != locNone }

// Addr gives a pointer to the value. The value must hold a location.
func (v Value) Addr() Value {
	if v.loc == locNone {
		panic("reflect.Value.Addr of unaddressable value")
	}
	return Value{value: dart.ValueAddr(v.read(), v.owner, v.index, v.loc), ro: v.ro, ok: true}
}

// MethodByName gives the method of the name, bound to the value. A name that
// no method holds gives the zero Value.
func (v Value) MethodByName(name string) Value {
	m, ok := v.Type().MethodByName(name)
	if !ok {
		return Value{}
	}
	return v.Method(m.Index)
}

// Equal reports whether the 2 values hold the same type and the same value.
func (v Value) Equal(u Value) bool {
	if v.Kind() == Interface {
		v = v.Elem()
	}
	if u.Kind() == Interface {
		u = u.Elem()
	}
	if !v.IsValid() || !u.IsValid() {
		return v.IsValid() == u.IsValid()
	}
	if v.Type() != u.Type() {
		return false
	}
	switch v.Kind() {
	case Array:
		if v.Len() == 0 && !v.Type().Elem().Comparable() {
			break
		}
		for i := 0; i < v.Len(); i++ {
			if !v.Index(i).Equal(u.Index(i)) {
				return false
			}
		}
		return true
	case Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).Equal(u.Field(i)) {
				return false
			}
		}
		return true
	case Func, Map, Slice:
	default:
		return dart.ValueEqual(v.read(), u.read())
	}
	panic("reflect.Value.Equal: values of type " + v.Type().String() + " are not comparable")
}

// Convert gives the value converted to the type t.
func (v Value) Convert(t Type) Value {
	if t == nil {
		panic("reflect: nil type passed to Value.Convert")
	}
	got := dart.ValueConvert(v.read(), t.handle())
	if t.Kind() == Interface {
		// The result holds the dynamic value with its own type, and the
		// interface is the static type of the location. Interface of the
		// result thus gives the concrete value, exactly as Go gives it.
		return Value{value: got, typ: t, ok: true}
	}
	return Value{value: got, ok: true}
}

// CanConvert reports whether a conversion of the value to t holds.
func (v Value) CanConvert(t Type) bool {
	if t == nil {
		return false
	}
	if v.Kind() == Slice && t.Kind() == Array && v.Len() < t.Len() {
		return false
	}
	return v.Type().ConvertibleTo(t)
}

// Slice gives the slice v[i:j].
func (v Value) Slice(i, j int) Value {
	v.mustBe("Slice", Array, Slice, String)
	if v.Kind() == Array && !v.CanAddr() {
		panic("reflect.Value.Slice: slice of unaddressable array")
	}
	var max int
	if v.Kind() == String {
		max = v.Len()
	} else {
		max = v.Cap()
	}
	if i < 0 || j < i || j > max {
		if v.Kind() == String {
			panic("reflect.Value.Slice: string slice index out of bounds")
		}
		panic("reflect.Value.Slice: slice index out of bounds")
	}
	return Value{value: dart.ValueSlice(v.read(), i, j, max), ro: v.ro, ok: true}
}

// Slice3 gives the slice v[i:j:k].
func (v Value) Slice3(i, j, k int) Value {
	v.mustBe("Slice3", Array, Slice)
	if v.Kind() == Array && !v.CanAddr() {
		panic("reflect.Value.Slice3: slice of unaddressable array")
	}
	if i < 0 || j < i || k < j || k > v.Cap() {
		panic("reflect.Value.Slice3: slice index out of bounds")
	}
	return Value{value: dart.ValueSlice(v.read(), i, j, k), ro: v.ro, ok: true}
}

// Complex gives the complex value.
func (v Value) Complex() complex128 {
	v.mustBe("Complex", Complex64, Complex128)
	return dart.AsComplex(v.read())
}

// SetComplex writes a complex value through the location.
func (v Value) SetComplex(x complex128) {
	v.mustBeAssignable("reflect.Value.SetComplex")
	v.mustBe("SetComplex", Complex64, Complex128)
	v.write(dart.WithComplex(v.descOf(), x))
}

// Pointer gives a number that names the object of the value.
//
// Dart gives no machine address, thus the number is the identity of the object
// and not an address. 2 reads of the same object give the same number, and the
// number names no memory. Read docs/parity.md.
func (v Value) Pointer() uintptr { return dart.ValuePointer(v.read()) }

// UnsafePointer gives the same number as Pointer.
func (v Value) UnsafePointer() uintptr { return v.Pointer() }

// Value.UnsafeAddr stays out. Go gives the address of the value, and Dart
// gives no address: Pointer reads the identity of an object instead. Addr
// builds a new cell at each call for a field of a scalar type, thus 2 reads of
// 1 field would give 2 numbers. A number that changes is a wrong answer, thus
// the name holds no place here.

// IsZero reports whether the value is the zero value of its type.
func (v Value) IsZero() bool {
	t := v.Type()
	if t == nil {
		return true
	}
	// Go answers each of these kinds with IsNil, and it walks an array and a
	// struct that hold no comparison. A struct with a field of a function type
	// is such a struct, and the equality of the runtime reports false for it.
	switch v.Kind() {
	case Chan, Func, Interface, Map, Pointer, Slice, UnsafePointer:
		return v.IsNil()
	case Array:
		if t.Comparable() {
			break
		}
		for i := 0; i < v.Len(); i++ {
			if !v.Index(i).IsZero() {
				return false
			}
		}
		return true
	case Struct:
		if t.Comparable() {
			break
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsZero() {
				return false
			}
		}
		return true
	}
	return dart.ValueEqual(v.read(), dart.DescZero(t.handle()))
}

// CanInterface reports whether the value has no field access restriction.
// Unexported interface methods retain true and fail in Interface or Call.
func (v Value) CanInterface() bool {
	if !v.IsValid() {
		panic(&ValueError{Method: "reflect.Value.CanInterface", Kind: Invalid})
	}
	return v.ro == 0
}

// FieldByName gives the field of the name. A name that no field holds gives
// the zero Value.
func (v Value) FieldByName(name string) Value {
	f, ok := v.Type().FieldByName(name)
	if !ok {
		return Value{}
	}
	return v.FieldByIndex(f.Index)
}

// FieldByIndex gives the field that each index of the path names.
func (v Value) FieldByIndex(index []int) Value {
	out := v
	for i, x := range index {
		if i > 0 && out.Kind() == Pointer {
			if out.IsNil() {
				panic("reflect: indirection through nil pointer to embedded struct")
			}
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
				return Value{}, errorText("reflect: indirection through nil pointer to embedded struct field " + out.Type().Elem().Name())
			}
			out = out.Elem()
		}
		out = out.Field(x)
	}
	return out, nil
}

// errorText is an error that holds 1 text. The overlay reads no errors
// package, because that package reads no reflect and the surface keeps the 2
// apart.
type errorText string

func (e errorText) Error() string { return string(e) }

// A MapIter walks the pairs of a map. The iterator reads the map itself, thus
// a clear inside the loop ends the loop and a delete drops the entry. A
// snapshot of the keys would walk each entry that the loop body dropped, and a
// NaN key gives no second read.
type MapIter struct {
	m    Value
	it   any
	done bool
}

// MapRange gives an iterator of the pairs of a map.
func (v Value) MapRange() *MapIter {
	if v.Kind() != Map {
		panic(&ValueError{"reflect.Value.MapRange", v.Kind()})
	}
	return &MapIter{m: v}
}

// Next steps to the next pair, and it reports whether a pair stands there.
func (it *MapIter) Next() bool {
	if !it.m.IsValid() {
		panic("reflect: MapIter.Next called without an associated map")
	}
	if it.done {
		panic("reflect: MapIter.Next called on exhausted iterator")
	}
	if it.it == nil {
		it.it = dart.MapIterOf(it.m.read())
	}
	it.done = !dart.MapIterNext(it.it)
	return !it.done
}

// Key gives the key of the pair.
func (it *MapIter) Key() Value {
	if it.it == nil {
		panic("reflect: MapIter.Key called before Next")
	}
	if it.done {
		panic("reflect: MapIter.Key called on exhausted iterator")
	}
	return Value{value: dart.MapIterKey(it.m.read(), it.it), typ: it.m.Type().Key(), ro: it.m.ro, ok: true}
}

// Value gives the value of the pair.
func (it *MapIter) Value() Value {
	if it.it == nil {
		panic("reflect: MapIter.Value called before Next")
	}
	if it.done {
		panic("reflect: MapIter.Value called on exhausted iterator")
	}
	return Value{value: dart.MapIterValue(it.m.read(), it.it), typ: it.m.Type().Elem(), ro: it.m.ro, ok: true}
}

// Reset points the iterator at another map.
func (it *MapIter) Reset(m Value) {
	if m.IsValid() && m.Kind() != Map {
		panic(&ValueError{"reflect.Value.MapIter.Reset", m.Kind()})
	}
	it.m = m
	it.it = nil
	it.done = false
}

// IsExported reports whether the field is exported.
func (f StructField) IsExported() bool { return f.PkgPath == "" }

// Recv reads 1 value of a channel. The flag is false when the channel closed.
func (v Value) Recv() (Value, bool) {
	v.mustBe("Recv", Chan)
	out, ok := dart.ValueRecv(v.read())
	return Value{value: out, ok: true}, ok
}

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
		elem := v.Type().Elem()
		if elem.Kind() != Uint8 {
			panic("reflect.Value.Bytes of non-byte slice")
		}
		if elem.PkgPath() == "" {
			// A defined slice type of bytes, for example `type Raw []byte`,
			// holds its own name in the descriptor. Thus an assertion to
			// []byte fails, and the value takes the conversion first. The
			// conversion keeps the same Dart list, thus a write through the
			// result still reaches the value, exactly as in Go.
			return v.Convert(TypeOf([]byte(nil))).read().([]byte)
		}
		// A defined element type, for example `type raw byte`, gives a slice
		// that no conversion of Go reaches. Go reads the memory of such a
		// slice again with the element type byte, and the Dart form of the 2
		// slices is 1 object. A write of that object into the box of a []byte
		// variable thus gives the alias that Go gives.
		var out []byte
		dart.PointerSet(&out, v.read())
		return out
	case Array:
		if v.Type().Elem().Kind() != Uint8 {
			panic("reflect.Value.Bytes of non-byte array")
		}
		if !v.CanAddr() {
			panic("reflect.Value.Bytes of unaddressable byte array")
		}
		// Go gives a slice that shares the storage of the array, thus a write
		// through the result reaches the array. The Dart form of an array is 1
		// list, and the slice of that list holds the same list.
		return v.Slice(0, v.Len()).Bytes()
	}
	panic(&ValueError{Method: "reflect.Value.Bytes", Kind: v.Kind()})
}

// SetBytes writes a byte slice into the location that v names.
func (v Value) SetBytes(x []byte) {
	v.mustBeAssignable("reflect.Value.SetBytes")
	if v.Kind() != Slice || v.Type().Elem().Kind() != Uint8 {
		panic("reflect.Value.SetBytes of " + v.Type().String())
	}
	v.write(x)
}

// SetLen writes the length of the slice that v names. The capacity holds, and
// the elements between the 2 lengths keep their values.
func (v Value) SetLen(n int) {
	v.mustBeAssignable("reflect.Value.SetLen")
	v.mustBe("SetLen", Slice)
	if n < 0 || n > v.Cap() {
		panic("reflect: SetLen: slice length out of range")
	}
	v.Set(v.Slice3(0, n, v.Cap()))
}

// Grow raises the capacity of the slice that v names, thus n more elements fit
// with no further allocation. The length holds.
func (v Value) Grow(n int) {
	v.mustBeAssignable("reflect.Value.Grow")
	v.mustBe("Grow", Slice)
	if n < 0 {
		panic("reflect.Value.Grow: negative len")
	}
	need := v.Len() + n
	if need < 0 {
		panic("reflect.Value.Grow: slice overflow")
	}
	if need <= v.Cap() {
		return
	}
	v.write(dart.GrowValue(v.read(), n))
}

// The channel operations of a Value. The runtime holds real channels, thus a
// send and a receive give up control exactly as the operators of Go do.

// Send writes 1 value into the channel that v holds. The call blocks the
// goroutine until a receiver takes the value or the buffer holds room.
func (v Value) Send(x Value) {
	if v.Kind() != Chan {
		panic(&ValueError{Method: "reflect.Value.Send", Kind: v.Kind()})
	}
	if v.Type().ChanDir() == RecvDir {
		panic("reflect: send on a receive-only channel")
	}
	dart.ValueSend(v.read(), x.read())
}

// Close closes the channel that v holds.
func (v Value) Close() {
	if v.Kind() != Chan {
		panic(&ValueError{Method: "reflect.Value.Close", Kind: v.Kind()})
	}
	if v.Type().ChanDir() == RecvDir {
		panic("reflect: close of receive-only channel")
	}
	dart.ValueClose(v.read())
}

// TryRecv reads 1 value of the channel that v holds and gives up no control.
//
// A receive that ends gives the value and a true flag. A receive that cannot
// end at this moment gives the zero Value and a false flag. A channel that
// closed gives the zero value of the element type and a false flag.
func (v Value) TryRecv() (Value, bool) {
	if v.Kind() != Chan {
		panic(&ValueError{Method: "reflect.Value.TryRecv", Kind: v.Kind()})
	}
	if v.Type().ChanDir() == SendDir {
		panic("reflect: receive on send-only channel")
	}
	out, ok, valid := dart.ValueTryRecv(v.read())
	if !valid {
		return Value{}, false
	}
	return Value{value: out, typ: v.Type().Elem(), ok: true}, ok
}

// TrySend writes 1 value into the channel that v holds and gives up no control.
// The flag is false when the send cannot end at this moment.
func (v Value) TrySend(x Value) bool {
	if v.Kind() != Chan {
		panic(&ValueError{Method: "reflect.Value.TrySend", Kind: v.Kind()})
	}
	if v.Type().ChanDir() == RecvDir {
		panic("reflect: send on a receive-only channel")
	}
	return dart.ValueTrySend(v.read(), x.read())
}

// A SelectDir names the direction of 1 case of a select statement.
type SelectDir int

// The 3 directions of a case of a select statement.
const (
	_             SelectDir = iota
	SelectSend              // case Chan <- Send
	SelectRecv              // case <-Chan:
	SelectDefault           // default
)

// A SelectCase describes 1 case of a select statement.
//
// The direction names the kind of the case. A case of the kind SelectDefault
// holds no channel and no value. A case whose Chan holds the zero Value never
// runs, exactly as in Go.
type SelectCase struct {
	Dir  SelectDir
	Chan Value
	Send Value
}

// Select runs a select statement of the cases and gives the number of the case
// that it took.
//
// A case of the kind SelectRecv also gives the value and the flag of the
// receive. Each other kind gives the zero Value and a false flag. Go takes a
// ready case at random, and this call takes the first one. A Go program must
// not depend on the choice.
func Select(cases []SelectCase) (chosen int, recv Value, recvOK bool) {
	dirs := make([]int, len(cases))
	chans := make([]any, len(cases))
	sends := make([]any, len(cases))
	seenDefault := false
	for i, c := range cases {
		switch c.Dir {
		case SelectDefault:
			if seenDefault {
				panic("reflect.Select: multiple default cases")
			}
			seenDefault = true
			if c.Chan.IsValid() {
				panic("reflect.Select: default case has Chan value")
			}
			if c.Send.IsValid() {
				panic("reflect.Select: default case has Send value")
			}
			dirs[i] = int(SelectDefault)
		case SelectSend:
			dirs[i] = int(SelectSend)
			if !c.Chan.IsValid() {
				break
			}
			if c.Chan.Kind() != Chan {
				panic("reflect.Select: SendDir case has no channel")
			}
			if c.Chan.Type().ChanDir() == RecvDir {
				panic("reflect.Select: SendDir case of a receive-only channel")
			}
			if !c.Send.IsValid() {
				panic("reflect.Select: SendDir case missing Send value")
			}
			chans[i] = c.Chan.read()
			sends[i] = c.Send.read()
		case SelectRecv:
			dirs[i] = int(SelectRecv)
			if c.Send.IsValid() {
				panic("reflect.Select: RecvDir case has Send value")
			}
			if !c.Chan.IsValid() {
				break
			}
			if c.Chan.Kind() != Chan {
				panic("reflect.Select: RecvDir case has no channel")
			}
			if c.Chan.Type().ChanDir() == SendDir {
				panic("reflect.Select: RecvDir case of a send-only channel")
			}
			chans[i] = c.Chan.read()
		default:
			panic("reflect.Select: invalid Dir")
		}
	}
	at, out, ok := dart.SelectValue(dirs, chans, sends)
	if at < 0 || at >= len(cases) || cases[at].Dir != SelectRecv {
		return at, Value{}, false
	}
	return at, Value{value: out, typ: cases[at].Chan.Type().Elem(), ok: true}, ok
}

// The predicates of a Value, and the small names of the package.

// A ValueError names a call of a method of Value on a Value of the wrong kind.
type ValueError struct {
	Method string
	Kind   Kind
}

// Error gives the text of the error.
func (e *ValueError) Error() string {
	if e.Kind == Invalid {
		return "reflect: call of " + e.Method + " on zero Value"
	}
	return "reflect: call of " + e.Method + " on " + e.Kind.String() + " Value"
}

// CanInt reports whether Int gives the value with no panic.
func (v Value) CanInt() bool {
	switch v.Kind() {
	case Int, Int8, Int16, Int32, Int64:
		return true
	}
	return false
}

// CanUint reports whether Uint gives the value with no panic.
func (v Value) CanUint() bool {
	switch v.Kind() {
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		return true
	}
	return false
}

// CanFloat reports whether Float gives the value with no panic.
func (v Value) CanFloat() bool {
	switch v.Kind() {
	case Float32, Float64:
		return true
	}
	return false
}

// CanComplex reports whether Complex gives the value with no panic.
func (v Value) CanComplex() bool {
	switch v.Kind() {
	case Complex64, Complex128:
		return true
	}
	return false
}

// Comparable reports whether the operator == of Go reads the value with no
// panic.
func (v Value) Comparable() bool {
	switch v.Kind() {
	case Invalid:
		return false
	case Array:
		switch v.Type().Elem().Kind() {
		case Interface, Array, Struct:
			for i := 0; i < v.Type().Len(); i++ {
				if !v.Index(i).Comparable() {
					return false
				}
			}
			return true
		}
		return v.Type().Comparable()
	case Interface:
		return v.IsNil() || v.Elem().Comparable()
	case Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).Comparable() {
				return false
			}
		}
		return true
	}
	return v.Type().Comparable()
}

// Clear drops each entry of a map, or writes the zero value into each element
// of a slice.
func (v Value) Clear() {
	switch v.Kind() {
	case Slice, Map:
		dart.ValueClear(v.read())
	default:
		panic(&ValueError{Method: "reflect.Value.Clear", Kind: v.Kind()})
	}
}

// maxFloat32 holds the largest float32. The overlay reads no math package,
// because that package reads no reflect and the surface keeps the 2 apart.
const maxFloat32 = 3.4028234663852886e+38

// overflowFloat32 reports whether a float64 does not fit a float32.
func overflowFloat32(x float64) bool {
	if x < 0 {
		x = -x
	}
	return maxFloat32 < x
}

// OverflowInt reports whether the number does not fit the type of v.
func (v Value) OverflowInt(x int64) bool {
	switch v.Kind() {
	case Int, Int8, Int16, Int32, Int64:
		bits := uint(v.Type().Bits())
		trunc := (x << (64 - bits)) >> (64 - bits)
		return x != trunc
	}
	panic(&ValueError{Method: "reflect.Value.OverflowInt", Kind: v.Kind()})
}

// OverflowUint reports whether the number does not fit the type of v.
func (v Value) OverflowUint(x uint64) bool {
	switch v.Kind() {
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		bits := uint(v.Type().Bits())
		trunc := (x << (64 - bits)) >> (64 - bits)
		return x != trunc
	}
	panic(&ValueError{Method: "reflect.Value.OverflowUint", Kind: v.Kind()})
}

// OverflowFloat reports whether the float does not fit the type of v.
func (v Value) OverflowFloat(x float64) bool {
	switch v.Kind() {
	case Float32:
		return overflowFloat32(x)
	case Float64:
		return false
	}
	panic(&ValueError{Method: "reflect.Value.OverflowFloat", Kind: v.Kind()})
}

// OverflowComplex reports whether the complex number does not fit the type of
// v.
func (v Value) OverflowComplex(x complex128) bool {
	switch v.Kind() {
	case Complex64:
		return overflowFloat32(real(x)) || overflowFloat32(imag(x))
	case Complex128:
		return false
	}
	panic(&ValueError{Method: "reflect.Value.OverflowComplex", Kind: v.Kind()})
}

// SetCap writes the capacity of the slice that v names. The length holds, and
// the elements keep their values.
func (v Value) SetCap(n int) {
	if v.Kind() != Slice {
		panic(&ValueError{Method: "reflect.Value.SetCap", Kind: v.Kind()})
	}
	if n < v.Len() || n > v.Cap() {
		panic("reflect: SetCap: slice capacity out of range")
	}
	v.Set(v.Slice3(0, v.Len(), n))
}

// SetIterKey writes the key of the pair that the iterator names into the
// location that v names.
func (v Value) SetIterKey(it *MapIter) {
	if it.it == nil {
		panic("reflect: Value.SetIterKey called before Next")
	}
	if it.done {
		panic("reflect: Value.SetIterKey called on exhausted iterator")
	}
	v.mustBeAssignable("reflect.Value.SetIterKey")
	v.Set(it.Key())
}

// SetIterValue writes the value of the pair that the iterator names into the
// location that v names.
func (v Value) SetIterValue(it *MapIter) {
	if it.it == nil {
		panic("reflect: Value.SetIterValue called before Next")
	}
	if it.done {
		panic("reflect: Value.SetIterValue called on exhausted iterator")
	}
	v.mustBeAssignable("reflect.Value.SetIterValue")
	v.Set(it.Value())
}

// CallSlice runs a variadic function and assigns the last argument to the
// variadic parameter.
//
// Go assigns the slice itself, thus the function reads the same backing array
// as the caller.
func (v Value) CallSlice(in []Value) []Value {
	v.mustBe("CallSlice", Func)
	v.mustAllowCall("CallSlice")
	v.mustAllowArguments("CallSlice", in)
	t := v.Type()
	args := make([]any, len(in))
	for i, a := range in {
		args[i] = a.read()
	}
	raw := dart.CallSliceValue(v.read(), args)
	out := make([]Value, len(raw))
	for i, r := range raw {
		out[i] = Value{value: r, typ: t.Out(i), ok: true}
	}
	return out
}

// Fields gives a sequence of the field description and the value of each field
// of a struct.
func (v Value) Fields() iter.Seq2[StructField, Value] {
	t := v.Type()
	if t.Kind() != Struct {
		panic("reflect: Fields of non-struct type " + t.String())
	}
	return func(yield func(StructField, Value) bool) {
		for i := 0; i < v.NumField(); i++ {
			if !yield(t.Field(i), v.Field(i)) {
				return
			}
		}
	}
}

// Methods gives a sequence of the method description and the bound method value
// of each exported method of the type of v.
func (v Value) Methods() iter.Seq2[Method, Value] {
	t := v.Type()
	n := v.NumMethod()
	return func(yield func(Method, Value) bool) {
		for i := 0; i < n; i++ {
			if !yield(t.Method(i), v.Method(i)) {
				return
			}
		}
	}
}

// FieldByNameFunc gives the field of the struct whose name the function
// accepts.
//
// A name that stands at 2 fields of 1 depth gives the zero Value, exactly as in
// Go.
func (v Value) FieldByNameFunc(match func(string) bool) Value {
	f, ok := v.Type().FieldByNameFunc(match)
	if !ok {
		return Value{}
	}
	return v.FieldByIndex(f.Index)
}

// Swapper gives a function that exchanges 2 elements of the slice.
func Swapper(slice any) func(i, j int) {
	v := ValueOf(slice)
	if v.Kind() != Slice {
		panic(&ValueError{Method: "Swapper", Kind: v.Kind()})
	}
	switch v.Len() {
	case 0:
		return func(i, j int) { panic("reflect: slice index out of range") }
	case 1:
		return func(i, j int) {
			if i != 0 || j != 0 {
				panic("reflect: slice index out of range")
			}
		}
	}
	if v.Type().Elem().Size() == 0 {
		length := v.Len()
		return func(i, j int) {
			if uint(i) >= uint(length) || uint(j) >= uint(length) {
				panic("reflect: slice index out of range")
			}
		}
	}
	return func(i, j int) { dart.SliceSwap(slice, i, j) }
}

// VisibleFields gives each field that a selector of the struct type reaches.
//
// The list holds each field of the type itself and each promoted field of an
// embedded struct. A name that stands at 2 fields of 1 depth reaches no
// selector, thus the list drops both.
func VisibleFields(t Type) []StructField {
	if t == nil {
		panic("reflect: VisibleFields(nil)")
	}
	if t.Kind() != Struct {
		panic("reflect.VisibleFields of non-struct type")
	}
	w := &fieldWalker{byName: map[string]int{}, visiting: map[Type]bool{}}
	w.walk(t)
	out := make([]StructField, 0, len(w.fields))
	for _, f := range w.fields {
		if f.Name != "" {
			out = append(out, f)
		}
	}
	return out
}

// A fieldWalker collects the visible fields of a struct type. A field whose
// name a shallower field already holds keeps no name, thus VisibleFields drops
// it.
type fieldWalker struct {
	byName   map[string]int
	visiting map[Type]bool
	fields   []StructField
	index    []int
}

func (w *fieldWalker) walk(t Type) {
	if w.visiting[t] {
		return
	}
	w.visiting[t] = true
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		w.index = append(w.index, i)
		add := true
		if old, ok := w.byName[f.Name]; ok {
			switch {
			case len(w.index) == len(w.fields[old].Index):
				w.fields[old].Name = ""
				add = false
			case len(w.index) < len(w.fields[old].Index):
				w.fields[old].Name = ""
			default:
				add = false
			}
		}
		if add {
			path := make([]int, len(w.index))
			copy(path, w.index)
			f.Index = path
			w.byName[f.Name] = len(w.fields)
			w.fields = append(w.fields, f)
		}
		if f.Anonymous {
			inner := f.Type
			if inner.Kind() == Pointer {
				inner = inner.Elem()
			}
			if inner.Kind() == Struct {
				w.walk(inner)
			}
		}
		w.index = w.index[:len(w.index)-1]
	}
	delete(w.visiting, t)
}
