//go:build dart

// Package godebug reads a setting of the GODEBUG variable.
//
// The Dart runtime reads no such variable, thus each setting holds its default
// and Value gives an empty text. docs/parity.md holds that gap.
package godebug

// A Setting names 1 GODEBUG setting.
type Setting struct {
	name string
}

// New gives the setting of the name.
func New(name string) *Setting { return &Setting{name: name} }

// Name gives the name of the setting.
func (s *Setting) Name() string { return s.name }

// Value gives the text of the setting. It is always empty, thus each caller
// takes the default.
func (s *Setting) Value() string { return "" }

// IncNonDefault counts a use that differs from the default. It performs no
// operation, because no setting differs.
func (s *Setting) IncNonDefault() {}
