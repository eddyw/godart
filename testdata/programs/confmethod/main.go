// The program proves the method rules of Go: a value receiver takes a copy
// and a pointer receiver takes the address, a method value binds the receiver
// at the point of the read, a method expression takes the receiver as the
// first argument, an embedded field gives its methods to the outer type, and
// the method set of a pointer holds each method of the value.
package main

import "fmt"

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

type counter struct {
	N    int
	Name string
}

// IncValue takes a copy, thus the write does not reach the caller.
func (c counter) IncValue() int {
	c.N++
	return c.N
}

// IncPointer takes the address, thus the write reaches the caller.
func (c *counter) IncPointer() int {
	c.N++
	return c.N
}

func (c counter) String() string { return fmt.Sprintf("%s=%d", c.Name, c.N) }

func (c *counter) Reset() { c.N = 0 }

func receivers() {
	c := counter{N: 1, Name: "a"}
	row("IncValue", c.IncValue())
	row("c after IncValue", c.N)
	row("IncPointer", c.IncPointer())
	row("c after IncPointer", c.N)

	// Go takes the address of an addressable value at a call of a pointer
	// method, thus c.IncPointer() is (&c).IncPointer().
	p := &c
	row("through a pointer value", p.IncValue())
	row("through a pointer pointer", p.IncPointer())
	row("c now", c.N)

	// A copy of the value holds its own storage.
	d := c
	d.IncPointer()
	row("c after a copy write", c.N)
	row("d", d.N)

	// An element of a slice is addressable.
	list := []counter{{N: 1}, {N: 2}}
	list[0].IncPointer()
	row("slice element", list[0].N)
	// An element of an array is addressable.
	var arr [2]counter
	arr[1].IncPointer()
	row("array element", arr[1].N)
	// A field of a struct is addressable.
	type holder struct{ C counter }
	var h holder
	h.C.IncPointer()
	row("field", h.C.N)
	// A value in a map is not addressable, thus the call must read it first.
	m := map[string]counter{"a": {N: 5}}
	tmp := m["a"]
	tmp.IncPointer()
	m["a"] = tmp
	row("map value", m["a"].N)
	// A pointer in a map reaches the method at once.
	pm := map[string]*counter{"a": {N: 5}}
	pm["a"].IncPointer()
	row("map pointer", pm["a"].N)
}

func methodValues() {
	c := counter{N: 1, Name: "a"}
	// A method value of a value receiver copies the receiver at the read.
	f := c.IncValue
	c.N = 100
	row("value method value", f())
	row("c", c.N)

	// A method value of a pointer receiver keeps the address.
	c = counter{N: 1, Name: "a"}
	g := c.IncPointer
	c.N = 100
	row("pointer method value", g())
	row("c after", c.N)

	// A method value of a pointer holds the same address.
	p := &counter{N: 1, Name: "b"}
	h := p.IncPointer
	row("first call", h())
	row("second call", h())
	row("p", p.N)

	// A method value of a String method reaches fmt.
	s := c.String
	row("String value", s())

	// A method value passes as an argument.
	row("apply", apply(c.IncValue))
	row("apply pointer", apply(c.IncPointer))
	row("c after apply", c.N)

	// A method value of an interface value.
	var st fmt.Stringer = c
	sv := st.String
	c.N = 7
	row("iface method value", sv())
}

func apply(f func() int) int { return f() + f() }

func methodExpressions() {
	c := counter{N: 1, Name: "a"}
	// A method expression of a value type takes the receiver first.
	fv := counter.IncValue
	row("counter.IncValue", fv(c))
	row("c after", c.N)
	// A method expression of a pointer type takes a pointer.
	fp := (*counter).IncPointer
	row("(*counter).IncPointer", fp(&c))
	row("c after", c.N)
	// A method expression of a value type also reaches a pointer type.
	fp2 := (*counter).IncValue
	row("(*counter).IncValue", fp2(&c))
	row("c after", c.N)
	// A method expression of String.
	fs := counter.String
	row("counter.String", fs(c))
	// A method expression as a value of a variable of a function type.
	var op func(counter) int = counter.IncValue
	row("through a variable", op(counter{N: 9}))
	// A method expression of an interface type.
	var si func(fmt.Stringer) string = fmt.Stringer.String
	row("iface expression", si(c))
	// A method expression inside a list.
	ops := []func(*counter) int{(*counter).IncPointer, (*counter).IncValue}
	for i, o := range ops {
		row(fmt.Sprint("op ", i), o(&c))
	}
	row("c at the end", c.N)
}

type animal struct{ Name string }

func (a animal) Speak() string { return a.Name + " speaks" }

func (a animal) Kind() string { return "animal" }

func (a *animal) Rename(n string) { a.Name = n }

type dog struct {
	animal
	Breed string
}

func (d dog) Kind() string { return "dog" }

type puppy struct {
	dog
}

func embeddedMethods() {
	d := dog{animal{"rex"}, "lab"}
	row("Speak", d.Speak())
	row("Kind", d.Kind())
	row("animal Kind", d.animal.Kind())
	d.Rename("max")
	row("after Rename", d.Name)
	row("animal Name", d.animal.Name)

	p := puppy{dog{animal{"tiny"}, "pug"}}
	row("puppy Speak", p.Speak())
	row("puppy Kind", p.Kind())
	row("puppy dog Kind", p.dog.Kind())
	row("puppy animal Kind", p.dog.animal.Kind())
	p.Rename("small")
	row("puppy Name", p.Name)

	// A method expression of a promoted method.
	f := dog.Speak
	row("dog.Speak", f(d))
	g := puppy.Kind
	row("puppy.Kind", g(p))
	h := (*dog).Rename
	h(&d, "via expression")
	row("after that", d.Name)

	// A method value of a promoted method.
	sp := d.Speak
	d.Name = "changed"
	row("method value", sp())

	// An embedded pointer.
	type wrap struct {
		*animal
		Extra int
	}
	w := wrap{&animal{"in"}, 1}
	row("wrap Speak", w.Speak())
	w.Rename("out")
	row("wrap after Rename", w.animal.Name)
}

// The named types prove a method on a type that is not a struct.
type ints []int

func (v ints) Sum() int {
	total := 0
	for _, n := range v {
		total += n
	}
	return total
}

func (v ints) Head() int { return v[0] }

func (v *ints) Push(n int) { *v = append(*v, n) }

type text string

func (t text) Upper() string {
	out := []byte(t)
	for i, b := range out {
		if b >= 'a' && b <= 'z' {
			out[i] = b - 32
		}
	}
	return string(out)
}

type num int

func (n num) Double() num { return n * 2 }

func (n *num) Add(k num) { *n += k }

func namedTypes() {
	v := ints{1, 2, 3}
	row("Sum", v.Sum())
	row("Head", v.Head())
	v.Push(4)
	row("after Push", v)
	f := v.Sum
	v.Push(5)
	row("method value of a slice", f())
	row("v now", v)
	g := ints.Sum
	row("method expression", g(v))

	var t text = "hello"
	row("Upper", t.Upper())
	row("method value", t.Upper())
	h := text.Upper
	row("text.Upper", h("world"))

	var n num = 3
	row("Double", n.Double())
	n.Add(4)
	row("after Add", n)
	k := (*num).Add
	k(&n, 10)
	row("after the expression", n)
	// A method on a named map type.
	type counts map[string]int
	_ = counts{}
}

// nilReceiver proves that a method with a pointer receiver runs with a nil
// receiver, as long as it reads no field.
type node struct {
	Value int
	Next  *node
}

func (n *node) Len() int {
	if n == nil {
		return 0
	}
	return 1 + n.Next.Len()
}

func (n *node) Sum() int {
	if n == nil {
		return 0
	}
	return n.Value + n.Next.Sum()
}

func nilReceiver() {
	var empty *node
	row("nil Len", empty.Len())
	row("nil Sum", empty.Sum())
	chain := &node{1, &node{2, &node{3, nil}}}
	row("Len", chain.Len())
	row("Sum", chain.Sum())
	// A nil pointer inside an interface also reaches the method.
	var i interface{ Len() int } = empty
	row("through iface", i.Len())
	row("iface is nil", i == nil)
}

// methodSet proves which method the compiler gives to a value and to a
// pointer.
type onValue struct{}

func (onValue) A() string { return "A" }

type onPointer struct{}

func (*onPointer) B() string { return "B" }

func methodSet() {
	var v onValue
	var p onPointer
	_, ok := any(v).(interface{ A() string })
	row("value has A", ok)
	_, ok = any(&v).(interface{ A() string })
	row("pointer has A", ok)
	_, ok = any(p).(interface{ B() string })
	row("value has B", ok)
	_, ok = any(&p).(interface{ B() string })
	row("pointer has B", ok)
	c := counter{}
	_, ok = any(c).(interface{ Reset() })
	row("counter value has Reset", ok)
	_, ok = any(&c).(interface{ Reset() })
	row("counter pointer has Reset", ok)
	_, ok = any(c).(fmt.Stringer)
	row("counter value is Stringer", ok)
	_, ok = any(&c).(fmt.Stringer)
	row("counter pointer is Stringer", ok)
	d := dog{}
	_, ok = any(d).(interface{ Speak() string })
	row("dog value has Speak", ok)
	_, ok = any(d).(interface{ Rename(string) })
	row("dog value has Rename", ok)
	_, ok = any(&d).(interface{ Rename(string) })
	row("dog pointer has Rename", ok)
}

func main() {
	fmt.Println("== the receiver ==")
	receivers()
	fmt.Println("== the method value ==")
	methodValues()
	fmt.Println("== the method expression ==")
	methodExpressions()
	fmt.Println("== the embedded method ==")
	embeddedMethods()
	fmt.Println("== a named type ==")
	namedTypes()
	fmt.Println("== a nil receiver ==")
	nilReceiver()
	fmt.Println("== the method set ==")
	methodSet()
}
