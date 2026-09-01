//go:build dart

package sort

import "syscall/dart"

// Reverse gives data with the reverse order of its Less.
func Reverse(data Interface) Interface { return reverse{data} }

type reverse struct{ Interface }

func (r reverse) Less(i, j int) bool { return r.Interface.Less(j, i) }

// IsSorted reports whether data stands in order.
func IsSorted(data Interface) bool {
	for i := data.Len() - 1; i > 0; i-- {
		if data.Less(i, i-1) {
			return false
		}
	}
	return true
}

// SliceIsSorted reports whether the slice x stands in the order of less.
func SliceIsSorted(x any, less func(i, j int) bool) bool {
	for i := dart.SliceLen(x) - 1; i > 0; i-- {
		if less(i, i-1) {
			return false
		}
	}
	return true
}

// An IntSlice orders a slice of int in the order of Less.
type IntSlice []int

func (x IntSlice) Len() int           { return len(x) }
func (x IntSlice) Less(i, j int) bool { return x[i] < x[j] }
func (x IntSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

// Sort orders the slice.
func (x IntSlice) Sort() { Sort(x) }

// Search gives the index of v in the sorted slice, and it gives len(x) for a
// value that the slice does not hold.
func (x IntSlice) Search(v int) int { return SearchInts(x, v) }

// A StringSlice orders a slice of string in the order of Less.
type StringSlice []string

func (x StringSlice) Len() int           { return len(x) }
func (x StringSlice) Less(i, j int) bool { return x[i] < x[j] }
func (x StringSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

// Sort orders the slice.
func (x StringSlice) Sort() { Sort(x) }

// Search gives the index of v in the sorted slice.
func (x StringSlice) Search(v string) int { return SearchStrings(x, v) }

// A Float64Slice orders a slice of float64 in the order of Less. A value that
// holds no number comes first.
type Float64Slice []float64

func (x Float64Slice) Len() int { return len(x) }

func (x Float64Slice) Less(i, j int) bool {
	return x[i] < x[j] || (isNaN(x[i]) && !isNaN(x[j]))
}

func (x Float64Slice) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// Sort orders the slice.
func (x Float64Slice) Sort() { Sort(x) }

// Search gives the index of v in the sorted slice.
func (x Float64Slice) Search(v float64) int { return SearchFloat64s(x, v) }

// isNaN reports a value that holds no number. A comparison with itself gives
// false for such a value.
func isNaN(f float64) bool { return f != f }

// SearchStrings gives the index of v in the sorted slice x, and it gives len(x)
// for a value that the slice does not hold.
func SearchStrings(x []string, v string) int {
	return Search(len(x), func(i int) bool { return x[i] >= v })
}

// SearchFloat64s gives the index of v in the sorted slice x.
func SearchFloat64s(x []float64, v float64) int {
	return Search(len(x), func(i int) bool { return x[i] >= v })
}

// StringsAreSorted reports whether the slice stands in order.
func StringsAreSorted(x []string) bool {
	for i := len(x) - 1; i > 0; i-- {
		if x[i] < x[i-1] {
			return false
		}
	}
	return true
}

// Float64sAreSorted reports whether the slice stands in order.
func Float64sAreSorted(x []float64) bool {
	return IsSorted(Float64Slice(x))
}
