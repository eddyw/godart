// Package name holds the rules that map a Go identifier to a Dart identifier.
//
// An unexported Go name gets a `_` prefix, thus the Dart library keeps the Go
// visibility rule. A name that collides with a Dart keyword gets a `$` suffix.
package name

import (
	"go/token"
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

// Local maps a local variable, a parameter, or a result.
func Local(goName string) string {
	if keywords[goName] {
		return goName + "$"
	}
	return goName
}

// Exported reports whether Go exports the name.
func Exported(goName string) bool {
	r, _ := utf8.DecodeRuneInString(goName)
	return unicode.IsUpper(r)
}

// Type maps the name of a named type. The Dart class takes a `$` suffix,
// because an embedded struct field carries the name of its own type.
func Type(goName string) string {
	return Package(goName) + "$"
}

// Package maps a name at package level: a function, a variable, a constant, a
// type, or a struct field.
func Package(goName string) string {
	if !token.IsExported(goName) {
		return "_" + goName
	}
	if keywords[goName] {
		return goName + "$"
	}
	return goName
}
