// Package name holds the rules that map a Go identifier to a Dart identifier.
//
// An unexported Go name gets a `_` prefix, thus the Dart library keeps the Go
// visibility rule. A name that collides with a Dart keyword gets a `$` suffix.
package name

import (
	"go/token"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var keywords = map[string]bool{
	"abstract": true, "as": true, "assert": true, "async": true, "await": true,
	"base": true, "break": true, "case": true, "catch": true, "class": true,
	"const": true, "continue": true, "covariant": true, "default": true,
	"deferred": true, "do": true, "dynamic": true, "else": true, "enum": true,
	"export": true, "extends": true, "extension": true, "external": true,
	"factory": true, "false": true, "final": true, "finally": true, "for": true,
	"Function": true, "get": true, "hide": true, "if": true, "implements": true,
	"import": true, "in": true, "interface": true, "is": true, "late": true,
	"library": true, "mixin": true, "new": true, "null": true, "on": true,
	"operator": true, "part": true, "required": true, "rethrow": true,
	"return": true, "sealed": true, "set": true, "show": true, "static": true,
	"super": true, "switch": true, "sync": true, "this": true, "throw": true,
	"true": true, "try": true, "typedef": true, "var": true, "void": true,
	"when": true, "while": true, "with": true, "yield": true,
}

// reserved holds each Dart name that the emitted code uses. A Go name that is
// equal to one of them takes a `$` suffix, thus it does not hide the Dart name.
var reserved = map[string]bool{
	// The runtime library.
	"Slice": true, "GoString": true, "GoMap": true, "Chan": true,
	"GoFloat32": true, "GoComplex32": true,
	"Iface": true, "TypeDesc": true, "FieldDesc": true, "Dict": true,
	"DeferStack": true, "GoPanic": true, "GoError": true,
	// The frame class of a function that can block holds this member.
	"resume":     true,
	"SelectCase": true, "RecvCase": true, "SendCase": true,
	// The typed data lists.
	"Int8List": true, "Int16List": true, "Int32List": true, "Int64List": true,
	"Uint8List": true, "Uint16List": true, "Uint32List": true,
	"Uint64List": true, "Float32List": true, "Float64List": true,
	// The core library and the async library.
	"Object": true, "List": true, "Map": true, "String": true, "Never": true,
	"Comparable": true, "Future": true, "FutureOr": true,
	// The scalar types of Dart. A Go name of this shape is always unexported,
	// thus only a local name and an embedded field reach these 4. Go accepts
	// each 1 as an identifier, and Dart reads the type at the same place.
	"int": true, "double": true, "bool": true, "num": true,
}

// Local maps a local variable, a parameter, or a result.
func Local(goName string) string {
	if keywords[goName] || reserved[goName] {
		return goName + "$"
	}
	return ascii(goName)
}

// ascii writes each rune of a Go name that Dart rejects as a hex escape. A
// Dart identifier holds only ASCII letters, ASCII digits, `_`, and `$`.
func ascii(goName string) string {
	plain := true
	for _, r := range goName {
		if r >= 128 {
			plain = false
			break
		}
	}
	if plain {
		return goName
	}
	out := make([]byte, 0, len(goName)+8)
	for _, r := range goName {
		if r < 128 {
			out = append(out, byte(r))
			continue
		}
		out = append(out, '$', 'u')
		out = strconv.AppendInt(out, int64(r), 16)
	}
	return string(out)
}

// Exported reports whether Go exports the name.
func Exported(goName string) bool {
	r, _ := utf8.DecodeRuneInString(goName)
	return unicode.IsUpper(r)
}

// Type maps the name of a named type. The Dart class takes a `$` suffix,
// because an embedded struct field carries the name of its own type.
//
// The name of an unexported type takes no `_` prefix. An exported variable can
// hold a type that Go does not export, and a second package then names that
// type. A Dart library reaches no name of another library that starts with an
// underscore.
func Type(goName string) string {
	if !token.IsExported(goName) {
		return goName + "$"
	}
	return Package(goName) + "$"
}

// Package maps a name at package level: a function, a variable, a constant, a
// type, or a struct field.
func Package(goName string) string {
	if !token.IsExported(goName) {
		return "_" + ascii(goName)
	}
	return Public(goName)
}

// Public maps a name that another Dart library must reach, whether or not Go
// exports it. An embedded field of an unexported type takes this form, because
// Go promotes each member of that type across a package boundary.
func Public(goName string) string {
	if keywords[goName] || reserved[goName] {
		return goName + "$"
	}
	return ascii(goName)
}
