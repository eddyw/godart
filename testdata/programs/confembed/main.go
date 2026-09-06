// The program proves the embedding rules of Go: a struct that embeds a
// pointer, a struct that embeds an interface, 2 embedded fields that both
// hold 1 name, and a method that the outer type hides.
//
// Go gives a name of a shallow depth priority over a name of a deeper level.
// 2 names of 1 depth hide each other, and only the explicit path reaches
// either 1. Such a name reaches no selector, thus the program spells the
// whole path there.
package main

import (
	"fmt"
	"sort"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

// base holds a field and 3 methods.
type base struct {
	ID   int
	Name string
}

func (b base) Describe() string { return fmt.Sprintf("base(%d,%s)", b.ID, b.Name) }

func (b base) Kind() string { return "base" }

func (b *base) Rename(n string) { b.Name = n }

func (b base) Depth() int { return 1 }

// embedPointer holds a pointer to base.
type embedPointer struct {
	*base
	Extra int
}

func (e embedPointer) Kind() string { return "embedPointer" }

func pointerEmbed() {
	v := embedPointer{base: &base{1, "a"}, Extra: 2}
	row("the promoted field", v.ID)
	row("the promoted method", v.Describe())
	row("the method of the outer", v.Kind())
	row("the method of the inner", v.base.Kind())
	row("Depth", v.Depth())
	v.Rename("b")
	row("after Rename", v.base.Name)
	row("the field of the pointer", v.base.ID)

	// A copy of the outer value shares the base, because the field holds a
	// pointer.
	other := v
	other.Name = "shared"
	row("the base of the first", v.Name)
	row("the base of the second", other.Name)
	row("1 pointer", v.base == other.base)

	// A write of the pointer field itself.
	other.base = &base{9, "own"}
	row("after a new base", fmt.Sprint(v.Name, " ", other.Name))

	// A nil pointer inside the outer type. A read of a promoted field panics,
	// and a method that reads no field runs.
	var empty embedPointer
	row("a nil base", empty.base == nil)
	row("a read of a field", catch(func() any { return empty.ID }))
	row("a method of it", catch(func() any { return empty.Kind() }))
	row("a promoted method of it", catch(func() any { return empty.Describe() }))

	// The outer type reaches an interface through the promoted method.
	var d describer = v
	row("through an interface", d.Describe())
	// The method set of the outer value holds Rename, because the embedded
	// field is a pointer.
	_, ok := any(v).(interface{ Rename(string) })
	row("the value holds Rename", ok)
	// A pointer to the outer type also holds it.
	_, ok = any(&v).(interface{ Rename(string) })
	row("the pointer holds Rename", ok)
	// %v of a value that holds a pointer writes an address in Go, and gart
	// holds no address. Read the section "The runtime" of docs/parity.md.
	// Thus the program prints the value behind the pointer.
	row("the whole form", fmt.Sprintf("%+v", *v.base))
	row("the outer fields", fmt.Sprint(v.Extra, " ", v.base != nil))
}

type describer interface{ Describe() string }

func catch(f func() any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprint("panic: ", r)
		}
	}()
	return fmt.Sprint(f())
}

// The interface embedding proves a struct that holds an interface field with
// no name.
type reader interface {
	Read() string
}

type namer interface {
	Name() string
}

type readNamer interface {
	reader
	namer
}

type textSource struct{ text string }

func (t textSource) Read() string { return t.text }

func (t textSource) Name() string { return "textSource" }

// wrapper embeds the interface, thus it holds each method of it.
type wrapper struct {
	reader
	Count int
}

// upper hides the method Read of the embedded interface.
type upper struct {
	reader
}

func (u upper) Read() string { return strings.ToUpper(u.reader.Read()) }

func ifaceEmbed() {
	w := wrapper{reader: textSource{"hello"}, Count: 1}
	row("the promoted method", w.Read())
	row("the explicit path", w.reader.Read())
	row("the field", w.Count)
	// The outer type holds the interface, thus it fits the interface itself.
	var r reader = w
	row("as a reader", r.Read())

	// A method that the outer type declares hides the method of the embedded
	// interface.
	u := upper{reader: textSource{"hello"}}
	row("the outer method", u.Read())
	row("the inner method", u.reader.Read())
	var ur reader = u
	row("through the interface", ur.Read())

	// A nil interface inside the struct panics at the call.
	var empty wrapper
	row("a nil interface", empty.reader == nil)
	row("a call of it", catch(func() any { return empty.Read() }))

	// An interface value that holds another struct with the same interface.
	nested := wrapper{reader: upper{reader: textSource{"deep"}}}
	row("2 levels", nested.Read())

	// A struct that embeds an interface of 2 embedded interfaces.
	type both struct {
		readNamer
	}
	b := both{readNamer: textSource{"x"}}
	row("Read of both", b.Read())
	row("Name of both", b.Name())
	var rn readNamer = b
	row("as a readNamer", fmt.Sprint(rn.Read(), " ", rn.Name()))

	// An interface field of a named type also holds a nil pointer.
	var np *pointerReader
	holder := wrapper{reader: np}
	row("a typed nil inside", holder.reader == nil)
	row("a call of a nil pointer", holder.Read())
}

type pointerReader struct{ v string }

func (p *pointerReader) Read() string {
	if p == nil {
		return "pointerReader(nil)"
	}
	return p.v
}

// The ambiguous names prove that 2 embedded fields of 1 depth hide each other.
type left struct {
	Shared string
	Only   int
}

func (l left) Which() string { return "left" }

func (l left) OnlyLeft() string { return "onlyLeft" }

type right struct {
	Shared string
	More   int
}

func (r right) Which() string { return "right" }

func (r right) OnlyRight() string { return "onlyRight" }

type middle struct {
	left
	right
}

// deep holds the name Shared at a deeper level, and its own field wins.
type deep struct {
	middle
	Shared string
}

func ambiguous() {
	m := middle{left{"L", 1}, right{"R", 2}}
	// m.Shared and m.Which reach 2 fields of 1 depth, thus Go rejects each
	// one. The explicit path reaches either 1.
	row("the left field", m.left.Shared)
	row("the right field", m.right.Shared)
	row("the left method", m.left.Which())
	row("the right method", m.right.Which())
	// A name that only 1 side holds reaches the selector.
	row("a name of 1 side", m.Only)
	row("a name of the other", m.More)
	row("a method of 1 side", m.OnlyLeft())
	row("a method of the other", m.OnlyRight())
	// The method set of the outer type holds neither Which.
	_, ok := any(m).(interface{ Which() string })
	row("Which is ambiguous", ok)
	_, ok = any(m).(interface{ OnlyLeft() string })
	row("OnlyLeft stands", ok)

	// A field of the outer type wins over the deeper name.
	d := deep{middle: m, Shared: "outer"}
	row("the outer field wins", d.Shared)
	row("the deep left", d.middle.left.Shared)
	row("the deep right", d.middle.right.Shared)
	// Which stays ambiguous at the deeper level.
	row("Which of the deep type", d.middle.left.Which())
	row("the whole value", d)
	row("the fields", fmt.Sprintf("%+v", d))
}

// The shadow types prove a method of the outer type that hides the method of
// the embedded type, and the call of the inner one.
type animal struct{ Name string }

func (a animal) Speak() string { return a.Name + " makes a sound" }

func (a animal) Twice() string { return a.Speak() + " / " + a.Speak() }

type dog struct {
	animal
}

func (d dog) Speak() string { return d.Name + " barks" }

func shadow() {
	d := dog{animal{"rex"}}
	row("the outer method", d.Speak())
	row("the inner method", d.animal.Speak())
	// Go binds the call inside Twice to the method of animal, because Go
	// holds no virtual dispatch of a struct method.
	row("the inner call", d.Twice())
	// An interface of the outer type reaches the outer method.
	var s interface{ Speak() string } = d
	row("through an interface", s.Speak())
	var a interface{ Speak() string } = d.animal
	row("the inner in an interface", a.Speak())
	// A method expression of each level.
	f := dog.Speak
	row("dog.Speak", f(d))
	g := animal.Speak
	row("animal.Speak", g(d.animal))
	// A method value of each level.
	h := d.Speak
	row("a method value", h())
	i := d.animal.Speak
	row("a method value of the inner", i())

	// A list of the method names of each type, through a type assertion.
	names := []string{}
	for _, probe := range []struct {
		name string
		ok   bool
	}{
		{"Speak of dog", hasSpeak(d)},
		{"Twice of dog", hasTwice(d)},
		{"Speak of animal", hasSpeak(d.animal)},
	} {
		names = append(names, fmt.Sprint(probe.name, "=", probe.ok))
	}
	sort.Strings(names)
	row("the method sets", names)
}

func hasSpeak(v any) bool {
	_, ok := v.(interface{ Speak() string })
	return ok
}

func hasTwice(v any) bool {
	_, ok := v.(interface{ Twice() string })
	return ok
}

// The named embed proves an embedded field of a type that is not a struct.
type count int

func (c count) Double() count { return c * 2 }

type record struct {
	count
	Label string
}

func namedEmbed() {
	r := record{count: 3, Label: "x"}
	row("the embedded value", r.count)
	row("the promoted method", r.Double())
	r.count = 5
	row("after a write", r.count)
	row("printed", r)
	row("the fields", fmt.Sprintf("%+v", r))
	// An embedded pointer to a named type.
	type holder struct {
		*count
	}
	n := count(7)
	h := holder{&n}
	row("through a pointer", *h.count)
	row("the method of it", h.Double())
	// An embedded slice type.
	type list []int
	type box struct {
		list
	}
	b := box{list{1, 2, 3}}
	row("an embedded slice", b.list)
	row("the length of it", len(b.list))
	b.list = append(b.list, 4)
	row("after an append", b.list)
}

func main() {
	fmt.Println("== an embedded pointer ==")
	pointerEmbed()
	fmt.Println("== an embedded interface ==")
	ifaceEmbed()
	fmt.Println("== 2 names of 1 depth ==")
	ambiguous()
	fmt.Println("== a method that hides another ==")
	shadow()
	fmt.Println("== an embedded named type ==")
	namedEmbed()
}
