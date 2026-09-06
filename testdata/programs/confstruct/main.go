// The program proves the struct rules of Go: a struct is a value, 2 values of
// 1 comparable struct type compare field by field, an embedded field promotes
// its own fields and its own methods, an unnamed struct type holds a value,
// and a field of a zero size holds no byte.
//
// The size, the alignment, and the offset of each field come from reflect.
// The subset holds no unsafe package.
package main

import (
	"fmt"
	"reflect"
)

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

type point struct {
	X, Y int
}

type named struct {
	Name string
	At   point
	Tags [2]string
}

func compares() {
	a := point{1, 2}
	b := point{1, 2}
	c := point{2, 1}
	row("a == b", a == b)
	row("a == c", a == c)
	row("a != c", a != c)
	row("zero == zero", point{} == point{})
	row("a == zero", a == point{})

	// A struct is a value, thus an assignment copies each field.
	d := a
	d.X = 9
	row("a after copy", a)
	row("d", d)

	// A nested struct and an array field also compare.
	n1 := named{"n", point{1, 2}, [2]string{"a", "b"}}
	n2 := named{"n", point{1, 2}, [2]string{"a", "b"}}
	n3 := named{"n", point{1, 2}, [2]string{"a", "c"}}
	row("n1 == n2", n1 == n2)
	row("n1 == n3", n1 == n3)
	n4 := n1
	n4.At.X = 8
	row("n1 after copy", n1.At)
	row("n4", n4.At)

	// A float field of NaN never compares equal.
	type box struct{ V float64 }
	nan := nanOf()
	row("NaN box == itself", box{nan} == box{nan})
	row("1 box == 1 box", box{1} == box{1})

	// A struct in a map and in an interface keeps the same rule.
	m := map[named]int{n1: 1}
	row("map read", m[n2])
	var i any = a
	var j any = b
	row("iface compare", i == j)
	var k any = c
	row("iface differ", i == k)

	// An array of a struct type.
	arr1 := [2]point{{1, 2}, {3, 4}}
	arr2 := [2]point{{1, 2}, {3, 4}}
	row("array compare", arr1 == arr2)

	// A pointer compares by identity.
	p1 := &a
	p2 := &a
	p3 := &b
	row("p1 == p2", p1 == p2)
	row("p1 == p3", p1 == p3)
	row("*p1 == *p3", *p1 == *p3)
}

func nanOf() float64 {
	var zero float64
	return zero / zero
}

// The embedded types prove the promotion of a field and of a method.
type base struct {
	ID   int
	Name string
}

func (b base) Describe() string { return fmt.Sprintf("base %d %s", b.ID, b.Name) }

func (b base) Kind() string { return "base" }

func (b *base) Rename(n string) { b.Name = n }

type middle struct {
	base
	Level int
}

func (m middle) Kind() string { return "middle" }

type outer struct {
	middle
	Name string
	Note string
}

func embedded() {
	o := outer{}
	o.ID = 7
	o.middle.base.Name = "inner"
	o.Name = "outer"
	o.Level = 3
	o.Note = "n"
	row("o.ID", o.ID)
	row("o.Name", o.Name)
	row("o.middle.Name", o.middle.Name)
	row("o.base.Name", o.base.Name)
	row("o.Level", o.Level)
	row("Describe", o.Describe())
	row("Kind", o.Kind())
	row("middle Kind", o.middle.Kind())
	row("base Kind", o.base.Kind())
	o.Rename("changed")
	row("after Rename", o.base.Name)
	row("outer Name", o.Name)
	row("whole", o)
	row("%+v", fmt.Sprintf("%+v", o))

	// A method of an embedded field also reaches an interface.
	var d interface{ Describe() string } = o
	row("through iface", d.Describe())
	var k interface{ Kind() string } = o
	row("Kind through iface", k.Kind())
	var k2 interface{ Kind() string } = o.middle
	row("middle through iface", k2.Kind())

	// An embedded pointer promotes the same names.
	type holder struct {
		*base
		Extra int
	}
	h := holder{base: &base{ID: 1, Name: "p"}, Extra: 2}
	row("holder ID", h.ID)
	row("holder Describe", h.Describe())
	h.Rename("via pointer")
	row("holder after Rename", h.base.Name)

	// A copy of the outer value copies the embedded value.
	o2 := o
	o2.base.Name = "copy"
	row("o after copy", o.base.Name)
	row("o2", o2.base.Name)
	row("o == o2", o == o2)
}

func anonymous() {
	v := struct {
		A int
		B string
	}{1, "x"}
	row("value", v)
	row("%+v", fmt.Sprintf("%+v", v))
	row("A", v.A)
	w := struct {
		A int
		B string
	}{1, "x"}
	row("equal", v == w)

	// An unnamed struct type also stands in a slice, a map, and an interface.
	list := []struct {
		K string
		N int
	}{{"a", 1}, {"b", 2}}
	row("list", list)
	m := map[struct{ K string }]int{{"a"}: 1}
	row("map key", m[struct{ K string }{"a"}])
	var i any = v
	row("in iface", i)
	x, ok := i.(struct {
		A int
		B string
	})
	row("assertion", fmt.Sprint(x, " ", ok))

	// A nested unnamed struct type.
	n := struct {
		Inner struct {
			V int
		}
		N int
	}{}
	n.Inner.V = 5
	n.N = 6
	row("nested", n)

	// An embedded field inside an unnamed struct type.
	e := struct {
		base
		Extra int
	}{base{1, "e"}, 2}
	row("embedded in anon", e.Describe())
	row("anon whole", e)
}

// The zero size types prove that a field of a zero size holds no byte.
type empty struct{}

type withEmpty struct {
	A int32
	E empty
	B int32
}

type onlyEmpty struct {
	A empty
	B empty
}

type endEmpty struct {
	A int32
	E empty
}

func zeroSize() {
	var e empty
	var f empty
	row("empty == empty", e == f)
	row("empty value", e)
	row("size of empty", reflect.TypeOf(e).Size())
	row("size of withEmpty", reflect.TypeOf(withEmpty{}).Size())
	row("size of onlyEmpty", reflect.TypeOf(onlyEmpty{}).Size())
	row("size of endEmpty", reflect.TypeOf(endEmpty{}).Size())
	row("array of empty", reflect.TypeOf([100]empty{}).Size())
	row("withEmpty value", withEmpty{1, empty{}, 2})
	s := []empty{{}, {}, {}}
	row("slice of empty", fmt.Sprint(len(s), " ", s))
	m := map[empty]int{{}: 1}
	row("empty as key", m[empty{}])
	row("empty in iface", any(e) == any(f))
}

// sizes proves the size, the alignment, and the offset of each field.
type mixed struct {
	A bool
	B int64
	C bool
	D int32
	E bool
}

type packed struct {
	A bool
	B bool
	C int16
	D int32
	E int64
}

type strings2 struct {
	A string
	B []int
	C map[string]int
	D *int
	E [3]byte
}

func describe(name string, v any) {
	t := reflect.TypeOf(v)
	fmt.Printf("%-12s size=%-4d align=%-3d fields=%d\n", name, t.Size(), t.Align(), t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Printf("  %-4s %-16s offset=%-4d size=%-4d align=%d\n",
			f.Name, f.Type.String(), f.Offset, f.Type.Size(), f.Type.Align())
	}
}

func layout() {
	describe("point", point{})
	describe("mixed", mixed{})
	describe("packed", packed{})
	describe("strings2", strings2{})
	describe("withEmpty", withEmpty{})
	describe("named", named{})
	for _, v := range []any{int8(0), int16(0), int32(0), int64(0), 0, "",
		float32(0), float64(0), true, [4]byte{}, []byte(nil)} {
		t := reflect.TypeOf(v)
		fmt.Printf("%-12s size=%-4d align=%d\n", t.String(), t.Size(), t.Align())
	}
}

func literals() {
	row("keyed", point{X: 1, Y: 2})
	row("keyed partial", point{Y: 2})
	row("positional", point{1, 2})
	row("nested", named{Name: "a", At: point{X: 1}})
	row("pointer literal", *(&point{1, 2}))
	list := []point{{1, 2}, {3, 4}}
	row("slice literal", list)
	m := map[string]point{"a": {1, 2}}
	row("map literal", m["a"])
	arr := [...]point{{1, 2}, {3, 4}, {5, 6}}
	row("array literal", fmt.Sprint(len(arr), " ", arr))
	pl := []*point{{1, 2}}
	row("pointer in slice", *pl[0])
	nested := [][]point{{{1, 2}}}
	row("nested slice", nested)
}

func main() {
	fmt.Println("== the comparison ==")
	compares()
	fmt.Println("== the embedded field ==")
	embedded()
	fmt.Println("== the unnamed type ==")
	anonymous()
	fmt.Println("== the zero size ==")
	zeroSize()
	fmt.Println("== the layout ==")
	layout()
	fmt.Println("== the literal ==")
	literals()
}
