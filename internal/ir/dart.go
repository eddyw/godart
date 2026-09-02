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
const DartDirective = "//gart:dart"

// DartRef names the Dart library and the Dart symbol of a declaration.
type DartRef struct {
	Lib  string
	Name string
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
		if len(fields) != 2 {
			continue
		}
		return &DartRef{Lib: fields[0], Name: fields[1]}
	}
	return nil
}
