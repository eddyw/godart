// The program proves the interface rules of Go: the 2 forms of a type
// assertion, a type switch, the difference between a nil interface and an
// interface that holds a nil pointer, the method set of a value type against
// the method set of a pointer type, and an embedded interface.
package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
)

func row(label string, value any) {
	fmt.Printf("%-32s %v\n", label, value)
}

type stringer interface{ String() string }

type sizer interface{ Size() int }

type both interface {
	stringer
	sizer
}

type reader interface {
	Read() string
}

// value holds each method on the value type.
type value struct{ N int }

func (v value) String() string { return fmt.Sprintf("value(%d)", v.N) }

func (v value) Size() int { return v.N }

// pointer holds each method on the pointer type.
type pointer struct{ N int }

func (p *pointer) String() string {
	if p == nil {
		return "pointer(nil)"
	}
	return fmt.Sprintf("pointer(%d)", p.N)
}

func (p *pointer) Size() int {
	if p == nil {
		return -1
	}
	return p.N
}

// mixed holds 1 method on each side.
type mixed struct{ N int }

func (m mixed) String() string { return fmt.Sprintf("mixed(%d)", m.N) }

func (m *mixed) Size() int { return m.N * 2 }

func assertions() {
	var s stringer = value{3}
	// The 1 result form panics when the type does not match.
	row("1 result", s.(value).N)
	v, ok := s.(value)
	row("2 result match", fmt.Sprint(v, " ", ok))
	p, ok := s.(*pointer)
	row("2 result no match", fmt.Sprint(p, " ", ok))
	sz, ok := s.(sizer)
	row("to an interface", fmt.Sprint(sz, " ", ok))
	rd, ok := s.(reader)
	row("to a wrong interface", fmt.Sprint(rd, " ", ok))
	bo, ok := s.(both)
	row("to a wider interface", fmt.Sprint(bo, " ", ok))

	// An assertion of any to a concrete type.
	var a any = 42
	n, ok := a.(int)
	row("any to int", fmt.Sprint(n, " ", ok))
	f, ok := a.(float64)
	row("any to float64", fmt.Sprint(f, " ", ok))
	i64, ok := a.(int64)
	row("any to int64", fmt.Sprint(i64, " ", ok))
	var b any = "text"
	t, ok := b.(string)
	row("any to string", fmt.Sprint(t, " ", ok))
	// A named type and its underlying type stay apart.
	type myInt int
	var c any = myInt(5)
	n, ok = c.(int)
	row("myInt to int", fmt.Sprint(n, " ", ok))
	mi, ok := c.(myInt)
	row("myInt to myInt", fmt.Sprint(mi, " ", ok))

	// A nil interface fails each assertion.
	var nilFace stringer
	nv, ok := nilFace.(value)
	row("nil to value", fmt.Sprint(nv, " ", ok))
	ns, ok := nilFace.(stringer)
	row("nil to stringer", fmt.Sprint(ns, " ", ok))
	var nilAny any
	na, ok := nilAny.(any)
	row("nil any to any", fmt.Sprint(na, " ", ok))

	// A recover reads the panic of the 1 result form.
	row("failed assertion", failAssert(s))
	row("failed on nil", failAssert(nilFace))
}

func failAssert(s stringer) (text string) {
	defer func() {
		if r := recover(); r != nil {
			text = fmt.Sprint(r)
		}
	}()
	_ = s.(*pointer)
	return "no panic"
}

func typeSwitch() {
	list := []any{
		nil, 1, int8(1), int64(1), uint(1), 1.5, float32(1.5), "text",
		true, []int{1}, [2]int{1, 2}, map[string]int{}, value{1},
		&pointer{2}, (*pointer)(nil), errors.New("e"), value{9},
	}
	for _, item := range list {
		fmt.Printf("%-20T %s\n", item, describe(item))
	}
}

func describe(item any) string {
	switch v := item.(type) {
	case nil:
		return "nil"
	case int:
		return fmt.Sprint("int ", v)
	case int8, int64:
		return fmt.Sprintf("small or wide %v", v)
	case uint:
		return fmt.Sprint("uint ", v)
	case float64:
		return fmt.Sprint("float64 ", v)
	case float32:
		return fmt.Sprint("float32 ", v)
	case string:
		return fmt.Sprintf("string %q len %d", v, len(v))
	case bool:
		return fmt.Sprint("bool ", v)
	case []int:
		return fmt.Sprint("slice len ", len(v))
	case [2]int:
		return fmt.Sprint("array ", v)
	case map[string]int:
		return fmt.Sprint("map len ", len(v))
	case error:
		return fmt.Sprint("error ", v.Error())
	case stringer:
		return fmt.Sprint("stringer ", v.String())
	default:
		return "other"
	}
}

// order proves that the first case that matches wins.
func order() {
	var s stringer = value{1}
	switch s.(type) {
	case sizer:
		row("first case", "sizer")
	case stringer:
		row("first case", "stringer")
	}
	switch s.(type) {
	case stringer:
		row("other order", "stringer")
	case sizer:
		row("other order", "sizer")
	}
	// A switch with no binding.
	var a any = 5
	switch a.(type) {
	case int:
		row("no binding", "int")
	}
	// A binding that each case reads.
	switch v := a.(type) {
	case int:
		row("binding type", fmt.Sprintf("%T %v", v, v+1))
	}
	// A fall of the switch.
	var b any = 1.5
	switch b.(type) {
	case int:
		row("fall", "int")
	default:
		row("fall", "default")
	}
}

func nilForms() {
	var face stringer
	row("nil interface", face == nil)
	var p *pointer
	face = p
	row("holds a nil pointer", face == nil)
	row("the value inside", face.String())
	row("the size inside", face.(sizer).Size())
	row("printed", fmt.Sprintf("%v", face))
	inner, ok := face.(*pointer)
	row("assertion of it", fmt.Sprint(inner == nil, " ", ok))
	row("compare to typed nil", face == (*pointer)(nil))

	// The classic error trap.
	row("returns a nil error", plainNil() == nil)
	row("returns a typed nil", typedNil() == nil)
	e := typedNil()
	row("the error text", e.Error())

	// 2 interfaces of 1 value compare equal.
	var a stringer = value{1}
	var b stringer = value{1}
	row("2 values", a == b)
	var c stringer = value{2}
	row("2 differ", a == c)
	p1 := &pointer{1}
	var d stringer = p1
	var f stringer = p1
	row("2 pointers", d == f)
	var g stringer = &pointer{1}
	row("2 new pointers", d == g)
	// An interface of a different concrete type never compares equal.
	row("value against pointer", a == d)

	var anyNil any
	var faceNil stringer
	row("any nil == nil", anyNil == nil)
	row("2 nil interfaces", anyNil == any(faceNil))
}

type failure struct{ msg string }

func (f *failure) Error() string {
	if f == nil {
		return "failure(nil)"
	}
	return f.msg
}

func plainNil() error { return nil }

func typedNil() error {
	var f *failure
	return f
}

func methodSets() {
	// A value type holds each method with a value receiver.
	v := value{1}
	var s stringer = v
	var z sizer = v
	row("value to stringer", s.String())
	row("value to sizer", z.Size())
	var b both = v
	row("value to both", b.String())
	// A pointer to a value type also holds each method with a value receiver.
	pv := &v
	s = pv
	row("pointer of value", s.String())
	b = pv
	row("pointer of value both", b.Size())

	// A pointer type holds each method with a pointer receiver, and a value
	// of that type does not. The line `var s stringer = m` for the mixed type
	// with a pointer method thus does not compile.
	m := mixed{2}
	s = m
	row("mixed value stringer", s.String())
	s = &m
	row("mixed pointer stringer", s.String())
	z = &m
	row("mixed pointer sizer", z.Size())
	_, ok := any(m).(sizer)
	row("mixed value is sizer", ok)
	_, ok = any(&m).(sizer)
	row("mixed pointer is sizer", ok)
	_, ok = any(m).(stringer)
	row("mixed value is stringer", ok)

	// A method value keeps the receiver of the moment of the read.
	f := m.String
	m.N = 99
	row("method value of a copy", f())
	g := (&m).String
	row("method value of a pointer", g())
	m.N = 5
	row("after a write", g())
}

// The embedded interfaces prove that a name of an embedded interface belongs
// to the outer one.
type readWriter interface {
	reader
	writer
}

type writer interface {
	Write(text string) int
}

type file struct{ data string }

func (f *file) Read() string { return f.data }

func (f *file) Write(text string) int {
	f.data += text
	return len(text)
}

func embeddedIface() {
	f := &file{"a"}
	var rw readWriter = f
	row("read", rw.Read())
	row("write", rw.Write("bc"))
	row("read again", rw.Read())
	var r reader = rw
	row("narrow", r.Read())
	back, ok := r.(readWriter)
	row("widen", fmt.Sprint(back != nil, " ", ok))
	var w writer = rw
	row("other side", w.Write("d"))
	row("data", f.data)
	// any also embeds nothing, thus each value fits.
	var a any = rw
	_, ok = a.(reader)
	row("any to reader", ok)
	// An interface of the standard library.
	var _ io.Reader
	row("io.Reader is an iface", fmt.Sprintf("%T", (*io.Reader)(nil)))
}

// sortIface proves an interface that a package of the library declares.
type byLen []string

func (b byLen) Len() int           { return len(b) }
func (b byLen) Less(i, j int) bool { return len(b[i]) < len(b[j]) }
func (b byLen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func sortIface() {
	list := byLen{"ccc", "a", "bb"}
	sort.Sort(list)
	row("sorted", list)
	row("IsSorted", sort.IsSorted(list))
	var s sort.Interface = list
	row("through the iface", s.Len())
}

func main() {
	fmt.Println("== the assertion ==")
	assertions()
	fmt.Println("== the type switch ==")
	typeSwitch()
	fmt.Println("== the order of a case ==")
	order()
	fmt.Println("== the nil forms ==")
	nilForms()
	fmt.Println("== the method set ==")
	methodSets()
	fmt.Println("== the embedded interface ==")
	embeddedIface()
	fmt.Println("== an interface of the library ==")
	sortIface()
}
