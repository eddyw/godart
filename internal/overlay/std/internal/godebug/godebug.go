//go:build dart

// Package godebug reads library settings from the GODEBUG variable.
package godebug

import (
	"os"
	"strings"
	"syscall/dart"
)

var initialEnv, _ = dart.Getenv("GODEBUG")

// A Setting names 1 GODEBUG setting.
type Setting struct {
	name string
}

// New gives the setting of the name.
func New(name string) *Setting { return &Setting{name: name} }

// Name gives the name of the setting.
func (s *Setting) Name() string { return strings.TrimPrefix(s.name, "#") }

// Value reads the last occurrence of the setting. Immutable settings retain
// the process-start value.
func (s *Setting) Value() string {
	name := s.Name()
	env := os.Getenv("GODEBUG")
	if name == "fips140" {
		env = initialEnv
	}
	for env != "" {
		i := strings.LastIndexByte(env, ',')
		key, value, found := strings.Cut(env[i+1:], "=")
		if found && key == name {
			return value
		}
		if i < 0 {
			break
		}
		env = env[:i]
	}
	return ""
}

// IncNonDefault has no native runtime counter in Dart.
func (s *Setting) IncNonDefault() {}
