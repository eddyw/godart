// The program proves the defer rules of Go: each deferred call runs in the
// reverse order of the defer statements, each argument of a deferred call
// takes its value at the defer statement, a deferred call writes a named
// result, a defer inside a loop waits for the end of the function, and
// recover gives a value only inside a deferred call of the panicking
// function.
package main

import (
	"errors"
	"fmt"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

// order proves the reverse order.
func order() []string {
	out := []string{}
	add := func(s string) { out = append(out, s) }
	func() {
		defer add("first")
		defer add("second")
		defer add("third")
		add("body")
	}()
	return out
}

// arguments proves that each argument takes its value at the defer statement.
func arguments() {
	out := []string{}
	add := func(s string) { out = append(out, s) }
	func() {
		n := 1
		defer add(fmt.Sprint("plain ", n))
		defer add(fmt.Sprint("call ", double(n)))
		n = 99
		add(fmt.Sprint("body ", n))
	}()
	row("arguments", out)

	// The receiver of a deferred method also takes its value at the defer.
	c := counter{1}
	func() {
		defer c.show("value receiver")
		defer c.ptr().show("pointer of a copy")
		c.N = 50
	}()
	row("counter now", c.N)

	// A closure reads the value of the moment of the run.
	out = out[:0]
	func() {
		n := 1
		defer func() { add(fmt.Sprint("closure ", n)) }()
		n = 99
	}()
	row("closure", out)

	// An index and a receiver of a slice take their value at the defer.
	list := []int{1, 2, 3}
	out = out[:0]
	func() {
		i := 0
		defer add(fmt.Sprint("element ", list[i]))
		i = 2
		list[0] = 100
	}()
	row("index", out)
}

func double(n int) int { return n * 2 }

type counter struct{ N int }

func (c counter) show(label string) { fmt.Printf("%-34s %d\n", label, c.N) }

type ptrView struct{ c *counter }

func (c counter) ptr() ptrView { return ptrView{&c} }

func (p ptrView) show(label string) { fmt.Printf("%-34s %d\n", label, p.c.N) }

// namedResult proves that a deferred call writes a named result.
func namedResult() {
	row("plain", plain())
	row("doubled", doubled())
	row("chain", chain())
	row("early return", early(true))
	row("late return", early(false))
	v, err := pair()
	row("pair", fmt.Sprint(v, " ", err))
	row("no name", noName())
	row("panic to result", panicToResult())
	row("2 named", twoNamed())
}

func plain() int {
	n := 1
	defer func() { n = 99 }()
	return n
}

func doubled() (n int) {
	defer func() { n *= 2 }()
	return 21
}

func chain() (n int) {
	defer func() { n += 1 }()
	defer func() { n *= 10 }()
	defer func() { n += 2 }()
	n = 1
	return n
}

func early(flag bool) (text string) {
	defer func() { text += "!" }()
	if flag {
		text = "yes"
		return
	}
	return "no"
}

func pair() (n int, err error) {
	defer func() {
		n++
		err = fmt.Errorf("wrapped: %w", err)
	}()
	return 1, errors.New("base")
}

func noName() int {
	out := 1
	defer func() { out = 99 }()
	return out
}

func panicToResult() (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprint("recovered ", r)
		}
	}()
	panic("boom")
}

func twoNamed() string {
	a, b := swapNames()
	return fmt.Sprint(a, " ", b)
}

func swapNames() (x, y string) {
	defer func() { x, y = y, x }()
	return "left", "right"
}

// loops proves a defer inside a loop.
func loops() {
	out := []int{}
	func() {
		for i := 0; i < 5; i++ {
			defer func() { out = append(out, i) }()
		}
	}()
	row("loop of 5", out)

	out = out[:0]
	func() {
		for i := 0; i < 3; i++ {
			defer collect(&out, i)
		}
	}()
	row("with an argument", out)

	// A defer inside a nested block runs at the end of the function, and not
	// at the end of the block.
	steps := []string{}
	func() {
		steps = append(steps, "start")
		{
			defer func() { steps = append(steps, "block defer") }()
			steps = append(steps, "in block")
		}
		steps = append(steps, "after block")
	}()
	row("block", steps)

	// A defer that a condition guards.
	row("guarded true", guarded(true))
	row("guarded false", guarded(false))

	// A defer inside a range over a channel.
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	close(ch)
	out = out[:0]
	func() {
		for v := range ch {
			defer func() { out = append(out, v) }()
		}
	}()
	row("range over a channel", out)
}

func collect(out *[]int, n int) { *out = append(*out, n) }

func guarded(flag bool) []string {
	out := []string{}
	func() {
		if flag {
			defer func() { out = append(out, "guarded") }()
		}
		defer func() { out = append(out, "always") }()
	}()
	return out
}

// recovers proves where recover gives a value.
func recovers() {
	row("inside a deferred call", insideDefer())
	row("outside a deferred call", outsideDefer())
	row("in a plain call", plainCall())
	row("nested function", nestedRecover())
	row("no panic", noPanic())
	row("2 levels", twoLevels())
	row("panic in a deferred call", panicInDefer())
	row("recover of a nil panic", nilPanic())
	row("recover twice", recoverTwice())
	row("panic after recover", panicAfterRecover())
	row("the value type", valueType())
	row("an error value", errorValue())
	row("the order of the chain", chainOrder())
}

func insideDefer() (out string) {
	defer func() { out = fmt.Sprint(recover()) }()
	panic("A")
}

func outsideDefer() (out string) {
	defer func() { out = fmt.Sprint("in the defer: ", recover()) }()
	// A call of recover outside a deferred call gives nil.
	if r := recover(); r != nil {
		return "recover gave a value"
	}
	panic("B")
}

func plainCall() (out string) {
	defer func() {
		// A recover inside a call of a deferred call gives nil in Go.
		out = fmt.Sprint("helper=", helperRecover(), " direct=", recover())
	}()
	panic("C")
}

func helperRecover() any { return recover() }

func nestedRecover() (out string) {
	defer func() { out = fmt.Sprint("outer sees ", recover()) }()
	func() {
		defer func() { _ = recover() }()
		panic("D")
	}()
	return "the inner call recovered"
}

func noPanic() (out string) {
	defer func() { out = fmt.Sprint("recover gives ", recover()) }()
	return "no panic"
}

func twoLevels() (out string) {
	defer func() { out = fmt.Sprint("outer ", recover()) }()
	inner()
	return "not reached"
}

func inner() { panic("E") }

func panicInDefer() (out string) {
	defer func() { out = fmt.Sprint("last ", recover()) }()
	defer func() { panic("F") }()
	panic("G")
}

func nilPanic() (out string) {
	defer func() {
		r := recover()
		out = fmt.Sprintf("%v %T", r, r)
	}()
	panic(nil)
}

func recoverTwice() (out string) {
	defer func() {
		first := recover()
		second := recover()
		out = fmt.Sprint(first, " then ", second)
	}()
	panic("H")
}

func panicAfterRecover() (out string) {
	defer func() { out = fmt.Sprint("second ", recover()) }()
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprint("again after ", r))
		}
	}()
	panic("I")
}

func valueType() (out string) {
	defer func() {
		r := recover()
		out = fmt.Sprintf("%v %T", r, r)
	}()
	panic(myError{7})
}

type myError struct{ Code int }

func (m myError) Error() string { return fmt.Sprint("code ", m.Code) }

func errorValue() (out string) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			out = "error: " + e.Error()
		} else {
			out = "not an error"
		}
	}()
	panic(errors.New("J"))
}

func chainOrder() []string {
	steps := []string{}
	func() {
		defer func() { steps = append(steps, "one") }()
		defer func() { steps = append(steps, "two") }()
		defer func() {
			steps = append(steps, fmt.Sprint("recovered ", recover()))
		}()
		defer func() { steps = append(steps, "three") }()
		panic("K")
	}()
	return steps
}

// blocking proves a deferred call that gives up control.
func blocking() {
	ch := make(chan string, 4)
	func() {
		defer func() { ch <- "third" }()
		defer func() { ch <- "second" }()
		ch <- "first"
	}()
	close(ch)
	out := []string{}
	for v := range ch {
		out = append(out, v)
	}
	row("through a channel", out)

	// A deferred call that reads a channel.
	feed := make(chan int, 1)
	feed <- 9
	got := 0
	func() {
		defer func() { got = <-feed }()
	}()
	row("a read in a defer", got)
}

func main() {
	fmt.Println("== the order ==")
	row("order", order())
	fmt.Println("== the argument ==")
	arguments()
	fmt.Println("== the named result ==")
	namedResult()
	fmt.Println("== the loop ==")
	loops()
	fmt.Println("== recover ==")
	recovers()
	fmt.Println("== a block operation ==")
	blocking()
}
