//go:build dart

// Package slices holds the API of the standard library package slices.
//
// The compiler cannot read the real source. The sort of that package takes the
// address of a local integer, and only a pointer to a struct or to an array
// stays in the subset. This file holds the same behavior.
//
// Each function that gives an iterator stays out. Such a function needs a range
// loop over a function, and the subset holds no such loop.
package slices

import "cmp"

func Equal[S ~[]E, E comparable](s1, s2 S) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func EqualFunc[S1 ~[]E1, S2 ~[]E2, E1, E2 any](s1 S1, s2 S2, eq func(E1, E2) bool) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if !eq(s1[i], s2[i]) {
			return false
		}
	}
	return true
}

func Compare[S ~[]E, E cmp.Ordered](s1, s2 S) int {
	for i, v1 := range s1 {
		if i >= len(s2) {
			return 1
		}
		if c := cmp.Compare(v1, s2[i]); c != 0 {
			return c
		}
	}
	if len(s1) < len(s2) {
		return -1
	}
	return 0
}

func CompareFunc[S1 ~[]E1, S2 ~[]E2, E1, E2 any](s1 S1, s2 S2, cmp func(E1, E2) int) int {
	for i, v1 := range s1 {
		if i >= len(s2) {
			return 1
		}
		if c := cmp(v1, s2[i]); c != 0 {
			return c
		}
	}
	if len(s1) < len(s2) {
		return -1
	}
	return 0
}

func Index[S ~[]E, E comparable](s S, v E) int {
	for i := range s {
		if s[i] == v {
			return i
		}
	}
	return -1
}

func IndexFunc[S ~[]E, E any](s S, f func(E) bool) int {
	for i := range s {
		if f(s[i]) {
			return i
		}
	}
	return -1
}

func Contains[S ~[]E, E comparable](s S, v E) bool { return Index(s, v) >= 0 }

func ContainsFunc[S ~[]E, E any](s S, f func(E) bool) bool { return IndexFunc(s, f) >= 0 }

func Insert[S ~[]E, E any](s S, i int, v ...E) S {
	out := make(S, 0, len(s)+len(v))
	out = append(out, s[:i]...)
	out = append(out, v...)
	return append(out, s[i:]...)
}

func Delete[S ~[]E, E any](s S, i, j int) S {
	return append(s[:i], s[j:]...)
}

func DeleteFunc[S ~[]E, E any](s S, del func(E) bool) S {
	i := IndexFunc(s, del)
	if i < 0 {
		return s
	}
	for j := i + 1; j < len(s); j++ {
		if !del(s[j]) {
			s[i] = s[j]
			i++
		}
	}
	return s[:i]
}

func Replace[S ~[]E, E any](s S, i, j int, v ...E) S {
	out := make(S, 0, len(s)-(j-i)+len(v))
	out = append(out, s[:i]...)
	out = append(out, v...)
	return append(out, s[j:]...)
}

func Clone[S ~[]E, E any](s S) S {
	if s == nil {
		return nil
	}
	return append(S(nil), s...)
}

func Compact[S ~[]E, E comparable](s S) S {
	if len(s) < 2 {
		return s
	}
	k := 1
	for i := 1; i < len(s); i++ {
		if s[i] != s[k-1] {
			s[k] = s[i]
			k++
		}
	}
	return s[:k]
}

func CompactFunc[S ~[]E, E any](s S, eq func(E, E) bool) S {
	if len(s) < 2 {
		return s
	}
	k := 1
	for i := 1; i < len(s); i++ {
		if !eq(s[i], s[k-1]) {
			s[k] = s[i]
			k++
		}
	}
	return s[:k]
}

func Grow[S ~[]E, E any](s S, n int) S {
	if n < 0 {
		panic("cannot be negative")
	}
	if cap(s)-len(s) >= n {
		return s
	}
	out := make(S, len(s), len(s)+n)
	copy(out, s)
	return out
}

// Clip gives a slice whose capacity is its length. The Dart form of a slice
// holds an offset and a length over 1 array, thus the operation is a reslice.
func Clip[S ~[]E, E any](s S) S { return s[:len(s):len(s)] }

func Reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func Concat[S ~[]E, E any](slices ...S) S {
	n := 0
	for _, s := range slices {
		n += len(s)
	}
	out := make(S, 0, n)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

func Repeat[S ~[]E, E any](x S, count int) S {
	if count < 0 {
		panic("cannot be negative")
	}
	out := make(S, 0, len(x)*count)
	for i := 0; i < count; i++ {
		out = append(out, x...)
	}
	return out
}

func Min[S ~[]E, E cmp.Ordered](x S) E {
	if len(x) < 1 {
		panic("slices.Min: empty list")
	}
	out := x[0]
	for _, v := range x[1:] {
		out = cmp.Or(minOne(out, v), out)
	}
	return out
}

// minOne gives the smaller value. A NaN gives NaN, exactly as cmp.Less does.
func minOne[E cmp.Ordered](a, b E) E {
	if cmp.Less(b, a) {
		return b
	}
	if isNaN(b) {
		return b
	}
	return a
}

func MinFunc[S ~[]E, E any](x S, cmp func(a, b E) int) E {
	if len(x) < 1 {
		panic("slices.MinFunc: empty list")
	}
	out := x[0]
	for _, v := range x[1:] {
		if cmp(v, out) < 0 {
			out = v
		}
	}
	return out
}

func Max[S ~[]E, E cmp.Ordered](x S) E {
	if len(x) < 1 {
		panic("slices.Max: empty list")
	}
	out := x[0]
	for _, v := range x[1:] {
		if cmp.Less(out, v) || isNaN(v) {
			out = v
		}
	}
	return out
}

func MaxFunc[S ~[]E, E any](x S, cmp func(a, b E) int) E {
	if len(x) < 1 {
		panic("slices.MaxFunc: empty list")
	}
	out := x[0]
	for _, v := range x[1:] {
		if cmp(v, out) > 0 {
			out = v
		}
	}
	return out
}

// isNaN reports whether a value is not equal to itself. Only a float holds such
// a value.
func isNaN[E cmp.Ordered](v E) bool { return v != v }

func IsSorted[S ~[]E, E cmp.Ordered](x S) bool {
	for i := len(x) - 1; i > 0; i-- {
		if cmp.Less(x[i], x[i-1]) {
			return false
		}
	}
	return true
}

func IsSortedFunc[S ~[]E, E any](x S, cmp func(a, b E) int) bool {
	for i := len(x) - 1; i > 0; i-- {
		if cmp(x[i], x[i-1]) < 0 {
			return false
		}
	}
	return true
}

func BinarySearch[S ~[]E, E cmp.Ordered](x S, target E) (int, bool) {
	i, j := 0, len(x)
	for i < j {
		h := int(uint(i+j) >> 1)
		if cmp.Less(x[h], target) {
			i = h + 1
			continue
		}
		j = h
	}
	return i, i < len(x) && (x[i] == target || (isNaN(x[i]) && isNaN(target)))
}

func BinarySearchFunc[S ~[]E, E, T any](x S, target T, cmp func(E, T) int) (int, bool) {
	i, j := 0, len(x)
	for i < j {
		h := int(uint(i+j) >> 1)
		if cmp(x[h], target) < 0 {
			i = h + 1
			continue
		}
		j = h
	}
	return i, i < len(x) && cmp(x[i], target) == 0
}

func Sort[S ~[]E, E cmp.Ordered](x S) {
	SortFunc(x, func(a, b E) int { return cmp.Compare(a, b) })
}

// SortFunc sorts the slice with a merge sort. The sort is stable, and the
// standard library gives no such promise for this function.
//
// A merge sort needs 1 extra buffer of the length of the slice. The pdqsort of
// the standard library needs none, and it takes the address of a local integer.
// Only a pointer to a struct or to an array stays in the subset.
func SortFunc[S ~[]E, E any](x S, cmp func(a, b E) int) {
	if len(x) < 2 {
		return
	}
	buf := make(S, len(x))
	mergeSort(x, buf, cmp)
}

func SortStableFunc[S ~[]E, E any](x S, cmp func(a, b E) int) { SortFunc(x, cmp) }

func mergeSort[S ~[]E, E any](x, buf S, cmp func(a, b E) int) {
	if len(x) < 2 {
		return
	}
	if len(x) < 12 {
		insertionSort(x, cmp)
		return
	}
	mid := len(x) / 2
	mergeSort(x[:mid], buf[:mid], cmp)
	mergeSort(x[mid:], buf[mid:], cmp)
	if cmp(x[mid-1], x[mid]) <= 0 {
		return
	}
	copy(buf, x)
	merge(x, buf[:mid], buf[mid:len(x)], cmp)
}

func merge[S ~[]E, E any](out, left, right S, cmp func(a, b E) int) {
	i, j, k := 0, 0, 0
	for i < len(left) && j < len(right) {
		if cmp(right[j], left[i]) < 0 {
			out[k] = right[j]
			j++
		} else {
			out[k] = left[i]
			i++
		}
		k++
	}
	for i < len(left) {
		out[k] = left[i]
		i++
		k++
	}
	for j < len(right) {
		out[k] = right[j]
		j++
		k++
	}
}

func insertionSort[S ~[]E, E any](x S, cmp func(a, b E) int) {
	for i := 1; i < len(x); i++ {
		for j := i; j > 0 && cmp(x[j], x[j-1]) < 0; j-- {
			x[j], x[j-1] = x[j-1], x[j]
		}
	}
}
