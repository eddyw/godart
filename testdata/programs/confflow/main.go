// The program proves the control flow of Go: a labeled break and a labeled
// continue, goto, a switch with no condition, fallthrough, a switch on a
// type, a select with a default, and each form of the for statement.
package main

import "fmt"

func row(label string, value any) {
	fmt.Printf("%-32s %v\n", label, value)
}

func labels() {
	// A labeled break leaves the outer loop.
	pairs := []string{}
outer:
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if i*j > 4 {
				break outer
			}
			pairs = append(pairs, fmt.Sprint(i, j))
		}
	}
	row("a labeled break", pairs)

	// A labeled continue takes the next pass of the outer loop.
	pairs = pairs[:0]
next:
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if j == 2 {
				continue next
			}
			pairs = append(pairs, fmt.Sprint(i, j))
		}
	}
	row("a labeled continue", pairs)

	// A plain break leaves the inner loop only.
	pairs = pairs[:0]
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				break
			}
			pairs = append(pairs, fmt.Sprint(i, j))
		}
	}
	row("a plain break", pairs)

	// A label on a switch inside a loop.
	out := []int{}
loop:
	for i := 0; i < 5; i++ {
		switch {
		case i == 2:
			continue loop
		case i == 4:
			break loop
		}
		out = append(out, i)
	}
	row("a label of a switch", out)

	// A label on a select inside a loop.
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	got := []int{}
feed:
	for {
		select {
		case v := <-ch:
			if v == 3 {
				break feed
			}
			got = append(got, v)
		default:
			break feed
		}
	}
	row("a label of a select", got)

	// A break of a switch alone leaves the switch and not the loop.
	out = out[:0]
	for i := 0; i < 3; i++ {
		switch i {
		case 1:
			break
		}
		out = append(out, i)
	}
	row("a break of a switch", out)

	// A labeled loop that a range clause holds.
	out = out[:0]
walk:
	for _, a := range []int{1, 2, 3} {
		for _, b := range []int{10, 20} {
			if a == 2 {
				continue walk
			}
			if a == 3 {
				break walk
			}
			out = append(out, a*b)
		}
	}
	row("a labeled range", out)
}

func gotos() {
	// A goto that moves backward.
	total := 0
	i := 0
loop:
	if i < 5 {
		total += i
		i++
		goto loop
	}
	row("a backward goto", total)

	// A goto that moves forward.
	n := 1
	if n > 0 {
		goto end
	}
	n = 2
end:
	row("a forward goto", n)

	// A goto that leaves a nested block.
	found := -1
	list := [][]int{{1, 2}, {3, 4}}
	for a := range list {
		for b := range list[a] {
			if list[a][b] == 3 {
				found = a*10 + b
				goto done
			}
		}
	}
done:
	row("a goto out of 2 loops", found)

	// A goto inside a switch.
	step := 0
	v := 2
again:
	switch v {
	case 2:
		step++
		v = 1
		goto again
	case 1:
		step += 10
	}
	row("a goto of a switch", step)

	// A goto that skips a declaration must not jump over it. The declaration
	// thus stands in front of the label.
	x := 0
	if x == 0 {
		goto tail
	}
	x = 99
tail:
	row("a goto over a write", x)
}

func switches() {
	// A switch with no condition holds a chain of tests.
	classify := func(n int) string {
		switch {
		case n < 0:
			return "negative"
		case n == 0:
			return "zero"
		case n < 10:
			return "small"
		default:
			return "large"
		}
	}
	for _, n := range []int{-1, 0, 5, 100} {
		row(fmt.Sprint("classify ", n), classify(n))
	}

	// A switch with a value.
	name := func(n int) string {
		switch n {
		case 1:
			return "one"
		case 2, 3:
			return "two or three"
		default:
			return "other"
		}
	}
	for _, n := range []int{1, 2, 3, 4} {
		row(fmt.Sprint("name ", n), name(n))
	}

	// A switch with a statement in front of the value.
	got := ""
	switch v := 5 * 2; v {
	case 10:
		got = fmt.Sprint("ten ", v)
	default:
		got = "other"
	}
	row("a statement in a switch", got)

	// fallthrough runs the body of the next case, and it tests no value.
	steps := []string{}
	switch 1 {
	case 1:
		steps = append(steps, "one")
		fallthrough
	case 2:
		steps = append(steps, "two")
		fallthrough
	case 3:
		steps = append(steps, "three")
	case 4:
		steps = append(steps, "four")
	}
	row("fallthrough", steps)

	// fallthrough into a default.
	steps = steps[:0]
	switch 9 {
	case 9:
		steps = append(steps, "nine")
		fallthrough
	default:
		steps = append(steps, "default")
	}
	row("fallthrough to a default", steps)

	// A default that stands in front of each case.
	got = ""
	switch 7 {
	default:
		got = "default"
	case 1:
		got = "one"
	}
	row("a default in front", got)

	// A switch of a text and of a struct type.
	type point struct{ X, Y int }
	switch (point{1, 2}) {
	case point{0, 0}:
		got = "zero"
	case point{1, 2}:
		got = "one two"
	}
	row("a switch of a struct", got)
	switch "b" {
	case "a":
		got = "a"
	case "b":
		got = "b"
	}
	row("a switch of a text", got)

	// An empty switch and a case with an empty body.
	switch {
	}
	switch 1 {
	case 1:
	}
	row("an empty body", "no operation")

	// A switch that a break leaves early.
	steps = steps[:0]
	switch 1 {
	case 1:
		steps = append(steps, "start")
		if true {
			break
		}
		steps = append(steps, "not reached")
	}
	row("a break of a case", steps)

	// The value of a case runs in order, and the search stops at the match.
	calls := []int{}
	probe := func(n int) int { calls = append(calls, n); return n }
	switch 2 {
	case probe(1):
	case probe(2):
	case probe(3):
	}
	row("the order of the tests", calls)
}

type shape interface{ Area() int }

type square struct{ Side int }

func (s square) Area() int { return s.Side * s.Side }

func typeSwitches() {
	describe := func(v any) string {
		switch x := v.(type) {
		case nil:
			return "nil"
		case int, int64:
			return fmt.Sprint("an integer ", x)
		case string:
			return fmt.Sprint("a text of ", len(x))
		case []int:
			return fmt.Sprint("a slice of ", len(x))
		case shape:
			return fmt.Sprint("a shape of ", x.Area())
		case error:
			return "an error"
		default:
			return fmt.Sprintf("other %T", x)
		}
	}
	for _, v := range []any{nil, 1, int64(2), "abc", []int{1, 2},
		square{3}, 1.5, true} {
		row(fmt.Sprintf("%T", v), describe(v))
	}
	// A type switch with a statement in front.
	got := ""
	switch v := any(5); v.(type) {
	case int:
		got = "int"
	}
	row("a statement in front", got)
	// A type switch with no binding and with a fall to the default.
	var v any = 1.5
	switch v.(type) {
	case int:
		got = "int"
	default:
		got = "default"
	}
	row("no binding", got)
	// A type switch that a break leaves.
	steps := []string{}
	switch any(1).(type) {
	case int:
		steps = append(steps, "start")
		break
	}
	row("a break of a type case", steps)
}

func loops() {
	// The 3 clause form.
	out := []int{}
	for i := 0; i < 3; i++ {
		out = append(out, i)
	}
	row("3 clauses", out)
	// The condition form.
	out = out[:0]
	i := 0
	for i < 3 {
		out = append(out, i)
		i++
	}
	row("a condition", out)
	// The endless form with a break.
	out = out[:0]
	i = 0
	for {
		if i == 3 {
			break
		}
		out = append(out, i)
		i++
	}
	row("no clause", out)
	// A clause with no initializer and no step.
	out = out[:0]
	i = 0
	for i < 3 {
		out = append(out, i)
		i++
	}
	row("only a condition", out)
	// A step with 2 variables.
	pairs := []string{}
	for a, b := 0, 5; a < b; a, b = a+1, b-1 {
		pairs = append(pairs, fmt.Sprint(a, b))
	}
	row("2 variables", pairs)
	// A loop with no pass.
	n := 0
	for i := 0; i < 0; i++ {
		n++
	}
	row("no pass", n)
	// A continue runs the step clause.
	out = out[:0]
	for i := 0; i < 5; i++ {
		if i%2 == 0 {
			continue
		}
		out = append(out, i)
	}
	row("a continue", out)
	// A nested loop that a variable of the outer loop controls.
	out = out[:0]
	for a := 0; a < 3; a++ {
		for b := a; b < 3; b++ {
			out = append(out, a*10+b)
		}
	}
	row("a nested loop", out)
	// A loop that a goroutine and a channel end.
	ch := make(chan int)
	go func() {
		for i := 0; i < 3; i++ {
			ch <- i
		}
		close(ch)
	}()
	out = out[:0]
	for {
		v, ok := <-ch
		if !ok {
			break
		}
		out = append(out, v)
	}
	row("a channel loop", out)
}

// selects proves a select with a default and a select of 1 case. The program
// writes no select with no case, because such a statement blocks forever.
func selects() {
	ch := make(chan int, 1)
	got := ""
	select {
	case v := <-ch:
		got = fmt.Sprint("read ", v)
	default:
		got = "the default"
	}
	row("an empty channel", got)
	ch <- 7
	select {
	case v := <-ch:
		got = fmt.Sprint("read ", v)
	default:
		got = "the default"
	}
	row("a ready channel", got)
	// A select of 1 case waits for that case.
	done := make(chan string)
	go func() { done <- "ready" }()
	select {
	case v := <-done:
		got = v
	}
	row("a select of 1 case", got)
	// A select inside a loop with a label.
	work := make(chan int, 3)
	work <- 1
	work <- 2
	close(work)
	sum := 0
run:
	for {
		select {
		case v, ok := <-work:
			if !ok {
				break run
			}
			sum += v
		}
	}
	row("a loop of a select", sum)
}

func ifs() {
	// An if with a statement in front.
	got := ""
	if v := 5; v > 3 {
		got = fmt.Sprint("above ", v)
	} else {
		got = fmt.Sprint("below ", v)
	}
	row("a statement in front", got)
	// An else if chain.
	grade := func(n int) string {
		if n >= 90 {
			return "A"
		} else if n >= 80 {
			return "B"
		} else if n >= 70 {
			return "C"
		}
		return "F"
	}
	for _, n := range []int{95, 85, 75, 65} {
		row(fmt.Sprint("grade ", n), grade(n))
	}
	// The variable of the statement reaches the else branch.
	if v, ok := lookup("a"); ok {
		row("found", v)
	} else {
		row("not found", v)
	}
	if v, ok := lookup("z"); ok {
		row("found", v)
	} else {
		row("not found", fmt.Sprintf("%q", v))
	}
}

func lookup(k string) (string, bool) {
	m := map[string]string{"a": "value of a"}
	v, ok := m[k]
	return v, ok
}

func closureGoto() int {
	f := func() int {
		n := 0
	again:
		n++
		if n < 3 {
			goto again
		}
		return n
	}
	return f()
}

func labeledFallthrough(value int) int {
	switch value {
	case 0:
		goto next
	next:
		fallthrough
	case 1:
		return 7
	}
	return 9
}

func main() {
	println(closureGoto())
	println(labeledFallthrough(0), labeledFallthrough(1), labeledFallthrough(2))
	fmt.Println("== a label ==")
	labels()
	fmt.Println("== goto ==")
	gotos()
	fmt.Println("== a switch ==")
	switches()
	fmt.Println("== a type switch ==")
	typeSwitches()
	fmt.Println("== a loop ==")
	loops()
	fmt.Println("== a select ==")
	selects()
	fmt.Println("== an if ==")
	ifs()
}
