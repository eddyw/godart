// The program proves the closure rules of Go: a closure holds the variable
// itself and not a copy, a for loop of Go 1.22 gives each pass a new variable,
// a closure inside a defer reads the value of the moment of the run, and a
// closure calls itself through a variable of a function type.
package main

import (
	"fmt"
	"sort"
)

func row(label string, value any) {
	fmt.Printf("%-32s %v\n", label, value)
}

// byReference proves that 2 closures of 1 scope share 1 variable.
func byReference() {
	n := 0
	inc := func() { n++ }
	read := func() int { return n }
	inc()
	inc()
	row("after 2 calls", read())
	n = 10
	row("after a write outside", read())
	inc()
	row("n", n)

	// A closure that a function gives back keeps the variable alive.
	next := counterOf(5)
	row("first", next())
	row("second", next())
	row("third", next())
	other := counterOf(5)
	row("a new counter", other())
	row("the first one", next())

	// 2 closures of 1 call share the variable.
	get, set := pairOf()
	set(42)
	row("shared pair", get())
	set(43)
	row("shared pair again", get())

	// A closure captures a parameter.
	add := adderOf(10)
	row("adder", add(5))
	row("adder again", add(5))

	// A closure captures a struct field through the variable.
	type box struct{ V int }
	b := box{1}
	touch := func() { b.V *= 2 }
	touch()
	touch()
	row("struct capture", b.V)

	// A closure captures a slice header. An append that grows builds a new
	// header, thus the write must reach the variable.
	list := []int{1}
	push := func(v int) { list = append(list, v) }
	push(2)
	push(3)
	row("slice capture", list)

	// A closure captures the receiver of a method.
	c := cell{7}
	f := c.read
	c.V = 8
	row("method value copy", f())
	g := (&c).read
	c.V = 9
	row("method value pointer", g())
}

type cell struct{ V int }

func (c cell) read() int { return c.V }

func counterOf(start int) func() int {
	n := start
	return func() int {
		n++
		return n
	}
}

func pairOf() (func() int, func(int)) {
	v := 0
	return func() int { return v }, func(n int) { v = n }
}

func adderOf(base int) func(int) int {
	total := base
	return func(n int) int {
		total += n
		return total
	}
}

// loopVariable proves the rule of Go 1.22: each pass of a for loop gives a
// new variable, thus each closure holds its own value.
func loopVariable() {
	var fns []func() int
	for i := 0; i < 3; i++ {
		fns = append(fns, func() int { return i })
	}
	out := []int{}
	for _, f := range fns {
		out = append(out, f())
	}
	row("3 clause loop", out)

	var rfns []func() int
	for _, v := range []int{10, 20, 30} {
		rfns = append(rfns, func() int { return v })
	}
	out = out[:0]
	for _, f := range rfns {
		out = append(out, f())
	}
	row("range loop", out)

	var kfns []func() string
	for k, v := range map[string]int{"a": 1} {
		kfns = append(kfns, func() string { return fmt.Sprint(k, v) })
	}
	texts := []string{}
	for _, f := range kfns {
		texts = append(texts, f())
	}
	sort.Strings(texts)
	row("map loop", texts)

	// A variable that the body declares also gives a new value each pass.
	var bfns []func() int
	for i := 0; i < 3; i++ {
		j := i * 10
		bfns = append(bfns, func() int { return j })
	}
	out = out[:0]
	for _, f := range bfns {
		out = append(out, f())
	}
	row("body variable", out)

	// A variable outside the loop keeps 1 cell.
	var ofns []func() int
	k := 0
	for k = 0; k < 3; k++ {
		ofns = append(ofns, func() int { return k })
	}
	out = out[:0]
	for _, f := range ofns {
		out = append(out, f())
	}
	row("outer variable", out)

	// A write inside the closure reaches the variable of that pass.
	var wfns []func() int
	for i := 0; i < 3; i++ {
		wfns = append(wfns, func() int { i *= 2; return i })
	}
	out = out[:0]
	for _, f := range wfns {
		out = append(out, f())
	}
	row("write in the closure", out)

	// A write of the loop variable inside the body reaches the next test.
	steps := []int{}
	for i := 0; i < 10; i++ {
		steps = append(steps, i)
		i++
	}
	row("write of the variable", steps)

	// A range over an integer also gives a new variable.
	var ifns []func() int
	for i := range 3 {
		ifns = append(ifns, func() int { return i })
	}
	out = out[:0]
	for _, f := range ifns {
		out = append(out, f())
	}
	row("range over an int", out)

	// A goroutine of a loop reads the value of its own pass.
	done := make(chan int, 3)
	for i := 0; i < 3; i++ {
		go func() { done <- i }()
	}
	got := []int{}
	for j := 0; j < 3; j++ {
		got = append(got, <-done)
	}
	sort.Ints(got)
	row("goroutine of a loop", got)
}

// inDefer proves that a closure of a defer reads the value of the moment of
// the run, and not the value of the moment of the defer.
func inDefer() {
	row("plain", deferClosure())
	row("named result", deferNamed())
	row("argument at the defer", deferArg())
	row("in a loop", deferLoop())
	row("of a loop variable", deferLoopVar())
}

func deferClosure() (out string) {
	n := 1
	defer func() { out = fmt.Sprint("n at the end is ", n) }()
	n = 2
	n = 3
	return "not this"
}

func deferNamed() (n int) {
	defer func() { n *= 3 }()
	defer func() { n += 1 }()
	n = 5
	return n
}

// deferArg proves the 2 rules together. A deferred call reads each argument
// at the defer statement, and a closure reads the variable at the run. The
// second defer runs first, thus it writes the argument into the result, and
// the first defer then reads the variable and that result.
func deferArg() (out string) {
	n := 1
	defer func() {
		out = fmt.Sprint("the closure sees ", n, ", and the argument was ", out)
	}()
	defer func(seen int) { out = fmt.Sprint(seen) }(n)
	n = 99
	return ""
}

func deferLoop() []int {
	seen := []int{}
	func() {
		for i := 0; i < 3; i++ {
			defer func() { seen = append(seen, i) }()
		}
	}()
	return seen
}

func deferLoopVar() []int {
	out := []int{}
	func() {
		for _, v := range []int{1, 2, 3} {
			defer func() { out = append(out, v*10) }()
		}
	}()
	return out
}

// recursive proves a closure that calls itself through a variable.
func recursive() {
	var fact func(int) int
	fact = func(n int) int {
		if n <= 1 {
			return 1
		}
		return n * fact(n-1)
	}
	row("fact 5", fact(5))
	row("fact 10", fact(10))

	var fib func(int) int
	fib = func(n int) int {
		if n < 2 {
			return n
		}
		return fib(n-1) + fib(n-2)
	}
	row("fib 10", fib(10))

	// 2 closures that call each other.
	var even, odd func(int) bool
	even = func(n int) bool {
		if n == 0 {
			return true
		}
		return odd(n - 1)
	}
	odd = func(n int) bool {
		if n == 0 {
			return false
		}
		return even(n - 1)
	}
	row("even 10", even(10))
	row("odd 7", odd(7))

	// A closure that walks a tree.
	type node struct {
		v    int
		kids []int
	}
	tree := map[int]node{1: {1, []int{2, 3}}, 2: {2, []int{4}}, 3: {3, nil}, 4: {4, nil}}
	sum := 0
	var walk func(int)
	walk = func(id int) {
		n := tree[id]
		sum += n.v
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(1)
	row("tree sum", sum)

	// A closure that a variable of a struct field holds.
	type ops struct{ run func(int) int }
	var o ops
	o.run = func(n int) int {
		if n == 0 {
			return 0
		}
		return n + o.run(n-1)
	}
	row("through a field", o.run(4))
}

// nested proves a closure inside a closure.
func nested() {
	outer := func(a int) func(int) func(int) int {
		return func(b int) func(int) int {
			return func(c int) int { return a*100 + b*10 + c }
		}
	}
	row("3 levels", outer(1)(2)(3))

	n := 0
	level1 := func() {
		level2 := func() {
			level3 := func() { n += 5 }
			level3()
			n *= 2
		}
		level2()
		n++
	}
	level1()
	row("nested write", n)

	// A closure that a map holds.
	table := map[string]func(int) int{
		"double": func(n int) int { return n * 2 },
		"square": func(n int) int { return n * n },
	}
	keys := []string{"double", "square"}
	for _, k := range keys {
		row(k, table[k](6))
	}

	// A closure of a variadic form.
	sum := func(vals ...int) int {
		total := 0
		for _, v := range vals {
			total += v
		}
		return total
	}
	row("variadic", sum(1, 2, 3))
	row("variadic spread", sum([]int{4, 5}...))
	row("variadic empty", sum())

	// A closure that a function takes.
	row("apply", apply(3, func(n int) int { return n + 1 }))
	// A closure value compares only with nil.
	var f func()
	row("nil closure", f == nil)
	f = func() {}
	row("set closure", f == nil)
}

func apply(n int, f func(int) int) int { return f(f(n)) }

func main() {
	fmt.Println("== the capture ==")
	byReference()
	fmt.Println("== the loop variable ==")
	loopVariable()
	fmt.Println("== inside a defer ==")
	inDefer()
	fmt.Println("== the recursion ==")
	recursive()
	fmt.Println("== the nesting ==")
	nested()
}
