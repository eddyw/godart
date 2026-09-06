// The program proves each form of the range clause of Go: an integer, a
// function of 1 value and of 2 values, a string, a slice, an array, a pointer
// to an array, a map, and a channel.
package main

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"sort"
)

func row(label string, value any) {
	fmt.Printf("%-32s %v\n", label, value)
}

// overInt proves the range over an integer, which Go 1.22 added.
func overInt() {
	out := []int{}
	for i := range 5 {
		out = append(out, i)
	}
	row("range 5", out)
	out = out[:0]
	for range 3 {
		out = append(out, 1)
	}
	row("no variable", out)
	n := 0
	for range 0 {
		n++
	}
	row("range 0", n)
	neg := -3
	for range neg {
		n++
	}
	row("a negative count", n)
	// A named integer type and a narrow type.
	type count int8
	out = out[:0]
	for i := range count(4) {
		out = append(out, int(i))
	}
	row("a named type", out)
	var u uint8 = 3
	out = out[:0]
	for i := range u {
		out = append(out, int(i))
	}
	row("a uint8", out)
	// A break and a continue.
	out = out[:0]
	for i := range 10 {
		if i == 2 {
			continue
		}
		if i == 5 {
			break
		}
		out = append(out, i)
	}
	row("break and continue", out)
	// The variable of each pass is new.
	fns := []func() int{}
	for i := range 3 {
		fns = append(fns, func() int { return i })
	}
	out = out[:0]
	for _, f := range fns {
		out = append(out, f())
	}
	row("a closure of each pass", out)
	// A range over a call.
	out = out[:0]
	for i := range three() {
		out = append(out, i)
	}
	row("a call in the clause", out)
}

func three() int { return 3 }

// The sequence functions prove a range over a function.
func upTo(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func pairsOf(list []string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, v := range list {
			if !yield(i, v) {
				return
			}
		}
	}
}

// counted reports how many values the loop took before it stopped.
func counted(n int, stop *int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			*stop = i
			if !yield(i) {
				return
			}
		}
	}
}

func overFunc() {
	out := []int{}
	for v := range upTo(4) {
		out = append(out, v)
	}
	row("Seq of 4", out)

	out = out[:0]
	for v := range upTo(10) {
		if v == 3 {
			break
		}
		out = append(out, v)
	}
	row("a break of a Seq", out)

	stop := -1
	out = out[:0]
	for v := range counted(10, &stop) {
		out = append(out, v)
		if v == 2 {
			break
		}
	}
	row("the yield saw the stop", fmt.Sprint(out, " ", stop))

	texts := []string{}
	for i, s := range pairsOf([]string{"a", "b", "c"}) {
		texts = append(texts, fmt.Sprint(i, s))
	}
	row("Seq2", texts)

	texts = texts[:0]
	for i := range pairsOf([]string{"a", "b"}) {
		texts = append(texts, fmt.Sprint(i))
	}
	row("Seq2 with 1 variable", texts)

	n := 0
	for range upTo(5) {
		n++
	}
	row("Seq with no variable", n)

	// A return inside the loop leaves the whole function.
	row("a return inside", firstOver(3))

	// A continue of a range over a function.
	out = out[:0]
	for v := range upTo(6) {
		if v%2 == 0 {
			continue
		}
		out = append(out, v)
	}
	row("a continue", out)

	// A range over a function of the standard library.
	list := []int{3, 1, 2}
	got := []int{}
	for _, v := range slices.All(list) {
		got = append(got, v)
	}
	row("slices.All", got)
	got = got[:0]
	for v := range slices.Values(list) {
		got = append(got, v)
	}
	row("slices.Values", got)
	rev := []int{}
	for _, v := range slices.Backward(list) {
		rev = append(rev, v)
	}
	row("slices.Backward", rev)
	m := map[string]int{"a": 1, "b": 2}
	keys := []string{}
	for k := range maps.Keys(m) {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	row("maps.Keys", keys)
	vals := []int{}
	for v := range maps.Values(m) {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	row("maps.Values", vals)

	// iter.Pull gives the values 1 at a time.
	next, done := iter.Pull(upTo(3))
	pulled := []string{}
	for {
		v, ok := next()
		if !ok {
			break
		}
		pulled = append(pulled, fmt.Sprint(v))
	}
	done()
	row("iter.Pull", pulled)
	next2, done2 := iter.Pull2(pairsOf([]string{"x", "y"}))
	pulled = pulled[:0]
	for {
		i, s, ok := next2()
		if !ok {
			break
		}
		pulled = append(pulled, fmt.Sprint(i, s))
	}
	done2()
	row("iter.Pull2", pulled)
}

func firstOver(n int) int {
	for v := range upTo(10) {
		if v > n {
			return v
		}
	}
	return -1
}

func overString() {
	inputs := []string{"abc", "héllo", "日本", "a\x80b", ""}
	for _, s := range inputs {
		parts := []string{}
		for i, r := range s {
			parts = append(parts, fmt.Sprintf("%d:%U", i, r))
		}
		fmt.Printf("%-12q %v\n", s, parts)
	}
	// 1 variable gives the byte offset.
	offsets := []int{}
	for i := range "日本語" {
		offsets = append(offsets, i)
	}
	row("the offsets", offsets)
	// No variable counts the runes.
	n := 0
	for range "日本語" {
		n++
	}
	row("the count of runes", n)
	// A break gives the offset of the moment.
	last := -1
	for i, r := range "abcdef" {
		last = i
		if r == 'c' {
			break
		}
	}
	row("the offset at a break", last)
	// A write of the variable does not change the walk.
	out := []rune{}
	for _, r := range "abc" {
		r = 'z'
		out = append(out, r)
	}
	row("a write of the variable", string(out))
}

func overArray() {
	arr := [4]int{1, 2, 3, 4}
	sum := 0
	for i, v := range arr {
		sum += i * v
	}
	row("an array", sum)

	// The range clause reads a copy of an array, thus a write does not reach
	// the walk.
	got := []int{}
	for i, v := range arr {
		if i == 0 {
			arr[1] = 99
		}
		got = append(got, v)
	}
	row("a write during the walk", got)
	row("the array now", arr)

	// A pointer to an array does not take a copy, thus the walk reads the
	// write.
	arr = [4]int{1, 2, 3, 4}
	p := &arr
	got = got[:0]
	for i, v := range p {
		if i == 0 {
			p[1] = 99
		}
		got = append(got, v)
	}
	row("through a pointer", got)
	row("the array now", arr)

	// A nil pointer to an array with 1 variable reads the length only.
	var nilp *[3]int
	n := 0
	for range nilp {
		n++
	}
	row("a nil pointer, no value", n)

	// A slice takes the header at the start, thus an append does not grow the
	// walk.
	s := []int{1, 2, 3}
	got = got[:0]
	for i, v := range s {
		if i == 0 {
			s = append(s, 99)
		}
		got = append(got, v)
	}
	row("an append during the walk", got)
	row("the slice now", s)

	// A write of an element reaches the walk, because the storage is shared.
	s = []int{1, 2, 3}
	got = got[:0]
	for i, v := range s {
		if i == 0 {
			s[1] = 99
		}
		got = append(got, v)
	}
	row("a write of an element", got)

	// A range over an array of a struct type gives a copy of each element.
	type point struct{ X int }
	pts := [2]point{{1}, {2}}
	for _, v := range pts {
		v.X = 100
	}
	row("a copy of each element", pts)
	for i := range pts {
		pts[i].X = 100
	}
	row("a write by the index", pts)
}

func overMapAndChan() {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := []string{}
	total := 0
	for k, v := range m {
		keys = append(keys, k)
		total += v
	}
	sort.Strings(keys)
	row("the keys", keys)
	row("the total", total)
	n := 0
	for range m {
		n++
	}
	row("the count", n)
	only := []string{}
	for k := range m {
		only = append(only, k)
	}
	sort.Strings(only)
	row("1 variable", only)
	// A break of a range over a map.
	seen := 0
	for range m {
		seen++
		break
	}
	row("a break", seen)
	// A range over a nil map.
	var nilMap map[string]int
	n = 0
	for range nilMap {
		n++
	}
	row("a nil map", n)

	// A channel gives each value until the close.
	ch := make(chan int, 4)
	for i := 1; i <= 3; i++ {
		ch <- i
	}
	close(ch)
	out := []int{}
	for v := range ch {
		out = append(out, v)
	}
	row("a channel", out)
	// A break leaves the values in the channel.
	ch2 := make(chan int, 4)
	for i := 1; i <= 3; i++ {
		ch2 <- i
	}
	close(ch2)
	out = out[:0]
	for v := range ch2 {
		out = append(out, v)
		break
	}
	row("a break of a channel", fmt.Sprint(out, " left=", len(ch2)))
	// A channel that a goroutine feeds.
	ch3 := make(chan string)
	go func() {
		for _, s := range []string{"x", "y"} {
			ch3 <- s
		}
		close(ch3)
	}()
	texts := []string{}
	for v := range ch3 {
		texts = append(texts, v)
	}
	row("a goroutine feeds", texts)
}

func main() {
	fmt.Println("== an integer ==")
	overInt()
	fmt.Println("== a function ==")
	overFunc()
	fmt.Println("== a string ==")
	overString()
	fmt.Println("== an array and a slice ==")
	overArray()
	fmt.Println("== a map and a channel ==")
	overMapAndChan()
}
