// The program proves the channel rules of Go: a buffered channel holds values
// and an unbuffered channel hands each value over, a closed channel gives the
// zero value at once, the 2 result form reports the close, a nil channel
// never becomes ready, a select with a default never waits, a range over a
// channel ends at the close, and len and cap report the buffer.
//
// The program never lets 2 cases of 1 select become ready at the same time.
// Go takes a random one there, and the Dart output takes the first one in
// source order.
package main

import (
	"fmt"
	"sort"
	"time"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

func buffered() {
	ch := make(chan int, 3)
	row("len and cap of an empty one", fmt.Sprint(len(ch), " ", cap(ch)))
	ch <- 1
	row("after 1 write", fmt.Sprint(len(ch), " ", cap(ch)))
	ch <- 2
	ch <- 3
	row("full", fmt.Sprint(len(ch), " ", cap(ch)))
	row("first read", <-ch)
	row("after 1 read", fmt.Sprint(len(ch), " ", cap(ch)))
	row("second read", <-ch)
	row("third read", <-ch)
	row("empty again", len(ch))

	// The order of a buffered channel is the order of the writes.
	q := make(chan string, 5)
	for _, s := range []string{"a", "b", "c"} {
		q <- s
	}
	out := []string{}
	for len(q) > 0 {
		out = append(out, <-q)
	}
	row("the order", out)

	// A channel of a struct type and of a pointer type.
	type item struct{ N int }
	sc := make(chan item, 1)
	sc <- item{5}
	row("a struct value", <-sc)
	pc := make(chan *item, 1)
	pc <- &item{6}
	row("a pointer value", *<-pc)
	// A channel of a channel.
	cc := make(chan chan int, 1)
	inner := make(chan int, 1)
	inner <- 7
	cc <- inner
	row("a channel of a channel", <-<-cc)
	// A channel of a zero size type.
	ec := make(chan struct{}, 1)
	ec <- struct{}{}
	row("a channel of struct{}", len(ec))
	<-ec
	row("after the read", len(ec))
}

func unbuffered() {
	ch := make(chan int)
	row("len and cap", fmt.Sprint(len(ch), " ", cap(ch)))
	done := make(chan bool)
	go func() {
		for i := 1; i <= 3; i++ {
			ch <- i
		}
		close(ch)
		done <- true
	}()
	out := []int{}
	for v := range ch {
		out = append(out, v)
	}
	<-done
	row("each value", out)

	// A hand over in the other direction.
	req := make(chan int)
	ans := make(chan int)
	go func() {
		for v := range req {
			ans <- v * v
		}
		close(ans)
	}()
	squares := []int{}
	for i := 1; i <= 4; i++ {
		req <- i
		squares = append(squares, <-ans)
	}
	close(req)
	row("the squares", squares)
	row("after the close", <-ans)
}

func closed() {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	close(ch)
	row("len after the close", len(ch))
	row("first read", <-ch)
	v, ok := <-ch
	row("2 result of a value", fmt.Sprint(v, " ", ok))
	v, ok = <-ch
	row("2 result of the close", fmt.Sprint(v, " ", ok))
	v, ok = <-ch
	row("2 result again", fmt.Sprint(v, " ", ok))
	row("1 result form", <-ch)
	row("len at the end", len(ch))

	// The zero value of each element type.
	sc := make(chan string)
	close(sc)
	s, ok := <-sc
	row("a text", fmt.Sprint(fmt.Sprintf("%q", s), " ", ok))
	type item struct{ N int }
	ic := make(chan item)
	close(ic)
	i, ok := <-ic
	row("a struct", fmt.Sprint(i, " ", ok))
	pc := make(chan *item)
	close(pc)
	p, ok := <-pc
	row("a pointer", fmt.Sprint(p == nil, " ", ok))
	fc := make(chan []int)
	close(fc)
	f, ok := <-fc
	row("a slice", fmt.Sprint(f == nil, " ", ok))

	// A range over a closed channel ends at once.
	ec := make(chan int)
	close(ec)
	n := 0
	for range ec {
		n++
	}
	row("range over a closed one", n)

	// A range over a channel that holds values and then closes.
	q := make(chan int, 3)
	q <- 1
	q <- 2
	close(q)
	out := []int{}
	for v := range q {
		out = append(out, v)
	}
	row("range to the close", out)
}

func selects() {
	// A select with a default and no ready case.
	empty := make(chan int)
	got := "none"
	select {
	case v := <-empty:
		got = fmt.Sprint("read ", v)
	default:
		got = "default"
	}
	row("no ready case", got)

	// A select with a default and 1 ready case.
	ready := make(chan int, 1)
	ready <- 5
	select {
	case v := <-ready:
		got = fmt.Sprint("read ", v)
	default:
		got = "default"
	}
	row("1 ready case", got)

	// A select of a write with a default.
	full := make(chan int, 1)
	full <- 1
	select {
	case full <- 2:
		got = "wrote"
	default:
		got = "the buffer is full"
	}
	row("a full buffer", got)
	room := make(chan int, 2)
	room <- 1
	select {
	case room <- 2:
		got = "wrote"
	default:
		got = "full"
	}
	row("room in the buffer", fmt.Sprint(got, " len=", len(room)))

	// A nil channel never becomes ready.
	var nilCh chan int
	select {
	case v := <-nilCh:
		got = fmt.Sprint("read ", v)
	default:
		got = "a nil channel is never ready"
	}
	row("a nil read", got)
	select {
	case nilCh <- 1:
		got = "wrote"
	default:
		got = "a nil write is never ready"
	}
	row("a nil write", got)

	// A nil channel beside a ready case gives the ready case.
	other := make(chan int, 1)
	other <- 9
	select {
	case v := <-nilCh:
		got = fmt.Sprint("nil ", v)
	case v := <-other:
		got = fmt.Sprint("other ", v)
	}
	row("nil against a ready one", got)

	// A select that waits for 1 case that a goroutine makes ready.
	slow := make(chan int)
	go func() { slow <- 11 }()
	select {
	case v := <-slow:
		got = fmt.Sprint("slow ", v)
	case <-nilCh:
		got = "nil"
	}
	row("waits for a goroutine", got)

	// A select of a close.
	toClose := make(chan int)
	go func() { close(toClose) }()
	select {
	case v, ok := <-toClose:
		got = fmt.Sprint(v, " ", ok)
	}
	row("a close in a select", got)

	// A select with no case and a default.
	select {
	default:
		got = "only the default"
	}
	row("only a default", got)

	// A select in a loop that reads until the close.
	feed := make(chan int, 3)
	feed <- 1
	feed <- 2
	close(feed)
	sum := 0
	open := true
	for open {
		select {
		case v, ok := <-feed:
			if !ok {
				open = false
				break
			}
			sum += v
		}
	}
	row("a loop of a select", sum)

	// A send and a receive of 1 select, with 1 ready case only.
	in := make(chan int, 1)
	out := make(chan int)
	in <- 3
	select {
	case v := <-in:
		got = fmt.Sprint("in ", v)
	case out <- 1:
		got = "out"
	}
	row("a mixed select", got)
}

// directions proves a channel of 1 direction.
func directions() {
	ch := make(chan int, 2)
	var send chan<- int = ch
	var read <-chan int = ch
	send <- 1
	send <- 2
	row("through a send channel", fmt.Sprint(len(ch), " ", cap(ch)))
	row("through a read channel", <-read)
	row("len of a read channel", fmt.Sprint(len(read), " ", cap(read)))
	row("a function", drain(ch))
	row("nil compare", ch == nil)
	var nilCh chan int
	row("a nil channel", fmt.Sprint(nilCh == nil, " ", len(nilCh), " ", cap(nilCh)))
	// 2 variables of 1 channel compare equal.
	other := ch
	row("2 names of 1 channel", other == ch)
	row("2 channels", make(chan int) == ch)
}

func drain(ch <-chan int) []int {
	out := []int{}
	for len(ch) > 0 {
		out = append(out, <-ch)
	}
	return out
}

// pipeline proves an order that a chain of channels makes deterministic.
func pipeline() {
	gen := make(chan int)
	sq := make(chan int)
	go func() {
		for i := 1; i <= 5; i++ {
			gen <- i
		}
		close(gen)
	}()
	go func() {
		for v := range gen {
			sq <- v * v
		}
		close(sq)
	}()
	out := []int{}
	for v := range sq {
		out = append(out, v)
	}
	row("the pipeline", out)

	// A fan out and a fan in. The order of the results is not fixed, thus the
	// program takes a sort.
	work := make(chan int, 5)
	res := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		work <- i
	}
	close(work)
	done := make(chan bool)
	for w := 0; w < 3; w++ {
		go func() {
			for v := range work {
				res <- v * 10
			}
			done <- true
		}()
	}
	for w := 0; w < 3; w++ {
		<-done
	}
	close(res)
	got := []int{}
	for v := range res {
		got = append(got, v)
	}
	sort.Ints(got)
	row("the fan out", got)
}

// timers proves a channel of the time package.
func timers() {
	t := time.NewTimer(time.Millisecond)
	<-t.C
	row("a timer fired", true)
	row("Stop after the run", t.Stop())
	tick := time.NewTicker(time.Millisecond)
	n := 0
	for i := 0; i < 3; i++ {
		<-tick.C
		n++
	}
	tick.Stop()
	row("3 ticks", n)
	select {
	case <-time.After(time.Millisecond):
		row("After", "fired")
	}
}

func main() {
	fmt.Println("== a buffered channel ==")
	buffered()
	fmt.Println("== an unbuffered channel ==")
	unbuffered()
	fmt.Println("== a closed channel ==")
	closed()
	fmt.Println("== a select ==")
	selects()
	fmt.Println("== a direction ==")
	directions()
	fmt.Println("== a pipeline ==")
	pipeline()
	fmt.Println("== a timer ==")
	timers()
}
