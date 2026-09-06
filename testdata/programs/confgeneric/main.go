// The program proves the generic rules of Go: a type parameter, a constraint
// that names a method, a constraint that names a union of types, the
// inference of a type argument, a generic type with a method, a generic
// function that takes an instance of a generic type, and the 2 constraints
// any and comparable.
package main

import (
	"cmp"
	"fmt"
	"sort"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

// The union constraints name a set of types.
//
// A constraint that mixes a narrow integer type with a float type stands
// outside the subset today. Stage 2 rejects each arithmetic operation on such
// a parameter, because the dictionary holds a mask for a set of integer types
// only. conf-blocked/confmixedparam holds the witness, and
// REPORT-conform.md, item 10, holds the cause. Thus the program splits the
// set into an integer set, a float set, and a mixed set with no narrow term.
type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type floating interface {
	~float32 | ~float64
}

type real interface {
	~int | ~int64 | ~float64
}

type integer interface {
	~int | ~int64
}

type ordered interface {
	~int | ~int64 | ~float64 | ~string
}

func sum[T number](list []T) T {
	var total T
	for _, v := range list {
		total += v
	}
	return total
}

func sumF[T floating](list []T) T {
	var total T
	for _, v := range list {
		total += v
	}
	return total
}

// sumMixed reads a set that holds an integer type and a float type. No term
// of the set is narrow, thus each operation holds 1 Dart form.
func sumMixed[T real](list []T) T {
	var total T
	for _, v := range list {
		total += v
	}
	return total
}

func maxOf[T ordered](list []T) T {
	best := list[0]
	for _, v := range list[1:] {
		if v > best {
			best = v
		}
	}
	return best
}

func mapTo[A, B any](list []A, f func(A) B) []B {
	out := make([]B, 0, len(list))
	for _, v := range list {
		out = append(out, f(v))
	}
	return out
}

func filter[T any](list []T, keep func(T) bool) []T {
	out := []T{}
	for _, v := range list {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func reduce[T, A any](list []T, start A, f func(A, T) A) A {
	acc := start
	for _, v := range list {
		acc = f(acc, v)
	}
	return acc
}

type myInt int

type myText string

func basics() {
	row("sum of ints", sum([]int{1, 2, 3}))
	row("sum of floats", sumF([]float64{1.5, 2.5}))
	row("sum of float32", sumF([]float32{1.5, 2.25}))
	row("sum of int8", sum([]int8{100, 100}))
	row("sum of uint8", sum([]uint8{200, 100}))
	row("sum of myInt", sum([]myInt{1, 2, 3}))
	row("sum with an argument", sum[int64]([]int64{1, 2}))
	row("max of ints", maxOf([]int{3, 9, 1}))
	row("max of texts", maxOf([]string{"b", "z", "a"}))
	row("max of myInt", maxOf([]myInt{4, 2}))
	row("mapTo", mapTo([]int{1, 2, 3}, func(n int) string { return fmt.Sprint(n * 2) }))
	row("mapTo to a struct", mapTo([]int{1}, func(n int) pair[int, string] {
		return pair[int, string]{n, "x"}
	}))
	row("filter", filter([]int{1, 2, 3, 4}, func(n int) bool { return n%2 == 0 }))
	row("reduce", reduce([]string{"a", "b"}, "", func(a string, s string) string {
		return a + s + "-"
	}))
	row("reduce to an int", reduce([]string{"aa", "b"}, 0, func(a int, s string) int {
		return a + len(s)
	}))
	// The empty case of each one.
	row("sum of nothing", sum([]int{}))
	row("mapTo of nothing", mapTo([]int{}, func(n int) int { return n }))
	row("filter of nothing", filter([]int{}, func(n int) bool { return true }))
	// A generic function as a value.
	f := sum[int]
	row("a generic value", f([]int{4, 5}))
	g := mapTo[int, int]
	row("a value of 2 parameters", g([]int{1, 2}, func(n int) int { return n * 3 }))
	var h func([]float64) float64 = sumF[float64]
	row("through a variable", h([]float64{0.5, 0.25}))
	row("a mixed set of ints", sumMixed([]int{1, 2}))
	row("a mixed set of floats", sumMixed([]float64{0.5, 0.25}))
	row("a mixed set of int64", sumMixed([]int64{5, 6}))
}

// The constraint with a method.
type stringer interface {
	String() string
}

type named interface {
	comparable
	Name() string
}

func join[T stringer](list []T, sep string) string {
	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = v.String()
	}
	return strings.Join(parts, sep)
}

func firstNamed[T named](list []T, want T) string {
	for _, v := range list {
		if v == want {
			return v.Name()
		}
	}
	return "none"
}

type point struct{ X, Y int }

func (p point) String() string { return fmt.Sprintf("(%d,%d)", p.X, p.Y) }

func (p point) Name() string { return "point" }

type tag string

func (t tag) String() string { return string(t) }

func (t tag) Name() string { return "tag " + string(t) }

func withMethod() {
	row("join of points", join([]point{{1, 2}, {3, 4}}, " "))
	row("join of tags", join([]tag{"a", "b"}, ","))
	row("firstNamed", firstNamed([]point{{1, 2}, {3, 4}}, point{3, 4}))
	row("firstNamed no match", firstNamed([]point{{1, 2}}, point{9, 9}))
	row("firstNamed of tags", firstNamed([]tag{"x", "y"}, tag("y")))
	// A pointer type that holds the method.
	row("join of pointers", join([]*counter{{1}, {2}}, "|"))
}

type counter struct{ N int }

func (c *counter) String() string { return fmt.Sprint("c", c.N) }

// The generic types prove a type with a method and a type with 2 parameters.
type pair[A, B any] struct {
	First  A
	Second B
}

func (p pair[A, B]) Swap() pair[B, A] { return pair[B, A]{p.Second, p.First} }

func (p pair[A, B]) String() string { return fmt.Sprintf("<%v %v>", p.First, p.Second) }

func (p *pair[A, B]) SetFirst(v A) { p.First = v }

type stack[T any] struct {
	items []T
}

func (s *stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

func (s stack[T]) Len() int { return len(s.items) }

type set[T comparable] struct {
	m map[T]struct{}
}

func newSet[T comparable](items ...T) *set[T] {
	s := &set[T]{m: map[T]struct{}{}}
	for _, v := range items {
		s.m[v] = struct{}{}
	}
	return s
}

func (s *set[T]) Has(v T) bool { _, ok := s.m[v]; return ok }

func (s *set[T]) Add(v T) { s.m[v] = struct{}{} }

func (s *set[T]) Len() int { return len(s.m) }

func genericTypes() {
	p := pair[int, string]{1, "a"}
	row("pair", p)
	row("pair String", p.String())
	row("pair Swap", p.Swap())
	p.SetFirst(9)
	row("pair after SetFirst", p)
	q := pair[string, pair[int, int]]{"k", pair[int, int]{1, 2}}
	row("a nested pair", q)
	row("the inner one", q.Second.First)

	var s stack[string]
	s.Push("a")
	s.Push("b")
	row("stack Len", s.Len())
	v, ok := s.Pop()
	row("stack Pop", fmt.Sprint(v, " ", ok))
	row("stack Len again", s.Len())
	var empty stack[int]
	n, ok := empty.Pop()
	row("stack Pop of an empty one", fmt.Sprint(n, " ", ok))

	set1 := newSet(1, 2, 3, 2)
	row("set Len", set1.Len())
	row("set Has", set1.Has(2))
	row("set Has no", set1.Has(9))
	set1.Add(9)
	row("set after Add", set1.Len())
	set2 := newSet("a", "b")
	row("a set of texts", set2.Has("a"))
	set3 := newSet(point{1, 2})
	row("a set of structs", set3.Has(point{1, 2}))

	// A generic type inside a slice and a map.
	list := []pair[int, int]{{1, 2}, {3, 4}}
	row("a slice of pairs", list)
	table := map[string]pair[int, int]{"a": {5, 6}}
	row("a map of pairs", table["a"])
	// A generic type inside an interface.
	var st fmt.Stringer = p
	row("a pair in an iface", st.String())
	back, ok := st.(pair[int, string])
	row("an assertion", fmt.Sprint(back, " ", ok))
	_, ok = st.(pair[string, int])
	row("a wrong assertion", ok)
	row("the type", fmt.Sprintf("%T", p))
	row("the type of a stack", fmt.Sprintf("%T", s))
}

// takesGeneric proves a generic function that takes an instance of a generic
// type.
func firstOf[A, B any](list []pair[A, B]) (A, B) {
	return list[0].First, list[0].Second
}

func pushAll[T any](s *stack[T], items ...T) int {
	for _, v := range items {
		s.Push(v)
	}
	return s.Len()
}

func merge[T comparable](a, b *set[T]) *set[T] {
	out := newSet[T]()
	for k := range a.m {
		out.Add(k)
	}
	for k := range b.m {
		out.Add(k)
	}
	return out
}

func takesGeneric() {
	a, b := firstOf([]pair[int, string]{{1, "x"}, {2, "y"}})
	row("firstOf", fmt.Sprint(a, " ", b))
	var s stack[int]
	row("pushAll", pushAll(&s, 1, 2, 3))
	row("merge", merge(newSet(1, 2), newSet(2, 3)).Len())
	// A generic function that a generic function calls.
	row("nested call", doubleSum([]int{1, 2, 3}))
	row("nested with floats", doubleSumF([]float64{1.5}))
}

func doubleSum[T number](list []T) T { return sum(list) + sum(list) }

func doubleSumF[T floating](list []T) T { return sumF(list) + sumF(list) }

// anyAndComparable proves the 2 constraints of the language.
func idOf[T any](v T) T { return v }

func equalOf[T comparable](a, b T) bool { return a == b }

func countOf[T comparable](list []T) map[T]int {
	out := map[T]int{}
	for _, v := range list {
		out[v]++
	}
	return out
}

func zeroOf[T any]() T {
	var z T
	return z
}

func anyAndComparable() {
	row("idOf int", idOf(5))
	row("idOf text", idOf("a"))
	row("idOf a slice", idOf([]int{1}))
	row("idOf a struct", idOf(point{1, 2}))
	row("idOf nil", idOf[any](nil))
	row("equalOf ints", equalOf(1, 1))
	row("equalOf texts", equalOf("a", "b"))
	row("equalOf structs", equalOf(point{1, 2}, point{1, 2}))
	row("equalOf arrays", equalOf([2]int{1, 2}, [2]int{1, 3}))
	row("equalOf any", equalOf[any](1, 1))
	row("equalOf any of 2 types", equalOf[any](1, "1"))
	counts := countOf([]string{"a", "b", "a"})
	keys := []string{}
	for k := range counts {
		keys = append(keys, fmt.Sprint(k, "=", counts[k]))
	}
	sort.Strings(keys)
	row("countOf", keys)
	row("zeroOf int", zeroOf[int]())
	row("zeroOf text", fmt.Sprintf("%q", zeroOf[string]()))
	row("zeroOf a struct", zeroOf[point]())
	row("zeroOf a pointer", zeroOf[*point]() == nil)
	row("zeroOf a slice", zeroOf[[]int]() == nil)
	row("zeroOf a map", zeroOf[map[string]int]() == nil)
	row("zeroOf an iface", zeroOf[fmt.Stringer]() == nil)
	row("zeroOf a func", zeroOf[func()]() == nil)
	row("zeroOf an array", zeroOf[[3]int]())
	row("zeroOf a chan", zeroOf[chan int]() == nil)
	// cmp of the standard library reads a constraint of an order.
	row("cmp.Compare", cmp.Compare(1, 2))
	row("cmp.Less", cmp.Less("b", "a"))
	row("cmp.Or", cmp.Or(0, 0, 3, 4))
	row("cmp.Or of texts", cmp.Or("", "x"))
}

// inference proves the inference of a type argument.
func inference() {
	// The argument gives the type.
	row("from an argument", sum([]int32{1, 2}))
	// The result of another call gives the type.
	row("from a call", sumF(mapTo([]int{1, 2}, func(n int) float64 { return float64(n) / 2 })))
	// A function argument gives the type of the result.
	row("from a function", mapTo([]int{1}, func(n int) bool { return n > 0 }))
	// A partial list of arguments.
	row("partial", mapTo[int]([]int{1, 2}, func(n int) string { return "n" }))
	// A constraint gives the type of the result of a generic type.
	var s stack[point]
	pushAll(&s, point{1, 2})
	row("through a pointer", s.Len())
	// An untyped constant takes its default type.
	row("an untyped constant", sumF([]float64{1, 2}))
	row("a variable of a named type", sum([]myInt{5}))
	// The inference of 2 parameters of 1 argument list.
	row("2 parameters", reduce([]int{1, 2, 3}, "", func(a string, n int) string {
		return a + fmt.Sprint(n)
	}))
}

func main() {
	fmt.Println("== the basics ==")
	basics()
	fmt.Println("== a constraint with a method ==")
	withMethod()
	fmt.Println("== a generic type ==")
	genericTypes()
	fmt.Println("== a generic argument ==")
	takesGeneric()
	fmt.Println("== any and comparable ==")
	anyAndComparable()
	fmt.Println("== the inference ==")
	inference()
}
