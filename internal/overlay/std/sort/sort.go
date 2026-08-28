//go:build dart

// Package sort orders a slice.
package sort

import "syscall/dart"

// Slice orders x with the less function. The order of 2 equal elements is not
// stable.
func Slice(x any, less func(i, j int) bool) {
	quick(x, less, 0, dart.SliceLen(x)-1)
}

// SliceStable orders x with the less function, and it keeps the order of 2
// equal elements.
func SliceStable(x any, less func(i, j int) bool) {
	n := dart.SliceLen(x)
	for i := 1; i < n; i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			dart.SliceSwap(x, j, j-1)
		}
	}
}

func quick(x any, less func(i, j int) bool, low, high int) {
	for low < high {
		p := partition(x, less, low, high)
		if p-low < high-p {
			quick(x, less, low, p-1)
			low = p + 1
			continue
		}
		quick(x, less, p+1, high)
		high = p - 1
	}
}

func partition(x any, less func(i, j int) bool, low, high int) int {
	i := low
	for j := low; j < high; j++ {
		if less(j, high) {
			dart.SliceSwap(x, i, j)
			i++
		}
	}
	dart.SliceSwap(x, i, high)
	return i
}

// Interface holds the 3 operations that Sort needs.
type Interface interface {
	Len() int
	Less(i, j int) bool
	Swap(i, j int)
}

// Sort orders data. The order of 2 equal elements is not stable.
func Sort(data Interface) {
	quickData(data, 0, data.Len()-1)
}

// Stable orders data, and it keeps the order of 2 equal elements.
func Stable(data Interface) {
	for i := 1; i < data.Len(); i++ {
		for j := i; j > 0 && data.Less(j, j-1); j-- {
			data.Swap(j, j-1)
		}
	}
}

func quickData(data Interface, low, high int) {
	if low >= high {
		return
	}
	middle := partitionData(data, low, high)
	quickData(data, low, middle-1)
	quickData(data, middle+1, high)
}

func partitionData(data Interface, low, high int) int {
	i := low
	for j := low; j < high; j++ {
		if data.Less(j, high) {
			data.Swap(i, j)
			i++
		}
	}
	data.Swap(i, high)
	return i
}

// Ints orders a slice of int.
func Ints(x []int) {
	Slice(x, func(i, j int) bool { return x[i] < x[j] })
}

// Strings orders a slice of string.
func Strings(x []string) {
	Slice(x, func(i, j int) bool { return x[i] < x[j] })
}

// Float64s orders a slice of float64.
func Float64s(x []float64) {
	Slice(x, func(i, j int) bool { return x[i] < x[j] })
}

// IntsAreSorted reports whether x is in order.
func IntsAreSorted(x []int) bool {
	for i := 1; i < len(x); i++ {
		if x[i] < x[i-1] {
			return false
		}
	}
	return true
}

// Search gives the smallest i in [0, n) where f(i) is true, or n.
func Search(n int, f func(int) bool) int {
	low, high := 0, n
	for low < high {
		mid := (low + high) / 2
		if f(mid) {
			high = mid
			continue
		}
		low = mid + 1
	}
	return low
}

// SearchInts gives the position of v in the ordered slice x.
func SearchInts(x []int, v int) int {
	return Search(len(x), func(i int) bool { return x[i] >= v })
}

// Find gives the smallest i in [0, n) where cmp(i) is not positive, and it
// reports whether cmp(i) is 0. The function cmp compares the target with the
// element i.
func Find(n int, cmp func(int) int) (int, bool) {
	i := Search(n, func(i int) bool { return cmp(i) <= 0 })
	return i, i < n && cmp(i) == 0
}
