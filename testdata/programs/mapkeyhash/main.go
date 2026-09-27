package main

import "fmt"

func panicText(call func()) (out string) {
	defer func() { out = fmt.Sprint(recover()) }()
	call()
	return ""
}

type record struct{ Value any }
type target struct{ Values []int }

type keyRecord struct{ Values [2]int }

func main() {
	copyKeys([2]int{1, 2}, [2]int{3, 4})
	copyKeys(keyRecord{[2]int{1, 2}}, keyRecord{[2]int{3, 4}})
	copyKeys([1][1]int{{1}}, [1][1]int{{2}})
	copyKeys(any([2]int{1, 2}), any([2]int{3, 4}))
	copyKeys(&keyRecord{[2]int{1, 2}}, &keyRecord{[2]int{3, 4}})
	generic([1]any{[]int{1}})
	generic(&target{[]int{1}})

	for _, allocated := range []bool{false, true} {
		var interfaces map[any]int
		var arrays map[[1]any]int
		var records map[record]int
		var pointers map[*target]int
		if allocated {
			interfaces = make(map[any]int)
			arrays = make(map[[1]any]int)
			records = make(map[record]int)
			pointers = make(map[*target]int)
		}
		value := []int{1}
		array := [1]any{value}
		entry := record{value}
		pointer := &target{value}
		fmt.Println("interface", allocated,
			panicText(func() { _ = interfaces[value] }),
			panicText(func() { _, _ = interfaces[value] }),
			panicText(func() { delete(interfaces, value) }), panicText(func() { interfaces[value] = 1 }))
		fmt.Println("array", allocated,
			panicText(func() { _ = arrays[array] }),
			panicText(func() { _, _ = arrays[array] }),
			panicText(func() { delete(arrays, array) }), panicText(func() { arrays[array] = 1 }))
		fmt.Println("record", allocated,
			panicText(func() { _ = records[entry] }),
			panicText(func() { _, _ = records[entry] }),
			panicText(func() { delete(records, entry) }), panicText(func() { records[entry] = 1 }))
		fmt.Println("pointer", allocated,
			panicText(func() { _ = pointers[pointer] }),
			panicText(func() { _, _ = pointers[pointer] }),
			panicText(func() { delete(pointers, pointer) }), panicText(func() { pointers[pointer] = 1 }))
	}
}

func generic[K comparable](key K) {
	var nilMap map[K]int
	for _, m := range []map[K]int{nilMap, make(map[K]int)} {
		fmt.Println("generic", panicText(func() { _ = m[key] }), panicText(func() { _, _ = m[key] }), panicText(func() { delete(m, key) }), panicText(func() { m[key] = 1 }))
	}
}

func copyKeys[K comparable](first, second K) {
	original := first
	m := make(map[K]int)
	m[first] = 1
	first = second
	fmt.Println("inserted key", len(m), m[original], m[first])
	first = original
	first, m[first] = second, 2
	fmt.Println("parallel key", len(m), m[original], m[first])
	first = original
	m[first] += 3
	first = second
	fmt.Println("updated key", len(m), m[original], m[first])
	first = original
	m[original] = func() int { first = second; return 7 }()
	fmt.Println("key after call", len(m), m[original], m[second])
	first = original
	ch := make(chan int)
	go func() { ch <- 8 }()
	m[first] = <-ch
	first = second
	fmt.Println("key after receive", len(m), m[original], m[first])
}
