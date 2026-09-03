// The program holds 4 shapes that a call which blocks reaches. Each 1 gave a
// wrong result before the work of 2026-09-03.
package main

import "fmt"

// wait blocks the goroutine. A channel with a buffer of 1 gives up control and
// takes it back, thus each caller of wait becomes a state machine.
func wait(n int) int {
	ch := make(chan int, 1)
	ch <- n
	return <-ch
}

type point struct {
	x int
	y int
}

// slow gives a struct after a call that blocks.
func slow(n int) point { return point{wait(n), n * 2} }

// apply calls a function value that blocks.
func apply(f func(int) bool) bool { return f(1) }

// 1. A function literal that blocks inside the condition of an if. The
// declaration of the local function must stand in the state that reads it.
func inCondition() string {
	if !apply(func(n int) bool {
		return wait(n)+n > 0
	}) {
		return "no"
	}
	return "yes"
}

// 2. The result of a call that blocks reaches a composite literal inside a
// loop. Each element must hold its own copy.
type holder struct {
	name string
	p    point
}

func inLoop() []holder {
	var out []holder
	for i, name := range []string{"a", "b", "c"} {
		out = append(out, holder{name, slow(i + 1)})
	}
	return out
}

// 3. An element of an array takes the result of a call that blocks. The write
// must reach the array of the function, and not a copy of it.
func intoArray() [3]int {
	var index [3]int
	index[0] = wait(7)
	index[2] = wait(9)
	return index
}

// 4. A pointer inside an interface serves as a key of a map. Go reads the
// identity of the pointer, thus a write to the target keeps the key.
type named interface{ name() string }

type box struct{ n int }

func (b *box) name() string { return "box" }

func pointerKey() (int, bool, bool) {
	m := map[named]int{}
	p := &box{1}
	q := &box{1}
	m[p] = 5
	p.n = 2
	v, ok := m[p]
	_, dup := m[q]
	return v, ok, dup
}

func main() {
	fmt.Println("condition:", inCondition())
	for _, h := range inLoop() {
		fmt.Printf("holder %s %d %d\n", h.name, h.p.x, h.p.y)
	}
	fmt.Println("array:", intoArray())
	v, ok, dup := pointerKey()
	fmt.Println("map:", v, ok, dup)
}
