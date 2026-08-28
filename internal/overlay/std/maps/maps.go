//go:build dart

// Package maps holds the API of the standard library package maps.
//
// The compiler cannot read the real source. That source imports the package
// iter, and each function of it gives an iterator. An iterator needs a range
// loop over a function, and the subset holds no such loop.
//
// This file holds each function of the package that gives no iterator. The 4
// functions that give one stay out: All, Keys, Values, Insert, and Collect.
package maps

func Equal[M1, M2 ~map[K]V, K, V comparable](m1 M1, m2 M2) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok || v1 != v2 {
			return false
		}
	}
	return true
}

func EqualFunc[M1 ~map[K]V1, M2 ~map[K]V2, K comparable, V1, V2 any](m1 M1, m2 M2, eq func(V1, V2) bool) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok || !eq(v1, v2) {
			return false
		}
	}
	return true
}

func Clone[M ~map[K]V, K comparable, V any](m M) M {
	if m == nil {
		return nil
	}
	out := make(M, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func Copy[M1 ~map[K]V, M2 ~map[K]V, K comparable, V any](dst M1, src M2) {
	for k, v := range src {
		dst[k] = v
	}
}

func DeleteFunc[M ~map[K]V, K comparable, V any](m M, del func(K, V) bool) {
	for k, v := range m {
		if del(k, v) {
			delete(m, k)
		}
	}
}
