package main

import (
	"fmt"
	"reflect"

	"example.com/reflectgenericstruct/foreign"
)

type Element struct{ N int }

func (e Element) Number() int { return e.N }

var early = makeValue(reflect.TypeFor[int](), 19).Addr().Interface()

func makeValue(t reflect.Type, value any) reflect.Value {
	shape := reflect.StructOf([]reflect.StructField{{Name: "X", Type: t}})
	v := reflect.New(shape).Elem()
	v.Field(0).Set(reflect.ValueOf(value))
	return v
}
func create[T comparable](value T) {
	v := makeValue(reflect.TypeFor[T](), value)
	x, ok := v.Interface().(struct{ X T })
	fmt.Println(ok, x.X)
	pointer := v.Addr().Interface().(*struct{ X T })
	field := v.Field(0).Addr().Interface().(*T)
	fmt.Println(pointer.X == *field, &pointer.X == field)
	list := reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
	list.Index(0).Set(v)
	items := list.Interface().([]struct{ X T })
	fmt.Println(items[0] == x)
	table := reflect.MakeMap(reflect.MapOf(v.Type(), reflect.TypeFor[int]()))
	table.SetMapIndex(v, reflect.ValueOf(23))
	indexed := table.Interface().(map[struct{ X T }]int)
	fmt.Println(indexed[x])
	copied := v.Interface().(struct{ X T })
	directCopy := *pointer
	v.SetZero()
	fmt.Println(copied == x, copied.X == value, directCopy.X == value, pointer.X == *field)
}
func readEarly[T any](value any) {
	p := value.(*struct{ X T })
	fmt.Println("early", p.X)
}
func tagged[T any](value T) {
	t := reflect.StructOf([]reflect.StructField{{Name: "X", Type: reflect.TypeFor[T](), Tag: `json:"value"`}})
	v := reflect.New(t).Elem()
	v.Field(0).Set(reflect.ValueOf(value))
	x, ok := v.Interface().(struct {
		X T `json:"value"`
	})
	fmt.Println("tagged", ok, x.X)
}
func embedded[T any](value T) {
	t := reflect.StructOf([]reflect.StructField{
		{Name: "Element", Type: reflect.TypeFor[Element](), Anonymous: true},
		{Name: "X", Type: reflect.TypeFor[T]()},
	})
	v := reflect.New(t).Elem()
	v.Field(0).Set(reflect.ValueOf(Element{29}))
	v.Field(1).Set(reflect.ValueOf(value))
	x, ok := v.Interface().(struct {
		Element
		X T
	})
	fmt.Println("embedded", ok, x.Number(), x.X)
	fmt.Println(v.MethodByName("Number").Call(nil)[0].Int())
	fmt.Println(v.Interface().(interface{ Number() int }).Number())
}
func private[T any]() {
	want := reflect.TypeFor[struct{ x T }]()
	field := want.Field(0)
	got := reflect.StructOf([]reflect.StructField{field})
	value, ok := reflect.Zero(got).Interface().(struct{ x T })
	fmt.Println("private", want == got, ok, value.x)
}
func readForeign[T any](input any) {
	value, ok := input.(struct{ X T })
	fmt.Println("foreign init", ok, value.X)
}
func (e *Element) Add(n int) { e.N += n }
func rejectedPointerEmbedding[T any]() {
	defer func() { fmt.Println("pointer embedded result", recover()) }()
	t := reflect.StructOf([]reflect.StructField{
		{Name: "Element", Type: reflect.TypeFor[*Element](), Anonymous: true},
		{Name: "X", Type: reflect.TypeFor[T]()},
	})
	fmt.Println(t)
}
func reference[T any](value T) T {
	v := makeValue(reflect.TypeFor[T](), value)
	x, ok := v.Interface().(struct{ X T })
	fmt.Println("reference", ok)
	return x.X
}
func embeddedIdentity() {
	embedded := reflect.TypeFor[struct{ Element }]()
	explicit := reflect.TypeFor[struct{ Element Element }]()
	a := reflect.StructOf([]reflect.StructField{{Name: "Element", Type: reflect.TypeFor[Element](), Anonymous: true}})
	b := reflect.StructOf([]reflect.StructField{{Name: "Element", Type: reflect.TypeFor[Element]()}})
	fmt.Println("embedded identity", embedded == explicit, a == b, a == embedded, b == explicit)
	fmt.Println("embedded methods", embedded.NumMethod(), explicit.NumMethod(), a.NumMethod(), b.NumMethod())
}
func missing[T any]() {
	defer func() { fmt.Println(recover()) }()
	var value any = 7
	_ = value.(interface {
		M(struct {
			Element
			X T
		})
	})
}
func main() {
	missing[int]()
	embeddedIdentity()
	slice := []int{109}
	reference(slice)[0] = 113
	table := map[string]int{"x": 127}
	reference(table)["x"] = 131
	pointer := &Element{137}
	fmt.Println(slice[0], table["x"], reference(pointer) == pointer)
	fmt.Println(reference(func() int { return 139 })())
	var nilPointer *Element
	fmt.Println(reference(nilPointer) == nil)
	fmt.Println(reference[any](Element{149}).(Element).N)
	readForeign[bool](foreign.Early)
	foreign.Private[int]()
	foreign.Private[string]()
	rejectedPointerEmbedding[int]()
	rejectedPointerEmbedding[string]()
	private[int]()
	private[string]()
	readEarly[int](early)
	create(7)
	create("abc")
	create(Element{31})
	create([2]int{37, 41})
	tagged(43)
	tagged("tag")
	embedded(47)
	embedded("embed")
	fmt.Println("complete")
}
