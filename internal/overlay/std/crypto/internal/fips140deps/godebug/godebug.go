//go:build dart

// This file patches the file godebug.go of the package
// crypto/internal/fips140deps/godebug.
//
// The real file converts a pointer to internal/godebug.Setting into a pointer
// to the local type. The subset rejects a pointer conversion between 2 defined
// struct types, thus the local type holds the setting in a field.
package godebug

import (
	"internal/godebug"
)

// Setting names 1 setting of GODEBUG.
type Setting struct {
	inner *godebug.Setting
}

// New gives the setting of the name.
func New(name string) *Setting {
	return &Setting{inner: godebug.New(name)}
}

// Value gives the text of the setting.
func (s *Setting) Value() string {
	v := s.inner.Value()
	if s.inner.Name() == "fips140" && v != "" && v != "off" {
		panic("crypto/fips140: Dart does not support FIPS 140-3 mode")
	}
	return v
}

// Value gives the text of the setting of the name.
func Value(name string) string {
	return New(name).Value()
}
