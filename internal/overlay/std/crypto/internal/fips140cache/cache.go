//go:build dart

// Package fips140cache holds a map that ties the life of a value to the life
// of a key.
//
// The real package reads the package weak, which needs the package unsafe.
// This overlay holds no entry, thus Get calls new at each call. The document of
// Get says that Get "may return the same value", thus a call of new each time
// gives the behavior of Go with no cache.
//
// Cache is an alias of a generic type over the 2 pointer types. A method whose
// parameter names a pointer to a type parameter loses the type in the Dart
// output, and the Dart compiler then rejects the call. The alias moves the
// pointer into the argument of the type, thus each parameter names a plain type
// parameter and the type stands.
package fips140cache

// Cache holds no entry.
type Cache[K, V any] = cache[*K, *V]

type cache[KP, VP any] struct{}

// Get gives the result of new for the key k.
func (c *cache[KP, VP]) Get(k KP, new func() (VP, error), check func(VP) bool) (VP, error) {
	return new()
}
