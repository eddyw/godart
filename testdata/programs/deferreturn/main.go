// The program proves the order of a return statement and of the deferred
// calls of the same function.
//
// Go reads each value of the return statement first, and the deferred calls
// run after that. A deferred call that changes a variable which the return
// expression reads thus cannot change the result. A named result reads the
// other order: the return statement writes it, and a deferred call still
// changes it.
//
// A function that gives up control becomes a state machine of stage 4, and
// that machine writes its own return statement. Each case of this program
// stands 2 times, in a plain form and in a form that blocks, thus the 2 paths
// of the compiler answer the same.
//
// jsontext.AppendFormat holds the shape of the first case: the return reads
// the buffer of the encoder, and the deferred call gives the encoder back to
// a pool that drops the buffer.
package main

import "fmt"

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

// box holds the buffer that a deferred call drops.
type box struct {
	buf   []byte
	drops int
}

var g = &box{}

// reset gives each case the same start.
func reset() { g.buf = make([]byte, 0, 16); g.drops = 0 }

// drop empties the buffer, exactly as the pool of jsontext does.
func drop() {
	g.buf = nil
	g.drops++
}

// read gives the text of the buffer.
func read() string { return string(g.buf) }

// pair gives 2 values that both read the buffer.
func pair() (string, int) { return string(g.buf), len(g.buf) }

// fill writes the text into the buffer.
func fill(text string) { g.buf = append(g.buf[:0], text...) }

// block gives up control. A call of it makes the caller a state machine.
func block() {
	ch := make(chan int, 1)
	ch <- 1
	<-ch
}

// 1. A return that reads a variable which a deferred call empties.

func plainSlice(text string) []byte {
	defer drop()
	fill(text)
	return append([]byte(nil), g.buf...)
}

func blockSlice(text string) []byte {
	defer drop()
	block()
	fill(text)
	return append([]byte(nil), g.buf...)
}

// 2. A return of a call, thus the value comes from another frame.

func plainCall(text string) string {
	defer drop()
	fill(text)
	return read()
}

func blockCall(text string) string {
	defer drop()
	block()
	fill(text)
	return read()
}

// 3. A return of a call that itself gives up control. The value of the return
// thus crosses a suspension, and the deferred call still runs after it.

func readSlow() string {
	block()
	return string(g.buf)
}

func plainSlow(text string) string {
	defer drop()
	fill(text)
	return readSlow()
}

// 4. A return inside a loop.

func plainLoop(text string) string {
	defer drop()
	for i := 0; i < 3; i++ {
		fill(text)
		g.buf = append(g.buf, byte('0'+i))
		if i == 2 {
			return read()
		}
	}
	return "none"
}

func blockLoop(text string) string {
	defer drop()
	for i := 0; i < 3; i++ {
		block()
		fill(text)
		g.buf = append(g.buf, byte('0'+i))
		if i == 2 {
			return read()
		}
	}
	return "none"
}

// 5. A return inside a select. The plain form takes the default arm, and the
// form that blocks waits for a channel.

func plainSelect(text string) string {
	defer drop()
	idle := make(chan int)
	select {
	case <-idle:
		return "idle"
	default:
		fill(text)
		return read()
	}
}

func blockSelect(text string) string {
	defer drop()
	ready := make(chan int, 1)
	idle := make(chan int)
	ready <- 1
	select {
	case <-ready:
		fill(text)
		return read()
	case <-idle:
		return "idle"
	}
}

// 6. A return of 2 values, and a return that forwards a call of 2 values.

func plainTwo(text string) (string, int) {
	defer drop()
	fill(text)
	return read(), len(g.buf)
}

func blockTwo(text string) (string, int) {
	defer drop()
	block()
	fill(text)
	return read(), len(g.buf)
}

func plainForward(text string) (string, int) {
	defer drop()
	fill(text)
	return pair()
}

func blockForward(text string) (string, int) {
	defer drop()
	block()
	fill(text)
	return pair()
}

// 7. A named result. The return statement writes it, and the deferred call
// changes it after that.

func plainNamed(text string) (out string) {
	defer func() { out += "+late" }()
	fill(text)
	return read()
}

func blockNamed(text string) (out string) {
	defer func() { out += "+late" }()
	block()
	fill(text)
	return read()
}

// 8. A named result whose deferred call also empties the buffer. The result
// thus holds the text of the return and the length after the drop.

func plainNamedDrop(text string) (out string) {
	defer func() { drop(); out += fmt.Sprintf("|%d", len(g.buf)) }()
	fill(text)
	return read()
}

func blockNamedDrop(text string) (out string) {
	defer func() { drop(); out += fmt.Sprintf("|%d", len(g.buf)) }()
	block()
	fill(text)
	return read()
}

// 9. A deferred call that itself gives up control.

func plainSlowDefer(text string) string {
	defer func() { block(); drop() }()
	fill(text)
	return read()
}

func blockSlowDefer(text string) string {
	defer func() { block(); drop() }()
	block()
	fill(text)
	return read()
}

func blockSlowDeferNamed(text string) (out string) {
	defer func() { block(); out += "+late" }()
	block()
	fill(text)
	return read()
}

// 10. Two deferred calls. The reverse order holds, and neither one reaches
// the value of the return.

func plainTwice(text string) string {
	defer drop()
	defer func() { g.buf = append(g.buf, '!') }()
	fill(text)
	return read()
}

func blockTwice(text string) string {
	defer drop()
	defer func() { g.buf = append(g.buf, '!') }()
	block()
	fill(text)
	return read()
}

// 11. A recover of a panic with a named result. The deferred call writes the
// result, thus the caller reads the text of the recover.

func plainRecover(text string) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("recovered %v drops %d", r, g.drops)
		}
	}()
	defer drop()
	fill(text)
	panic(read())
}

func blockRecover(text string) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("recovered %v drops %d", r, g.drops)
		}
	}()
	defer drop()
	block()
	fill(text)
	panic(read())
}

func show(label string, value any) {
	row(label, fmt.Sprintf("%v drops=%d buf=%q", value, g.drops, string(g.buf)))
}

func main() {
	reset()
	show("plain slice", string(plainSlice("world")))
	reset()
	show("block slice", string(blockSlice("world")))

	reset()
	show("plain call", plainCall("hello"))
	reset()
	show("block call", blockCall("hello"))

	reset()
	show("plain slow call", plainSlow("slow"))

	reset()
	show("plain loop", plainLoop("loop"))
	reset()
	show("block loop", blockLoop("loop"))

	reset()
	show("plain select", plainSelect("pick"))
	reset()
	show("block select", blockSelect("pick"))

	reset()
	a, n := plainTwo("two")
	show("plain two", fmt.Sprintf("%s %d", a, n))
	reset()
	a, n = blockTwo("two")
	show("block two", fmt.Sprintf("%s %d", a, n))

	reset()
	a, n = plainForward("fwd")
	show("plain forward", fmt.Sprintf("%s %d", a, n))
	reset()
	a, n = blockForward("fwd")
	show("block forward", fmt.Sprintf("%s %d", a, n))

	reset()
	show("plain named", plainNamed("name"))
	reset()
	show("block named", blockNamed("name"))

	reset()
	show("plain named drop", plainNamedDrop("name"))
	reset()
	show("block named drop", blockNamedDrop("name"))

	reset()
	show("plain slow defer", plainSlowDefer("slowd"))
	reset()
	show("block slow defer", blockSlowDefer("slowd"))
	reset()
	show("block slow defer named", blockSlowDeferNamed("slowd"))

	reset()
	show("plain twice", plainTwice("twice"))
	reset()
	show("block twice", blockTwice("twice"))

	reset()
	show("plain recover", plainRecover("boom"))
	reset()
	show("block recover", blockRecover("boom"))
}
