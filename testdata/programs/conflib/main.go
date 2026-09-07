// The program proves errors, sort, slices, and maps against real Go: each
// function of errors with a wrap chain of 2 verbs, each sort function with a
// search, each function of slices from Go 1.21 to Go 1.24, and each function
// of maps with a sorted form of the result.
package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func row(label string, value any) {
	fmt.Printf("%-30s %v\n", label, value)
}

type codeError struct {
	Code int
	Text string
}

func (e *codeError) Error() string { return fmt.Sprintf("code %d: %s", e.Code, e.Text) }

type valueError struct{ Kind string }

func (e valueError) Error() string { return "kind " + e.Kind }

// isError matches any error whose text starts with the prefix.
type isError struct{ Prefix string }

func (e isError) Error() string { return "prefix " + e.Prefix }

func (e isError) Is(target error) bool {
	return strings.HasPrefix(target.Error(), e.Prefix)
}

// asError converts itself into a codeError on demand.
type asError struct{}

func (asError) Error() string { return "as error" }

func (asError) As(target any) bool {
	if p, ok := target.(**codeError); ok {
		*p = &codeError{Code: 99, Text: "from As"}
		return true
	}
	return false
}

var errBase = errors.New("the base error")

func errorsPkg() {
	e1 := errors.New("first")
	e2 := errors.New("first")
	row("New text", e1)
	row("New identity", e1 == e2)
	row("Is same text", errors.Is(e1, e2))
	row("Is self", errors.Is(e1, e1))
	row("Is nil nil", errors.Is(nil, nil))
	row("Is nil target", errors.Is(e1, nil))
	row("Is nil err", errors.Is(nil, e1))
	row("Unwrap plain", errors.Unwrap(e1) == nil)
	row("Unwrap nil", errors.Unwrap(nil) == nil)
	wrapped := fmt.Errorf("level 1: %w", errBase)
	twice := fmt.Errorf("level 2: %w", wrapped)
	row("wrap text", twice)
	row("Unwrap once", errors.Unwrap(twice))
	row("Unwrap twice", errors.Unwrap(errors.Unwrap(twice)))
	row("Unwrap thrice", errors.Unwrap(errors.Unwrap(errors.Unwrap(twice))) == nil)
	row("Is through 2 levels", errors.Is(twice, errBase))
	row("Is wrong target", errors.Is(twice, e1))
	noWrap := fmt.Errorf("level 1: %v", errBase)
	row("%v does not wrap", errors.Unwrap(noWrap) == nil)
	row("%v Is", errors.Is(noWrap, errBase))
	two := fmt.Errorf("both: %w and %w", errBase, e1)
	row("2 verbs text", two)
	row("2 verbs Is base", errors.Is(two, errBase))
	row("2 verbs Is e1", errors.Is(two, e1))
	row("2 verbs Is other", errors.Is(two, e2))
	row("2 verbs Unwrap nil", errors.Unwrap(two) == nil)
	multi, ok := two.(interface{ Unwrap() []error })
	row("2 verbs Unwrap list", fmt.Sprintf("%v %d", ok, len(multi.Unwrap())))
	nilWrap := fmt.Errorf("nil: %w", nil)
	row("wrap of nil", nilWrap)
	row("wrap of nil Unwrap", errors.Unwrap(nilWrap) == nil)
	row("%w twice of nil", fmt.Errorf("%w %w", nil, errBase))
	var ce *codeError
	chain := fmt.Errorf("outer: %w", &codeError{Code: 7, Text: "inner"})
	row("As pointer type", fmt.Sprintf("%v %d %s", errors.As(chain, &ce), ce.Code, ce.Text))
	row("As found is chain", errors.Unwrap(chain) == ce)
	var ve valueError
	row("As value type absent", errors.As(chain, &ve))
	chain2 := fmt.Errorf("outer: %w", valueError{Kind: "v"})
	row("As value type", fmt.Sprintf("%v %q", errors.As(chain2, &ve), ve.Kind))
	var target *codeError
	row("As nil err", errors.As(nil, &target))
	// The 3 bad targets travel through a variable of type any, because vet
	// rejects the direct form.
	var nilTarget any
	catch("As nil target", func() { errors.As(chain, nilTarget) })
	row("Is via method", errors.Is(fmt.Errorf("w: %w", isError{Prefix: "the base"}), errBase))
	row("Is via method false", errors.Is(isError{Prefix: "zzz"}, errBase))
	pe := &fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}
	deep := fmt.Errorf("layer: %w", fmt.Errorf("layer: %w", pe))
	var gotPE *fs.PathError
	row("As PathError", fmt.Sprintf("%v %s", errors.As(deep, &gotPE), gotPE.Path))
	row("Is ErrNotExist deep", errors.Is(deep, fs.ErrNotExist))
	row("Is io.EOF", errors.Is(fmt.Errorf("read: %w", io.EOF), io.EOF))
	var ne *strconv.NumError
	_, aerr := strconv.Atoi("x")
	row("As NumError", fmt.Sprintf("%v %s", errors.As(fmt.Errorf("w: %w", aerr), &ne), ne.Func))
	row("Join none", errors.Join() == nil)
	row("Join nils", errors.Join(nil, nil) == nil)
	j1 := errors.Join(e1)
	row("Join 1", fmt.Sprintf("%q", j1.Error()))
	j2 := errors.Join(e1, errBase)
	row("Join 2", fmt.Sprintf("%q", j2.Error()))
	j3 := errors.Join(e1, nil, errBase, nil, valueError{Kind: "k"})
	row("Join 3 with nils", fmt.Sprintf("%q", j3.Error()))
	row("Join Is", fmt.Sprintf("%v %v %v", errors.Is(j3, e1), errors.Is(j3, errBase), errors.Is(j3, e2)))
	row("Join As", fmt.Sprintf("%v %q", errors.As(j3, &ve), ve.Kind))
	row("Join Unwrap nil", errors.Unwrap(j3) == nil)
	jm, ok := j3.(interface{ Unwrap() []error })
	row("Join Unwrap list", fmt.Sprintf("%v %d", ok, len(jm.Unwrap())))
	nested := errors.Join(j2, fmt.Errorf("w: %w", pe))
	row("Join nested text", fmt.Sprintf("%q", nested.Error()))
	row("Join nested Is", errors.Is(nested, fs.ErrNotExist))
	row("Join nested As", errors.As(nested, &gotPE))
	row("Join %v", fmt.Sprintf("%v", j2))
	row("Join %+v", fmt.Sprintf("%+v", j2))
	row("ErrUnsupported", errors.ErrUnsupported)
	row("Is ErrUnsupported", errors.Is(fmt.Errorf("op: %w", errors.ErrUnsupported), errors.ErrUnsupported))
	var nilCode *codeError
	var asIface error = nilCode
	row("typed nil is not nil", asIface != nil)
	row("typed nil Is", errors.Is(asIface, nilCode))
	row("%T of wrap", fmt.Sprintf("%T %T %T", wrapped, two, j2))
	row("%T of New", fmt.Sprintf("%T", e1))
}

func catch(label string, f func()) {
	defer func() {
		fmt.Printf("%-30s %v\n", label, recover())
	}()
	f()
}

type person struct {
	Name string
	Age  int
}

type byName []person

func (b byName) Len() int           { return len(b) }
func (b byName) Less(i, j int) bool { return b[i].Name < b[j].Name }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func people() []person {
	return []person{{"eve", 30}, {"adam", 25}, {"cain", 30}, {"abel", 25}, {"seth", 40}, {"noah", 25}, {"enos", 40}}
}

func sortPkg() {
	ints := []int{5, -2, 9, 0, 7, 3, 3, -8}
	sort.Ints(ints)
	row("Ints", ints)
	row("IntsAreSorted", sort.IntsAreSorted(ints))
	row("IntsAreSorted false", sort.IntsAreSorted([]int{2, 1}))
	strs := []string{"pear", "Apple", "fig", "", "apple", "日本", "z"}
	sort.Strings(strs)
	row("Strings", fmt.Sprintf("%q", strs))
	row("StringsAreSorted", sort.StringsAreSorted(strs))
	floats := []float64{2.5, -1.5, 0, 3.5, -0.0, 1e300, -1e300}
	sort.Float64s(floats)
	row("Float64s", floats)
	row("Float64sAreSorted", sort.Float64sAreSorted(floats))
	list := people()
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	row("Slice by name", list)
	row("SliceIsSorted", sort.SliceIsSorted(list, func(i, j int) bool { return list[i].Name < list[j].Name }))
	list = people()
	sort.SliceStable(list, func(i, j int) bool { return list[i].Age < list[j].Age })
	row("SliceStable by age", list)
	sort.SliceStable(list, func(i, j int) bool { return list[i].Age > list[j].Age })
	row("SliceStable reverse", list)
	list = people()
	sort.Sort(byName(list))
	row("Sort Interface", list)
	row("IsSorted", sort.IsSorted(byName(list)))
	sort.Sort(sort.Reverse(byName(list)))
	row("Reverse", list)
	row("IsSorted reversed", sort.IsSorted(byName(list)))
	list = people()
	sort.Stable(byName(list))
	row("Stable Interface", list)
	is := sort.IntSlice{3, 1, 2}
	sort.Sort(is)
	row("IntSlice", fmt.Sprintf("%v len=%d search=%d", is, is.Len(), is.Search(2)))
	ss := sort.StringSlice{"b", "c", "a"}
	ss.Sort()
	row("StringSlice", fmt.Sprintf("%v %d", ss, ss.Search("c")))
	fsl := sort.Float64Slice{2.5, 1.5}
	fsl.Sort()
	row("Float64Slice", fmt.Sprintf("%v %d", fsl, fsl.Search(2)))
	sorted := []int{1, 3, 5, 7, 9, 9, 11}
	for _, want := range []int{0, 1, 4, 9, 11, 12} {
		row(fmt.Sprintf("SearchInts %d", want), sort.SearchInts(sorted, want))
	}
	row("Search func", sort.Search(len(sorted), func(i int) bool { return sorted[i] >= 6 }))
	row("Search none", sort.Search(len(sorted), func(i int) bool { return sorted[i] >= 100 }))
	row("Search all", sort.Search(len(sorted), func(i int) bool { return true }))
	row("Search 0", sort.Search(0, func(i int) bool { return true }))
	row("SearchStrings", sort.SearchStrings([]string{"a", "c", "e"}, "d"))
	row("SearchFloat64s", sort.SearchFloat64s([]float64{1.5, 2.5, 3.5}, 2.5))
	i, found := sort.Find(len(sorted), func(i int) int { return cmp.Compare(7, sorted[i]) })
	row("Find", fmt.Sprint(i, " ", found))
	i, found = sort.Find(len(sorted), func(i int) int { return cmp.Compare(8, sorted[i]) })
	row("Find absent", fmt.Sprint(i, " ", found))
	i, found = sort.Find(0, func(i int) int { return 0 })
	row("Find empty", fmt.Sprint(i, " ", found))
	var none []int
	sort.Ints(none)
	row("Ints nil", none == nil)
	stable := make([]int, 40)
	for i := range stable {
		stable[i] = i % 7
	}
	sort.Stable(sort.IntSlice(stable))
	row("Stable of many", stable)
	large := make([]int, 200)
	for i := range large {
		large[i] = (i * 7919) % 211
	}
	sort.Ints(large)
	row("Ints large sorted", sort.IntsAreSorted(large))
	row("Ints large ends", fmt.Sprint(large[0], " ", large[99], " ", large[199]))
	words := strings.Fields("the quick brown fox jumps over the lazy dog the end")
	sort.SliceStable(words, func(i, j int) bool { return len(words[i]) < len(words[j]) })
	row("SliceStable by length", words)
}

func slicesPkg() {
	nums := []int{3, 1, 4, 1, 5, 9, 2, 6}
	row("Contains", fmt.Sprint(slices.Contains(nums, 5), " ", slices.Contains(nums, 7)))
	row("ContainsFunc", slices.ContainsFunc(nums, func(n int) bool { return n > 8 }))
	row("Index", fmt.Sprint(slices.Index(nums, 1), " ", slices.Index(nums, 7)))
	row("IndexFunc", slices.IndexFunc(nums, func(n int) bool { return n%2 == 0 }))
	row("Equal", fmt.Sprint(slices.Equal(nums, nums), " ", slices.Equal(nums, nums[:3]), " ", slices.Equal([]int{}, nil)))
	row("EqualFunc", slices.EqualFunc([]int{1, 2}, []string{"1", "2"}, func(a int, b string) bool { return strconv.Itoa(a) == b }))
	row("Compare", fmt.Sprint(slices.Compare([]int{1, 2}, []int{1, 3}), " ", slices.Compare([]int{1, 2}, []int{1, 2}), " ", slices.Compare([]int{1, 2, 3}, []int{1, 2}), " ", slices.Compare([]int{}, []int{1})))
	row("CompareFunc", slices.CompareFunc([]int{1, 2}, []string{"1", "3"}, func(a int, b string) int { return cmp.Compare(strconv.Itoa(a), b) }))
	sorted := slices.Clone(nums)
	slices.Sort(sorted)
	row("Sort", sorted)
	row("Clone kept the source", nums)
	row("Clone nil", slices.Clone([]int(nil)) == nil)
	row("IsSorted", fmt.Sprint(slices.IsSorted(sorted), " ", slices.IsSorted(nums)))
	list := people()
	slices.SortFunc(list, func(a, b person) int { return cmp.Compare(a.Name, b.Name) })
	row("SortFunc", list)
	row("IsSortedFunc", slices.IsSortedFunc(list, func(a, b person) int { return cmp.Compare(a.Name, b.Name) }))
	list = people()
	slices.SortStableFunc(list, func(a, b person) int { return cmp.Compare(a.Age, b.Age) })
	row("SortStableFunc", list)
	row("Min Max", fmt.Sprint(slices.Min(nums), " ", slices.Max(nums)))
	row("MinFunc MaxFunc", fmt.Sprint(slices.MinFunc(list, func(a, b person) int { return cmp.Compare(a.Name, b.Name) }), " ", slices.MaxFunc(list, func(a, b person) int { return cmp.Compare(a.Name, b.Name) })))
	catch("Min of empty", func() { _ = slices.Min([]int{}) })
	catch("Max of empty", func() { _ = slices.Max([]int{}) })
	row("Min with NaN", fmt.Sprint(slices.Min([]float64{2, 1}), " ", slices.Max([]float64{-1, -2})))
	for _, want := range []int{0, 1, 4, 9, 10} {
		i, found := slices.BinarySearch(sorted, want)
		row(fmt.Sprintf("BinarySearch %d", want), fmt.Sprint(i, " ", found))
	}
	i, found := slices.BinarySearchFunc(list, 30, func(p person, age int) int { return cmp.Compare(p.Age, age) })
	row("BinarySearchFunc", fmt.Sprint(i, " ", found))
	row("Reverse", func() []int { r := slices.Clone(sorted); slices.Reverse(r); return r }())
	row("Compact", slices.Compact([]int{1, 1, 2, 2, 2, 3, 1, 1}))
	row("CompactFunc", slices.CompactFunc([]string{"a", "A", "b", "B", "b"}, strings.EqualFold))
	row("Insert", slices.Insert([]int{1, 5}, 1, 2, 3, 4))
	row("Insert at end", slices.Insert([]int{1, 2}, 2, 3))
	row("Insert nothing", slices.Insert([]int{1, 2}, 1))
	row("Delete", slices.Delete([]int{1, 2, 3, 4, 5}, 1, 3))
	row("Delete none", slices.Delete([]int{1, 2}, 1, 1))
	row("DeleteFunc", slices.DeleteFunc([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 }))
	row("Replace", slices.Replace([]int{1, 2, 3, 4}, 1, 3, 9, 8, 7))
	row("Replace shrink", slices.Replace([]int{1, 2, 3, 4}, 1, 3))
	row("Concat", slices.Concat([]int{1}, nil, []int{2, 3}, []int{}))
	row("Repeat", slices.Repeat([]string{"a", "b"}, 3))
	row("Repeat 0", len(slices.Repeat([]int{1}, 0)))
	catch("Repeat negative", func() { _ = slices.Repeat([]int{1}, -1) })
	grown := slices.Grow([]int{1, 2}, 10)
	row("Grow", fmt.Sprint(grown, " len=", len(grown), " cap>=12 ", cap(grown) >= 12))
	catch("Grow negative", func() { _ = slices.Grow([]int{1}, -1) })
	clipped := slices.Clip(make([]int, 2, 10))
	row("Clip", fmt.Sprint(len(clipped), " ", cap(clipped)))
	var idx []string
	for i, v := range slices.All([]string{"a", "b", "c"}) {
		idx = append(idx, fmt.Sprint(i, v))
	}
	row("All", idx)
	idx = idx[:0]
	for i, v := range slices.Backward([]string{"a", "b", "c"}) {
		idx = append(idx, fmt.Sprint(i, v))
	}
	row("Backward", idx)
	row("Values", slices.Collect(slices.Values([]int{7, 8})))
	row("Collect nil", slices.Collect(slices.Values([]int{})) == nil)
	row("AppendSeq", slices.AppendSeq([]int{1}, slices.Values([]int{2, 3})))
	row("Sorted", slices.Sorted(slices.Values([]int{3, 1, 2})))
	row("SortedFunc", slices.SortedFunc(slices.Values([]int{3, 1, 2}), func(a, b int) int { return b - a }))
	row("SortedStableFunc", slices.SortedStableFunc(slices.Values([]string{"bb", "a", "cc", "d"}), func(a, b string) int { return len(a) - len(b) }))
	var chunks [][]int
	for c := range slices.Chunk([]int{1, 2, 3, 4, 5}, 2) {
		chunks = append(chunks, c)
	}
	row("Chunk", chunks)
	catch("Chunk 0", func() { _ = slices.Chunk([]int{1}, 0) })
	count := 0
	for range slices.Chunk([]int{}, 3) {
		count++
	}
	row("Chunk of empty", count)
	count = 0
	for _, v := range slices.All([]int{1, 2, 3, 4}) {
		count++
		if v == 2 {
			break
		}
	}
	row("All with break", count)
	row("Index of strings", slices.Index([]string{"x", "y"}, "y"))
	type pt struct{ X, Y int }
	row("Contains struct", slices.Contains([]pt{{1, 2}, {3, 4}}, pt{3, 4}))
	row("Sort strings", func() []string { s := []string{"b", "A", "a", "B"}; slices.Sort(s); return s }())
	row("Sort floats", func() []float64 { s := []float64{2.5, -1, 0, 1e9}; slices.Sort(s); return s }())
	row("Equal nil empty", slices.Equal([]int(nil), []int{}))
}

func mapsPkg() {
	m := map[string]int{"pear": 3, "apple": 1, "fig": 2, "kiwi": 5}
	row("Keys sorted", slices.Sorted(maps.Keys(m)))
	row("Values sorted", slices.Sorted(maps.Values(m)))
	row("Keys count", len(slices.Collect(maps.Keys(m))))
	var pairs []string
	for k, v := range maps.All(m) {
		pairs = append(pairs, fmt.Sprintf("%s=%d", k, v))
	}
	sort.Strings(pairs)
	row("All sorted", pairs)
	clone := maps.Clone(m)
	row("Clone Equal", maps.Equal(m, clone))
	clone["pear"] = 30
	row("Clone is a copy", fmt.Sprint(m["pear"], " ", clone["pear"]))
	row("Equal after change", maps.Equal(m, clone))
	row("Equal different len", maps.Equal(m, map[string]int{"pear": 3}))
	row("Equal nil empty", maps.Equal(map[string]int(nil), map[string]int{}))
	row("Equal both nil", maps.Equal(map[string]int(nil), map[string]int(nil)))
	row("Clone nil", maps.Clone(map[string]int(nil)) == nil)
	row("Clone empty", fmt.Sprint(maps.Clone(map[string]int{}) == nil, " ", len(maps.Clone(map[string]int{}))))
	row("EqualFunc", maps.EqualFunc(m, map[string]string{"pear": "3", "apple": "1", "fig": "2", "kiwi": "5"}, func(a int, b string) bool { return strconv.Itoa(a) == b }))
	row("EqualFunc false", maps.EqualFunc(m, map[string]string{"pear": "x"}, func(a int, b string) bool { return true }))
	dst := map[string]int{"apple": 100, "extra": 0}
	maps.Copy(dst, m)
	row("Copy", fmt.Sprint(len(dst), " ", dst["apple"], " ", dst["extra"], " ", slices.Sorted(maps.Keys(dst))))
	maps.DeleteFunc(dst, func(k string, v int) bool { return v%2 == 1 })
	row("DeleteFunc", fmt.Sprint(slices.Sorted(maps.Keys(dst)), " ", slices.Sorted(maps.Values(dst))))
	maps.DeleteFunc(dst, func(k string, v int) bool { return true })
	row("DeleteFunc all", fmt.Sprint(len(dst), " ", dst == nil))
	collected := maps.Collect(maps.All(m))
	row("Collect", fmt.Sprint(len(collected), " ", maps.Equal(m, collected)))
	inserted := map[string]int{"zero": 0}
	maps.Insert(inserted, maps.All(map[string]int{"one": 1, "two": 2}))
	row("Insert", slices.Sorted(maps.Keys(inserted)))
	byIndex := map[int]int{}
	maps.Insert(byIndex, slices.All([]int{7, 8}))
	row("Insert from a slice", fmt.Sprint(byIndex[0], " ", byIndex[1], " ", len(byIndex)))
	empty := maps.Collect(maps.All(map[int]int{}))
	row("Collect of empty", fmt.Sprint(empty == nil, " ", len(empty)))
	count := 0
	for range maps.Keys(m) {
		count++
		if count == 2 {
			break
		}
	}
	row("Keys with break", count)
	byLen := map[int][]string{}
	for _, w := range strings.Fields("a bb cc d eee ff") {
		byLen[len(w)] = append(byLen[len(w)], w)
	}
	for _, k := range slices.Sorted(maps.Keys(byLen)) {
		row(fmt.Sprintf("group %d", k), byLen[k])
	}
	type key struct{ A, B int }
	km := map[key]string{{1, 2}: "x", {0, 5}: "y", {1, 1}: "z"}
	keys := slices.SortedFunc(maps.Keys(km), func(a, b key) int { return cmp.Or(cmp.Compare(a.A, b.A), cmp.Compare(a.B, b.B)) })
	row("struct keys sorted", keys)
	row("cmp.Or", fmt.Sprint(cmp.Or(0, 0, 3, 4), " ", cmp.Or("", "x"), " ", cmp.Or[int]()))
	row("cmp.Compare", fmt.Sprint(cmp.Compare(1, 2), " ", cmp.Compare("b", "a"), " ", cmp.Compare(1.5, 1.5)))
	row("cmp.Less", fmt.Sprint(cmp.Less(1, 2), " ", cmp.Less(2.0, 1.0)))
	nan := 0.0
	nan = nan / nan
	row("cmp.Compare NaN", fmt.Sprint(cmp.Compare(nan, 1.0), " ", cmp.Compare(1.0, nan), " ", cmp.Compare(nan, nan)))
	row("cmp.Less NaN", fmt.Sprint(cmp.Less(nan, 1.0), " ", cmp.Less(1.0, nan)))
	row("min max builtin", fmt.Sprint(min(3, 1, 2), " ", max(3, 1, 2), " ", min("b", "a"), " ", max(1.5, 2.5)))
}

func main() {
	fmt.Println("== errors ==")
	errorsPkg()
	fmt.Println("== sort ==")
	sortPkg()
	fmt.Println("== slices ==")
	slicesPkg()
	fmt.Println("== maps ==")
	mapsPkg()
}
