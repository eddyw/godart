//go:build dart

// Package godebug reads library settings from the GODEBUG variable.
package godebug

import (
	"internal/bisect"
	"os"
	"strings"
	"syscall/dart"
)

var initialEnv, _ = dart.Getenv("GODEBUG")

type cachedValue struct {
	version int
	matcher *bisect.Matcher
}

var values = map[string]cachedValue{}

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
	version := dart.RuntimeGODEBUGVersion()
	if name == "fips140" {
		env = initialEnv
		version = 0
	}
	env = dart.DefaultGODEBUG() + "," + env
	for env != "" {
		i := strings.LastIndexByte(env, ',')
		key, value, found := strings.Cut(env[i+1:], "=")
		if found && key == name {
			text, pattern, _ := strings.Cut(value, "#")
			cached, ok := values[name]
			if !ok || cached.version != version {
				cached.version = version
				cached.matcher, _ = bisect.New(pattern)
				values[name] = cached
			}
			matcher := cached.matcher
			if matcher != nil && !matcher.Stack(os.Stderr) {
				return ""
			}
			return text
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
