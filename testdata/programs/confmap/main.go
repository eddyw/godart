// The program proves the map rules of Go: a key of a struct type, of an array
// type, and of an interface type, the read of a key that the map does not
// hold, the 2 result form, a delete during a range, and clear.
//
// The order of a range over a map is random in Go, thus each output takes a
// sort first.
package main

import (
	"fmt"
	"sort"
)

func row(label string, value any) {
	fmt.Printf("%-26s %v\n", label, value)
}

// dump writes each pair of a map in the order of the key.
func dump(label string, m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	fmt.Printf("%-26s %v\n", label, parts)
}

type point struct {
	X, Y int
}

type label struct {
	Name string
	At   point
}

func structKeys() {
	m := map[point]string{}
	m[point{1, 2}] = "a"
	m[point{3, 4}] = "b"
	m[point{1, 2}] = "c"
	row("len", len(m))
	row("read 1 2", m[point{1, 2}])
	row("read 3 4", m[point{3, 4}])
	row("read 9 9", fmt.Sprintf("%q", m[point{9, 9}]))
	v, ok := m[point{9, 9}]
	row("2 result form", fmt.Sprint(fmt.Sprintf("%q", v), " ", ok))
	v, ok = m[point{3, 4}]
	row("2 result found", fmt.Sprint(v, " ", ok))

	// A struct that holds a struct is also a key.
	n := map[label]int{}
	n[label{"a", point{1, 2}}] = 1
	n[label{"a", point{1, 2}}] += 10
	n[label{"a", point{2, 1}}] = 2
	row("nested len", len(n))
	row("nested read", n[label{"a", point{1, 2}}])

	// A zero value of a struct is a key of its own.
	var zero point
	m[zero] = "zero"
	row("zero key", m[point{0, 0}])
	row("len after zero", len(m))
	keys := make([]string, 0, len(m))
	for k, val := range m {
		keys = append(keys, fmt.Sprintf("%v=%s", k, val))
	}
	sort.Strings(keys)
	row("each pair", keys)
}

func arrayKeys() {
	m := map[[3]int]string{}
	m[[3]int{1, 2, 3}] = "a"
	m[[3]int{1, 2, 4}] = "b"
	m[[3]int{1, 2, 3}] = "c"
	row("len", len(m))
	row("read 1 2 3", m[[3]int{1, 2, 3}])
	row("read absent", fmt.Sprintf("%q", m[[3]int{0, 0, 0}]))
	var key [3]int
	key[0], key[1], key[2] = 1, 2, 4
	row("built key", m[key])

	// An array of a string type.
	n := map[[2]string]int{}
	n[[2]string{"a", "b"}] = 1
	row("string array", n[[2]string{"a", "b"}])
	row("string gap", n[[2]string{"b", "a"}])

	// An array of an array.
	o := map[[2][2]int]bool{}
	o[[2][2]int{{1, 2}, {3, 4}}] = true
	row("array of array", o[[2][2]int{{1, 2}, {3, 4}}])
	row("array gap", o[[2][2]int{{1, 2}, {3, 5}}])
}

type shape interface{ area() int }

type square struct{ side int }

func (s square) area() int { return s.side * s.side }

type circle struct{ r int }

func (c circle) area() int { return 3 * c.r * c.r }

func ifaceKeys() {
	m := map[any]string{}
	m[1] = "int 1"
	m["1"] = "text 1"
	m[int64(1)] = "int64 1"
	m[1.0] = "float 1"
	m[true] = "bool"
	m[point{1, 2}] = "point"
	m[[2]int{1, 2}] = "array"
	row("len", len(m))
	row("int 1", m[1])
	row("text 1", m["1"])
	row("int64 1", m[int64(1)])
	row("float 1", m[1.0])
	row("bool", m[true])
	row("point", m[point{1, 2}])
	row("array", m[[2]int{1, 2}])
	row("absent", fmt.Sprintf("%q", m[byte(1)]))

	// An interface with a method is also a key.
	n := map[shape]int{}
	n[square{2}] = 4
	n[square{3}] = 9
	n[circle{1}] = 3
	row("shape len", len(n))
	row("square 2", n[square{2}])
	row("circle 1", n[circle{1}])
	row("shape gap", n[square{9}])
	total := 0
	for k, v := range n {
		total += k.area() + v
	}
	row("sum through iface", total)

	// A nil key of an interface type.
	var nilKey shape
	n[nilKey] = -1
	row("nil key", n[nil])
	row("len with nil", len(n))
}

func absent() {
	m := map[string]int{"a": 1}
	row("absent int", m["z"])
	s := map[string]string{"a": "x"}
	row("absent string", fmt.Sprintf("%q", s["z"]))
	p := map[string][]int{"a": {1}}
	row("absent slice", p["z"] == nil)
	q := map[string]point{"a": {1, 2}}
	row("absent struct", q["z"])
	r := map[string]*point{"a": {1, 2}}
	row("absent pointer", r["z"] == nil)
	f := map[string]func() int{}
	row("absent func", f["z"] == nil)
	i := map[string]any{}
	row("absent any", i["z"] == nil)
	// An increment of a key that the map does not hold starts at the zero.
	counts := map[string]int{}
	for _, w := range []string{"a", "b", "a", "c", "a"} {
		counts[w]++
	}
	dump("counts", counts)
	// A read of a nil map gives the zero value, and it reports no error.
	var nilMap map[string]int
	row("nil map read", nilMap["a"])
	row("nil map len", len(nilMap))
	v, ok := nilMap["a"]
	row("nil map 2 result", fmt.Sprint(v, " ", ok))
	row("nil map range", rangeLen(nilMap))
	delete(nilMap, "a")
	row("delete of nil map", "no panic")
}

func rangeLen(m map[string]int) int {
	n := 0
	for range m {
		n++
	}
	return n
}

// deleteInRange proves that a delete during a range is legal. A pair that the
// loop already read stays read, and a pair that the delete removes before the
// loop reaches it does not appear.
func deleteInRange() {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	seen := []string{}
	for k := range m {
		seen = append(seen, k)
		delete(m, k)
	}
	sort.Strings(seen)
	row("each key once", seen)
	row("len after", len(m))

	// A delete of another key during the loop.
	n := map[int]int{}
	for i := 0; i < 10; i++ {
		n[i] = i
	}
	count := 0
	for k := range n {
		count++
		delete(n, k)
		delete(n, (k+1)%10)
	}
	row("count at most 10", count <= 10)
	row("len after 2", len(n))

	// A write during a range can give the new pair or not, thus the program
	// only proves that the loop ends.
	o := map[int]int{1: 1}
	steps := 0
	for k := range o {
		steps++
		if steps > 100 {
			break
		}
		if k < 3 {
			o[k+10] = k
		}
		delete(o, k)
	}
	// The specification leaves a pair that a write makes during the loop
	// open: the loop can give it, or the loop can skip it. Thus the program
	// reads no content of o here.
	row("loop ends", steps <= 100)
	row("o len at most 4", len(o) <= 4)
	dump2("the keys of n", map[int]int{1: 10, 2: 20})

	// A delete of a key that the map does not hold performs no operation.
	p := map[string]int{"a": 1}
	delete(p, "z")
	dump("p", p)
}

func dump2(label string, m map[int]int) {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%d=%d", k, m[k]))
	}
	fmt.Printf("%-26s %v\n", label, parts)
}

func clears() {
	m := map[string]int{"a": 1, "b": 2}
	clear(m)
	row("len after clear", len(m))
	row("read after clear", m["a"])
	m["c"] = 3
	dump("after write", m)
	var nilMap map[string]int
	clear(nilMap)
	row("clear of nil map", len(nilMap))
	// clear also removes a key of NaN, which no read can reach.
	n := map[float64]int{}
	n[nan()] = 1
	n[nan()] = 2
	row("2 NaN keys", len(n))
	row("read of NaN", n[nan()])
	clear(n)
	row("len after clear", len(n))
}

func nan() float64 {
	var zero float64
	return zero / zero
}

func values() {
	// A map value of a struct type is not addressable, thus the program must
	// read it, change it, and write it again.
	m := map[string]point{"a": {1, 2}}
	p := m["a"]
	p.X = 9
	m["a"] = p
	row("struct value", m["a"])
	// A map value of a pointer type reaches the field.
	n := map[string]*point{"a": {1, 2}}
	n["a"].X = 9
	row("pointer value", *n["a"])
	// A map of a map.
	o := map[string]map[string]int{}
	o["a"] = map[string]int{"b": 1}
	o["a"]["c"] = 2
	dump("inner", o["a"])
	row("outer len", len(o))
	// A map of a slice grows through append.
	q := map[string][]int{}
	q["a"] = append(q["a"], 1)
	q["a"] = append(q["a"], 2)
	row("slice value", q["a"])
	// A map is a reference, thus a copy of the variable shares the storage.
	r := map[string]int{"a": 1}
	s := r
	s["b"] = 2
	dump("shared", r)
	row("len of both", fmt.Sprint(len(r), len(s)))
	// A map holds no comparison, thus only nil compares.
	var t map[string]int
	row("nil compare", t == nil)
	row("made compare", r == nil)
}

func main() {
	fmt.Println("== a struct key ==")
	structKeys()
	fmt.Println("== an array key ==")
	arrayKeys()
	fmt.Println("== an interface key ==")
	ifaceKeys()
	fmt.Println("== an absent key ==")
	absent()
	fmt.Println("== a delete during a range ==")
	deleteInRange()
	fmt.Println("== clear ==")
	clears()
	fmt.Println("== the value ==")
	values()
}
