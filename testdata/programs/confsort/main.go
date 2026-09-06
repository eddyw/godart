// The program proves the sort operations of Go: sort.Slice, sort.SliceStable,
// sort.Sort, slices.Sort, slices.SortFunc, slices.SortStableFunc,
// slices.BinarySearch, and the key operations of maps.
//
// An unstable sort leaves 2 equal elements in an order that the algorithm
// chooses, and Go names no such order. Thus the program reads the order of 2
// equal elements only after a stable sort.
package main

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-32s %v\n", label, value)
}

type person struct {
	Name string
	Age  int
}

func people() []person {
	return []person{
		{"eve", 30}, {"adam", 25}, {"cain", 30}, {"abel", 25},
		{"seth", 40}, {"noah", 25},
	}
}

func names(list []person) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = fmt.Sprint(p.Name, ":", p.Age)
	}
	return out
}

func sortSlice() {
	// A sort by a key with no equal pair gives 1 answer.
	list := people()
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	row("by name", names(list))
	row("IsSorted", sort.SliceIsSorted(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	}))

	// A stable sort by the age keeps the order of each equal pair.
	list = people()
	sort.SliceStable(list, func(i, j int) bool { return list[i].Age < list[j].Age })
	row("stable by age", names(list))

	// A second stable sort of the result keeps the first order inside a group.
	sort.SliceStable(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	row("stable by name", names(list))

	// A sort of the numbers.
	nums := []int{5, 2, 9, 1, 7, 3}
	sort.Ints(nums)
	row("sort.Ints", nums)
	texts := []string{"pear", "apple", "fig"}
	sort.Strings(texts)
	row("sort.Strings", texts)
	floats := []float64{2.5, 1.5, 3.5}
	sort.Float64s(floats)
	row("sort.Float64s", floats)
	row("IntsAreSorted", sort.IntsAreSorted(nums))
	row("StringsAreSorted", sort.StringsAreSorted(texts))

	// A sort with an order of its own.
	nums = []int{5, 2, 9, 1, 7, 3}
	sort.Sort(sort.Reverse(sort.IntSlice(nums)))
	row("Reverse", nums)

	// The search operations.
	nums = []int{1, 3, 5, 7, 9}
	row("SearchInts found", sort.SearchInts(nums, 5))
	row("SearchInts absent", sort.SearchInts(nums, 6))
	row("SearchInts below", sort.SearchInts(nums, 0))
	row("SearchInts above", sort.SearchInts(nums, 100))
	row("Search", sort.Search(len(nums), func(i int) bool { return nums[i] >= 7 }))
	texts = []string{"a", "c", "e"}
	row("SearchStrings", sort.SearchStrings(texts, "c"))
	row("SearchStrings absent", sort.SearchStrings(texts, "d"))
	fl := []float64{1.5, 2.5}
	row("SearchFloat64s", sort.SearchFloat64s(fl, 2.5))

	// An empty list and a list of 1 element.
	var none []int
	sort.Ints(none)
	row("an empty list", none)
	one := []int{1}
	sort.Ints(one)
	row("1 element", one)
	// A list that is already sorted.
	ready := []int{1, 2, 3}
	sort.Ints(ready)
	row("already sorted", ready)
	// A list in the reverse order.
	back := []int{3, 2, 1}
	sort.Ints(back)
	row("the reverse order", back)
	// A list of equal values.
	same := []int{2, 2, 2}
	sort.Ints(same)
	row("each value equal", same)
}

// The sortIface types prove sort.Interface.
type byAge []person

func (b byAge) Len() int           { return len(b) }
func (b byAge) Less(i, j int) bool { return b[i].Age < b[j].Age }
func (b byAge) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func sortIface() {
	list := people()
	sort.Stable(byAge(list))
	row("sort.Stable", names(list))
	row("sort.IsSorted", sort.IsSorted(byAge(list)))
	list = people()
	sort.Stable(sort.Reverse(byAge(list)))
	row("Stable and Reverse", names(list))
}

func slicesSort() {
	nums := []int{5, 2, 9, 1, 7, 3}
	slices.Sort(nums)
	row("slices.Sort", nums)
	row("IsSorted", slices.IsSorted(nums))
	texts := []string{"pear", "apple", "fig"}
	slices.Sort(texts)
	row("slices.Sort of texts", texts)

	// SortFunc with a key that holds no equal pair.
	list := people()
	slices.SortFunc(list, func(a, b person) int { return cmp.Compare(a.Name, b.Name) })
	row("SortFunc by name", names(list))

	// SortStableFunc keeps the order of each equal pair.
	list = people()
	slices.SortStableFunc(list, func(a, b person) int { return cmp.Compare(a.Age, b.Age) })
	row("SortStableFunc by age", names(list))

	// A comparison of 2 keys.
	list = people()
	slices.SortFunc(list, func(a, b person) int {
		if n := cmp.Compare(a.Age, b.Age); n != 0 {
			return n
		}
		return cmp.Compare(a.Name, b.Name)
	})
	row("2 keys", names(list))

	// IsSortedFunc and the reverse order.
	row("IsSortedFunc", slices.IsSortedFunc(list, func(a, b person) int {
		if n := cmp.Compare(a.Age, b.Age); n != 0 {
			return n
		}
		return cmp.Compare(a.Name, b.Name)
	}))
	slices.Reverse(nums)
	row("slices.Reverse", nums)

	// BinarySearch of a sorted list.
	sorted := []int{1, 3, 5, 7, 9}
	for _, want := range []int{1, 4, 5, 9, 10, 0} {
		i, found := slices.BinarySearch(sorted, want)
		fmt.Printf("BinarySearch %-4d index=%d found=%v\n", want, i, found)
	}
	// BinarySearchFunc.
	list = people()
	slices.SortFunc(list, func(a, b person) int { return cmp.Compare(a.Name, b.Name) })
	i, found := slices.BinarySearchFunc(list, person{Name: "cain"},
		func(a, b person) int { return cmp.Compare(a.Name, b.Name) })
	row("BinarySearchFunc", fmt.Sprint(i, " ", found))
	i, found = slices.BinarySearchFunc(list, person{Name: "zed"},
		func(a, b person) int { return cmp.Compare(a.Name, b.Name) })
	row("BinarySearchFunc absent", fmt.Sprint(i, " ", found))
	// A search of an empty list.
	var none []int
	i, found = slices.BinarySearch(none, 1)
	row("BinarySearch of nothing", fmt.Sprint(i, " ", found))

	// The other operations of slices.
	row("Contains", slices.Contains(sorted, 5))
	row("Index", slices.Index(sorted, 7))
	row("IndexFunc", slices.IndexFunc(sorted, func(n int) bool { return n > 4 }))
	row("Max", slices.Max(sorted))
	row("Min", slices.Min(sorted))
	row("MaxFunc", slices.MaxFunc(people(), func(a, b person) int {
		return cmp.Compare(a.Age, b.Age)
	}))
	row("Equal", slices.Equal([]int{1, 2}, []int{1, 2}))
	row("Compare", slices.Compare([]int{1, 2}, []int{1, 3}))
	row("Clone", slices.Clone(sorted))
	row("Compact", slices.Compact([]int{1, 1, 2, 2, 3}))
	row("Insert", slices.Insert([]int{1, 4}, 1, 2, 3))
	row("Delete", slices.Delete([]int{1, 2, 3, 4}, 1, 3))
	row("Concat", slices.Concat([]int{1}, []int{2, 3}))
	row("Repeat", slices.Repeat([]int{1, 2}, 2))
	row("Sorted of a Seq", slices.Sorted(slices.Values([]int{3, 1, 2})))
	row("Chunk", chunks(sorted, 2))
}

func chunks(list []int, n int) [][]int {
	out := [][]int{}
	for c := range slices.Chunk(list, n) {
		out = append(out, c)
	}
	return out
}

func mapKeys() {
	m := map[string]int{"pear": 3, "apple": 1, "fig": 2}
	keys := slices.Sorted(maps.Keys(m))
	row("sorted keys", keys)
	values := slices.Sorted(maps.Values(m))
	row("sorted values", values)
	// A collection of the keys in the plain way.
	list := make([]string, 0, len(m))
	for k := range m {
		list = append(list, k)
	}
	sort.Strings(list)
	row("the plain way", list)
	// A sort of the keys by the value.
	byValue := slices.Collect(maps.Keys(m))
	slices.SortFunc(byValue, func(a, b string) int { return cmp.Compare(m[a], m[b]) })
	row("by the value", byValue)
	// maps.Clone and maps.Equal.
	clone := maps.Clone(m)
	row("Clone equal", maps.Equal(m, clone))
	clone["new"] = 9
	row("after a write", fmt.Sprint(len(m), " ", len(clone)))
	// maps.Insert and maps.Collect.
	other := maps.Collect(maps.All(m))
	row("Collect", len(other))
	// A count of each word.
	counts := map[string]int{}
	for _, w := range strings.Fields("a b a c b a") {
		counts[w]++
	}
	pairs := make([]string, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, fmt.Sprint(k, "=", v))
	}
	sort.Strings(pairs)
	row("the counts", pairs)
	// A sort of a list of pairs by 2 keys.
	slices.SortFunc(pairs, func(a, b string) int { return cmp.Compare(b, a) })
	row("the reverse order", pairs)
}

// stability proves that a stable sort keeps the order of each equal pair, and
// that a second stable sort keeps the first order inside a group.
func stability() {
	type item struct {
		Key   int
		Order int
	}
	list := []item{}
	for i := 0; i < 12; i++ {
		list = append(list, item{Key: i % 3, Order: i})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Key < list[j].Key })
	out := []string{}
	for _, it := range list {
		out = append(out, fmt.Sprint(it.Key, "/", it.Order))
	}
	row("SliceStable", out)

	list = list[:0]
	for i := 0; i < 12; i++ {
		list = append(list, item{Key: i % 4, Order: i})
	}
	slices.SortStableFunc(list, func(a, b item) int { return cmp.Compare(a.Key, b.Key) })
	out = out[:0]
	for _, it := range list {
		out = append(out, fmt.Sprint(it.Key, "/", it.Order))
	}
	row("SortStableFunc", out)

	// sort.Stable of an interface.
	plain := []int{}
	for i := 0; i < 20; i++ {
		plain = append(plain, i%5)
	}
	sort.Stable(sort.IntSlice(plain))
	row("Stable of ints", plain)
}

func main() {
	fmt.Println("== sort.Slice ==")
	sortSlice()
	fmt.Println("== sort.Interface ==")
	sortIface()
	fmt.Println("== slices ==")
	slicesSort()
	fmt.Println("== maps ==")
	mapKeys()
	fmt.Println("== the stability ==")
	stability()
}
