// Package bench holds the Go side of the text property measurement. The Dart
// harness of flutter/bench/dart builds a tree here and walks it, thus the pair
// gives the cost of a text property at the boundary between Go and Dart.
//
// The package holds no test of behavior. flutter/ui holds that test.
package bench

import (
	"strconv"

	"gart.dev/ui"
)

// The 3 literals hold 5, 20, and 80 ASCII characters. gart writes
// GoString.fromString at each evaluation of a Go string literal, thus a
// literal text property pays the encode cost of its length at each build.
const (
	lit5  = "Ready"
	lit20 = "Total price in euros"
	lit80 = "The order of 2026-09-02 holds 4 lines, and the driver waits for the answer today"
)

// litUTF8 holds 20 runes in 37 bytes. A decode of such text takes the slow
// path of utf8.decode, and the ASCII text of the same rune count does not.
const litUTF8 = "Цена заказа за товар"

// Model holds the state of 1 case. Setup writes it, and each tree function
// reads it.
type Model struct {
	texts []string
	alt   []string
	sigs  []*ui.Signal[int]
	n     int
	l     int
	utf8  bool
}

var m = &Model{}

// Setup writes the model of 1 case, and it drops the signal graph of the case
// before it. The Dart harness calls it between 2 measurements.
func Setup(n, l int, utf8 bool) {
	ui.Reset()
	texts := make([]string, n)
	alt := make([]string, n)
	for i := range texts {
		texts[i] = fill(l, byte('a'+i%26))
		alt[i] = flip(texts[i])
	}
	sigs := make([]*ui.Signal[int], n)
	for i := range sigs {
		sigs[i] = ui.NewSignal(0)
	}
	m = &Model{texts: texts, alt: alt, sigs: sigs, n: n, l: l, utf8: utf8}
}

// Nodes gives the node count of the tree that the current case builds. The
// Dart harness reports it, thus a reader can check the shape.
func Nodes() int { return m.n + 1 }

// literal gives the Go literal of the current case. The switch keeps the value
// out of a hoist of the Dart compiler, because the result differs per branch.
func literal() string {
	if m.utf8 {
		return litUTF8
	}
	switch m.l {
	case 5:
		return lit5
	case 20:
		return lit20
	}
	return lit80
}

// flip changes the last byte. A counter that counts 1 up changes its text in
// the same place, thus a compare of the 2 reads each byte before it stops.
func flip(s string) string {
	b := []byte(s)
	b[len(b)-1] ^= 1
	return string(b)
}

func fill(l int, c byte) string {
	b := make([]byte, l)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

// TreeHeld gives a tree whose text stands in the model already. The build
// creates no GoString, thus the number holds the structure cost alone.
func TreeHeld() ui.Node {
	kids := make([]ui.Node, m.n)
	for i := range kids {
		kids[i] = ui.Text{Data: m.texts[i]}
	}
	return ui.Column{Children: kids}
}

// TreeLiteral gives a tree whose text comes from a Go literal. This is the
// common shape of a label on a real screen.
func TreeLiteral() ui.Node {
	kids := make([]ui.Node, m.n)
	for i := range kids {
		kids[i] = ui.Text{Data: literal()}
	}
	return ui.Column{Children: kids}
}

// TreeItoa gives a tree whose text comes from a number and a concatenation.
// The measurement of 2026-09-01 holds this shape, thus the 2 numbers compare.
func TreeItoa() ui.Node {
	kids := make([]ui.Node, m.n)
	for i := range kids {
		kids[i] = ui.Text{Data: literal() + strconv.Itoa(i)}
	}
	return ui.Column{Children: kids}
}

// TreeWatch gives a tree where each row stands behind a boundary of its own. A
// write of 1 signal then rebuilds 1 row alone, which is the normal case of the
// framework.
func TreeWatch(wide bool) ui.Node {
	kids := make([]ui.Node, m.n)
	for i := range kids {
		if wide {
			kids[i] = ui.Watch{Build: m.row3(i)}
		} else {
			kids[i] = ui.Watch{Build: m.row1(i)}
		}
	}
	return ui.Column{Children: kids}
}

// row1 gives the smallest boundary: 1 text node.
func (mm *Model) row1(i int) ui.Builder {
	return func() ui.Node {
		mm.sigs[i].Get()
		return ui.Text{Data: literal()}
	}
}

// row3 gives a boundary of a list row: a label, a value, and a note.
func (mm *Model) row3(i int) ui.Builder {
	return func() ui.Node {
		mm.sigs[i].Get()
		return ui.Row{Children: []ui.Node{
			ui.Text{Data: literal()},
			ui.Text{Data: literal()},
			ui.Text{Data: literal()},
		}}
	}
}

// Bump writes 1 signal. It stands for the event handler of a real screen, thus
// it marks 1 boundary and no other.
func Bump(i int) {
	s := mm(i)
	s.Set(s.Peek() + 1)
}

func mm(i int) *ui.Signal[int] { return m.sigs[i%len(m.sigs)] }

// Dirty gives the boundaries that must rebuild, and it clears the set.
func Dirty() []int { return ui.TakeDirty() }

// TreePick gives a tree where the first k nodes take the second text pool. A
// diff against TreePick(0) then finds exactly k changed nodes.
func TreePick(k int) ui.Node {
	kids := make([]ui.Node, m.n)
	for i := range kids {
		if i < k {
			kids[i] = ui.Text{Data: m.alt[i]}
		} else {
			kids[i] = ui.Text{Data: m.texts[i]}
		}
	}
	return ui.Column{Children: kids}
}

// TreePickStyled gives the same tree, and each node sets 1 optional field. The
// build allocates a fresh pointer for that field at each call, thus the
// generated == of the node reads a change that no reader sees.
func TreePickStyled(k int) ui.Node {
	kids := make([]ui.Node, m.n)
	for i := range kids {
		text := m.texts[i]
		if i < k {
			text = m.alt[i]
		}
		kids[i] = ui.Text{Data: text, TextAlign: ui.P("center")}
	}
	return ui.Column{Children: kids}
}
