// The program proves the assignment rules of Go: an assignment of many values
// reads each operand of the right side first and it then writes each target
// from left to right, a swap thus needs no temporary variable, and the
// operands of a call take their value in the order that the specification
// gives.
//
// Go rejects an assignment to a field of a value of a map, for example
// `m["a"].X = 1`, because such a value is not addressable. The program holds
// the form that works in its place.
package main

import "fmt"

func row(label string, value any) {
	fmt.Printf("%-32s %v\n", label, value)
}

func multiple() {
	// The right side reads each value first.
	a, b := 1, 2
	a, b = b, a
	row("a swap", fmt.Sprint(a, " ", b))

	x, y, z := 1, 2, 3
	x, y, z = z, x, y
	row("a rotation", fmt.Sprint(x, " ", y, " ", z))

	// The read of the old value.
	n := 1
	n, m := n+10, n+20
	row("the old value", fmt.Sprint(n, " ", m))

	// An index of the left side takes its value before the write.
	list := []int{0, 1, 2}
	i := 0
	i, list[i] = 2, 99
	row("the index of the left", fmt.Sprint(i, " ", list))

	list = []int{0, 1, 2}
	i = 0
	list[i], i = 99, 2
	row("the other order", fmt.Sprint(i, " ", list))

	// A swap of 2 elements of 1 slice.
	s := []int{1, 2, 3}
	s[0], s[2] = s[2], s[0]
	row("a swap of elements", s)

	// A swap through a pointer.
	p, q := 1, 2
	pp, qq := &p, &q
	*pp, *qq = *qq, *pp
	row("a swap of pointers", fmt.Sprint(p, " ", q))

	// A rotation of a slice.
	r := []int{1, 2, 3, 4}
	r[0], r[1], r[2], r[3] = r[3], r[0], r[1], r[2]
	row("a rotation of a slice", r)

	// A struct field on each side.
	type point struct{ X, Y int }
	pt := point{1, 2}
	pt.X, pt.Y = pt.Y, pt.X
	row("a swap of fields", pt)

	// A map value on each side.
	table := map[string]int{"a": 1, "b": 2}
	table["a"], table["b"] = table["b"], table["a"]
	row("a swap of map values", fmt.Sprint(table["a"], " ", table["b"]))

	// A blank name takes a value and drops it.
	_, second := two()
	row("a blank name", second)
	first, _ := two()
	row("the other side", first)
	_, _ = two()
	row("2 blank names", "no operation")

	// The 2 result form of a map, of an assertion, and of a channel.
	v, ok := table["z"]
	row("a map read", fmt.Sprint(v, " ", ok))
	var face any = 5
	iv, ok := face.(int)
	row("an assertion", fmt.Sprint(iv, " ", ok))
	ch := make(chan int, 1)
	close(ch)
	cv, ok := <-ch
	row("a channel read", fmt.Sprint(cv, " ", ok))
}

func two() (int, int) { return 1, 2 }

// order proves the order of the evaluation. Go runs each call, each method
// call, each receive, and each binary logical operation from left to right.
var steps []string

func note(s string) { steps = append(steps, s) }

func value(n int, label string) int {
	note(label)
	return n
}

func evaluation() {
	steps = nil
	sum := value(1, "a") + value(2, "b") + value(3, "c")
	row("a sum", fmt.Sprint(sum, " ", steps))

	steps = nil
	got := fmt.Sprint(value(1, "x"), value(2, "y"), value(3, "z"))
	row("the arguments of a call", fmt.Sprint(got, " ", steps))

	steps = nil
	list := []int{value(1, "1"), value(2, "2")}
	row("a composite literal", fmt.Sprint(list, " ", steps))

	steps = nil
	table := map[string]int{"a": value(1, "va"), "b": value(2, "vb")}
	row("a map literal", fmt.Sprint(table["a"], table["b"], " ", steps))

	steps = nil
	a, b := value(1, "p"), value(2, "q")
	row("a short declaration", fmt.Sprint(a, b, " ", steps))

	// A logical operator stops at the answer.
	steps = nil
	res := truth(false, "l1") && truth(true, "l2")
	row("&& stops", fmt.Sprint(res, " ", steps))
	steps = nil
	res = truth(true, "r1") || truth(false, "r2")
	row("|| stops", fmt.Sprint(res, " ", steps))
	steps = nil
	res = truth(true, "s1") && truth(false, "s2")
	row("&& runs both", fmt.Sprint(res, " ", steps))

	// A method call reads the receiver first.
	steps = nil
	c := holder{3}
	got = fmt.Sprint(c.at("recv").add(value(4, "arg")))
	row("a receiver first", fmt.Sprint(got, " ", steps))

	// An index expression.
	steps = nil
	arr := []int{10, 20, 30}
	n := arr[value(1, "idx")] + value(5, "add")
	row("an index", fmt.Sprint(n, " ", steps))

	// A call inside an index of the left side.
	steps = nil
	target := []int{0, 0, 0}
	target[value(1, "left")] = value(9, "right")
	row("the left side first", fmt.Sprint(target, " ", steps))

	// The write of a result list reads each call first.
	steps = nil
	p, q := pairOf()
	row("a result list", fmt.Sprint(p, " ", q, " ", steps))

	// A defer reads each argument at the statement.
	steps = nil
	func() {
		defer note("deferred")
		note(fmt.Sprint("arg ", value(1, "in defer")))
	}()
	row("a defer", steps)

	// A send statement reads the channel and then the value.
	steps = nil
	ch := chanOf(1)
	ch <- value(7, "sent")
	row("a send", fmt.Sprint(<-ch, " ", steps))
}

func truth(v bool, label string) bool { note(label); return v }

type holder struct{ N int }

func (h holder) at(label string) holder { note(label); return h }

func (h holder) add(n int) int { return h.N + n }

func pairOf() (int, int) {
	note("call")
	return value(1, "first"), value(2, "second")
}

func chanOf(n int) chan int {
	note("chan")
	return make(chan int, n)
}

// mapValues proves the forms that reach a field of a value of a map.
type point struct{ X, Y int }

func mapValues() {
	// Go rejects `m["a"].X = 1`, because a value of a map is not
	// addressable. The 3 forms below all work.
	m := map[string]point{"a": {1, 2}}

	// 1. Read the value, change it, and write it again.
	v := m["a"]
	v.X = 9
	m["a"] = v
	row("read, change, write", m["a"])

	// 2. A whole new value.
	m["a"] = point{7, 8}
	row("a whole value", m["a"])

	// 3. A map of a pointer type reaches the field.
	p := map[string]*point{"a": {1, 2}}
	p["a"].X = 9
	row("a map of pointers", *p["a"])

	// A read of a field of a value of a map is legal.
	row("a read of a field", m["a"].X)
	row("a read of a gap", m["z"].X)

	// An element of a slice and of an array is addressable.
	list := []point{{1, 2}}
	list[0].X = 9
	row("an element of a slice", list)
	var arr [1]point
	arr[0].Y = 5
	row("an element of an array", arr)

	// A field of a struct inside a struct.
	type outer struct{ In point }
	var o outer
	o.In.X = 3
	row("a nested field", o)

	// An operation that assigns on a map value.
	counts := map[string]int{}
	counts["a"] += 2
	counts["a"]++
	row("an operation on a map", counts["a"])

	// A slice inside a map grows through append.
	lists := map[string][]int{}
	lists["a"] = append(lists["a"], 1, 2)
	row("an append into a map", lists["a"])

	// A map inside a map.
	nested := map[string]map[string]int{"a": {}}
	nested["a"]["b"] = 1
	row("a map in a map", nested["a"]["b"])
}

func operations() {
	n := 10
	n += 5
	row("+=", n)
	n -= 3
	row("-=", n)
	n *= 2
	row("*=", n)
	n /= 4
	row("/=", n)
	n %= 4
	row("%%=", n)
	n <<= 3
	row("<<=", n)
	n >>= 1
	row(">>=", n)
	n |= 1
	row("|=", n)
	n &= 6
	row("&=", n)
	n ^= 3
	row("^=", n)
	n &^= 1
	row("&^=", n)
	s := "a"
	s += "b"
	row("+= of a text", s)
	list := []int{1}
	list = append(list, 2)
	row("an append", list)
	// An operation that assigns reads the target 1 time.
	steps = nil
	target := []int{0, 0}
	target[value(1, "index")] += value(5, "value")
	row("the index reads 1 time", fmt.Sprint(target, " ", steps))
	// An increment of a field.
	type box struct{ N int }
	b := box{1}
	b.N++
	row("a field", b)
	pb := &b
	pb.N += 10
	row("through a pointer", b)
}

func declarations() {
	// A short declaration needs 1 new name at least.
	a := 1
	a, b := 2, 3
	row("1 new name", fmt.Sprint(a, " ", b))
	// A declaration of a block hides the outer name.
	x := 1
	{
		x := 2
		_ = x
	}
	row("a name of a block", x)
	// A declaration inside an if hides the outer name.
	if x := 3; x > 0 {
		_ = x
	}
	row("a name of an if", x)
	// A var declaration of many names.
	var p, q, r int = 1, 2, 3
	row("a var of 3 names", fmt.Sprint(p, q, r))
	var s, t = "a", 2
	row("2 types", fmt.Sprint(s, t))
	var (
		u = 1
		v = u + 1
	)
	row("a block of vars", fmt.Sprint(u, v))
	// The zero value of each type.
	var (
		zi int
		zf float64
		zs string
		zb bool
		zp *int
		zl []int
		zm map[string]int
		zc chan int
		zn func()
		za [2]int
		zt struct{ N int }
	)
	row("the zero values", fmt.Sprint(zi, " ", zf, " ", fmt.Sprintf("%q", zs),
		" ", zb, " ", zp == nil, " ", zl == nil, " ", zm == nil, " ",
		zc == nil, " ", zn == nil, " ", za, " ", zt))
}

func main() {
	fmt.Println("== an assignment of many values ==")
	multiple()
	fmt.Println("== the order of the evaluation ==")
	evaluation()
	fmt.Println("== a value of a map ==")
	mapValues()
	fmt.Println("== an operation that assigns ==")
	operations()
	fmt.Println("== a declaration ==")
	declarations()
}
