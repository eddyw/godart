package ui

import (
	"errors"
	"strconv"
	"strings"
)

// This file holds the call that runs from Go into the host and gives an
// answer.
//
// The bridge of the other files runs 1 way: the host calls the Go build, and
// the Go tree carries no call of Dart. A date picker, a share sheet, a dialog
// with an answer, and each plugin of a platform channel need the other
// direction.
//
// The order of 1 ask:
//
//  1. A goroutine calls Call. The call writes a request and it parks on a
//     channel.
//  2. The host reads TakeCalls after the scheduler stops, and it runs the Dart
//     work of the request.
//  3. The host calls Answer with the result. The channel wakes the goroutine,
//     and the next pump runs it.
//
// Call must run in a goroutine. A build gives its tree at once, thus a build
// that calls Call blocks the frame. The verifier of the compiler already
// rejects a build that reads a channel, and Call reads 1.

// hostCall holds 1 request that waits for the host.
type hostCall struct {
	id   int
	name string
	arg  string
	back chan callReply
}

type callReply struct {
	value string
	err   string
}

var (
	callLast    int
	callPending []hostCall
	callOpen    map[int]chan callReply
)

// Call sends a request to the host and it waits for the answer.
//
// The name tells the host which call it must run. The argument and the answer
// each hold a text of the application, and a text of JSON carries a value that
// holds more than 1 field.
func Call(name, arg string) (string, error) {
	callLast++
	id := callLast
	back := make(chan callReply, 1)
	if callOpen == nil {
		callOpen = map[int]chan callReply{}
	}
	callOpen[id] = back
	callPending = append(callPending, hostCall{id: id, name: name, arg: arg, back: back})
	r := <-back
	if r.err != "" {
		return "", errors.New(r.err)
	}
	return r.value, nil
}

// TakeCalls gives each request that stands and it clears the queue. The host
// calls it after each run of the scheduler.
//
// The answer is a JSON array of objects that hold the keys id, name, and arg.
// An empty text means that no request stands.
func TakeCalls() string {
	if len(callPending) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, a := range callPending {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"id":`)
		b.WriteString(strconv.Itoa(a.id))
		b.WriteString(`,"name":`)
		writeJSONText(&b, a.name)
		b.WriteString(`,"arg":`)
		writeJSONText(&b, a.arg)
		b.WriteByte('}')
	}
	b.WriteByte(']')
	callPending = nil
	return b.String()
}

// Answer writes the answer of 1 request. The host calls it, and an empty error
// means that the call gave the value.
func Answer(id int, value, err string) {
	back, ok := callOpen[id]
	if !ok {
		return
	}
	delete(callOpen, id)
	back <- callReply{value: value, err: err}
}

// resetCalls drops each request. Reset calls it.
func resetCalls() {
	callLast = 0
	callPending = nil
	callOpen = nil
}

// writeJSONText writes 1 JSON string. encoding/json would hold the same
// answer, and this writer keeps the package free of that dependency.
func writeJSONText(b *strings.Builder, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\r':
			b.WriteString(`\r`)
		case c == '\t':
			b.WriteString(`\t`)
		case c < 0x20:
			b.WriteString(`\u00`)
			const hex = "0123456789abcdef"
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xf])
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
}
