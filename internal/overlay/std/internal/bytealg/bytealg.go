//go:build dart

// Package bytealg gives the search operations that the standard library reaches
// through internal/bytealg.
//
// The compiler cannot read the real source. That source holds assembly, the
// package unsafe, and a type parameter with the constraint `string | []byte`.
// The 2 terms of that constraint give 2 Dart types.
//
// Each function below holds a plain search. The real package holds a Rabin Karp
// search and an assembly path, and the 2 give the same result.
package bytealg

// MaxLen is the length of a pattern that the assembly path accepts. A value of
// 0 keeps each caller on the portable path.
var MaxLen int = 0

// MaxBruteForce is the length of a text that the assembly path accepts. A value
// of 0 keeps each caller on the portable path.
const MaxBruteForce = 0

// Cutover gives the count of failures that the caller accepts before it changes
// to the Rabin Karp search. This package holds no such search, thus the result
// stays above each count.
func Cutover(n int) int { return (n + 16) / 8 }

func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Compare(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func CompareString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func IndexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

func IndexByteString(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func LastIndexByte(s []byte, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func LastIndexByteString(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func Count(b []byte, c byte) int {
	n := 0
	for i := range b {
		if b[i] == c {
			n++
		}
	}
	return n
}

func CountString(s string, c byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			n++
		}
	}
	return n
}

// Index gives the offset of the first b inside a. The caller holds
// 2 <= len(b) <= MaxLen, and MaxLen is 0, thus no caller reaches this function.
func Index(a, b []byte) int {
	return IndexRabinKarpBytes(a, b)
}

// IndexString gives the offset of the first b inside a. The caller holds
// 2 <= len(b) <= MaxLen, and MaxLen is 0, thus no caller reaches this function.
func IndexString(a, b string) int {
	return IndexRabinKarpString(a, b)
}

// IndexRabinKarp gives the offset of the first sep inside s.
//
// The type parameter holds `string | []byte`, and the 2 terms give 2 Dart types.
// No operation of the subset reads such a parameter, thus the body converts the
// value to an interface and reads the real type from it.
func IndexRabinKarp[T string | []byte](s, sep T) int {
	switch v := any(s).(type) {
	case string:
		return IndexRabinKarpString(v, any(sep).(string))
	case []byte:
		return IndexRabinKarpBytes(v, any(sep).([]byte))
	}
	return -1
}

// LastIndexRabinKarp gives the offset of the last sep inside s.
func LastIndexRabinKarp[T string | []byte](s, sep T) int {
	switch v := any(s).(type) {
	case string:
		return LastIndexRabinKarpString(v, any(sep).(string))
	case []byte:
		return LastIndexRabinKarpBytes(v, any(sep).([]byte))
	}
	return -1
}

// IndexRabinKarpBytes gives the offset of the first sep inside s.
func IndexRabinKarpBytes(s, sep []byte) int {
	if len(sep) == 0 {
		return 0
	}
	if len(sep) > len(s) {
		return -1
	}
	for i := 0; i+len(sep) <= len(s); i++ {
		if Equal(s[i:i+len(sep)], sep) {
			return i
		}
	}
	return -1
}

func IndexRabinKarpString(s, sep string) int {
	if len(sep) == 0 {
		return 0
	}
	if len(sep) > len(s) {
		return -1
	}
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

func LastIndexRabinKarpBytes(s, sep []byte) int {
	if len(sep) == 0 {
		return len(s)
	}
	for i := len(s) - len(sep); i >= 0; i-- {
		if Equal(s[i:i+len(sep)], sep) {
			return i
		}
	}
	return -1
}

func LastIndexRabinKarpString(s, sep string) int {
	if len(sep) == 0 {
		return len(s)
	}
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

// MakeNoZero gives a slice of n bytes. The real function skips the write of the
// zero value, and Dart gives no such control.
func MakeNoZero(n int) []byte { return make([]byte, n) }
