package ir

import (
	"go/ast"
	"strings"
)

// DartDirective is the marker of an external Dart declaration. A Go function
// declaration with no body that carries it stands for 1 Dart symbol.
//
// The form is `//gart:dart <library> <name>`, for example:
//
//	//gart:dart dart:math max
//	func dartMax(a, b float64) float64
//
// A third field `blocking` marks a symbol that gives up control. The Dart form
// of such a symbol waits for a future of the event loop, thus each call of it
// is a state boundary.
const DartDirective = "//gart:dart"

// DartBlockMark is the third field of a directive that names a symbol which
// gives up control.
const DartBlockMark = "blocking"

// DartRef names the Dart library and the Dart symbol of a declaration.
type DartRef struct {
	Lib  string
	Name string

	// Blocks marks a symbol that gives up control.
	Blocks bool
}

// DartMarker reads the //gart:dart directive of a declaration. It gives nil
// when the declaration carries none, and it gives nil when the directive holds
// no library and no name.
func DartMarker(doc *ast.CommentGroup) *DartRef {
	if doc == nil {
		return nil
	}
	for _, c := range doc.List {
		rest, ok := strings.CutPrefix(c.Text, DartDirective)
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		switch len(fields) {
		case 2:
			return &DartRef{Lib: fields[0], Name: fields[1]}
		case 3:
			if fields[2] != DartBlockMark {
				continue
			}
			return &DartRef{Lib: fields[0], Name: fields[1], Blocks: true}
		}
	}
	return nil
}
