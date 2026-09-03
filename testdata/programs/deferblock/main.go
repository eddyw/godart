package main

import "fmt"

// take reads 1 value. The call blocks, thus each function that defers it holds
// its chain of deferred calls inside the state machine.
func take(ch chan int) int {
	return <-ch
}

// order proves that each deferred call runs after the last statement, in the
// reverse order of the defer statements.
func order(ch chan int) {
	defer fmt.Println("defer 1 read", take(ch))
	defer fmt.Println("defer 2 read", take(ch))
	fmt.Println("the body of order")
}

// result proves that a deferred call that blocks runs in front of the return,
// and that it can still write a named result.
func result(ch chan int) (n int) {
	defer func() {
		n += take(ch)
	}()
	return 5
}

// plain proves that a value result keeps the value of the return statement.
func plain(ch chan int) int {
	v := 1
	defer func() {
		v = take(ch)
	}()
	return v
}

// loop proves that a deferred call of a loop runs at the return.
func loop(ch chan int) string {
	out := ""
	for i := 0; i < 3; i++ {
		defer func() {
			out += fmt.Sprint(take(ch))
		}()
	}
	fmt.Println("the body of loop, out is", out)
	return out
}

// nested proves that a return inside a plain statement also runs the chain of
// the deferred calls. Such a statement holds no operation that blocks. Stage 4
// left it outside the state machine before. The return then wrote a plain Dart
// return that ran no deferred call.
func nested(ch chan int, pick int) (out string) {
	defer func() {
		out += fmt.Sprint(" and ", take(ch))
	}()
	if pick == 1 {
		return "the first"
	}
	switch pick {
	case 2:
		return "the second"
	}
	for i := 0; i < 2; i++ {
		if pick == 3 {
			return "the third"
		}
	}
	return "the last"
}

// stopped proves that a deferred call that blocks runs on the way out of a
// panic.
func stopped(ch chan int) {
	defer fmt.Println("the defer of stopped read", take(ch))
	panic("the reason")
}

func feed(ch chan int, count int) {
	for i := 1; i <= count; i++ {
		ch <- i
	}
}

func main() {
	ch := make(chan int)

	go feed(ch, 2)
	order(ch)
	fmt.Println("order returned")

	go feed(ch, 1)
	fmt.Println("result:", result(ch))

	go feed(ch, 1)
	fmt.Println("plain:", plain(ch))

	go feed(ch, 3)
	fmt.Println("loop:", loop(ch))

	for pick := 1; pick <= 4; pick++ {
		go feed(ch, 1)
		fmt.Println("nested:", nested(ch, pick))
	}

	go feed(ch, 1)
	stopped(ch)
}
