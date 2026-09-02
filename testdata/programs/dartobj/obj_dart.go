//go:build dart

package main

import "syscall/dart"

// object names the opaque Dart value. syscall/dart carries //go:build dart,
// thus obj_go.go holds the alias of the go command.
type object = dart.Object

//gart:dart dart:convert jsonDecode
func decode(text string) object

//gart:dart dart:convert jsonEncode
func encode(value object) string
