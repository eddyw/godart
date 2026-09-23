package main

import "fmt"

func panics(call func()) (out bool) {
	defer func() { out = recover() != nil }()
	call()
	return false
}

type record struct{ Value any }
type target struct{ Values []int }

func main() {
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
			panics(func() { _ = interfaces[value] }),
			panics(func() { _, _ = interfaces[value] }),
			panics(func() { delete(interfaces, value) }))
		fmt.Println("array", allocated,
			panics(func() { _ = arrays[array] }),
			panics(func() { _, _ = arrays[array] }),
			panics(func() { delete(arrays, array) }))
		fmt.Println("record", allocated,
			panics(func() { _ = records[entry] }),
			panics(func() { _, _ = records[entry] }),
			panics(func() { delete(records, entry) }))
		fmt.Println("pointer", allocated,
			panics(func() { _ = pointers[pointer] }),
			panics(func() { _, _ = pointers[pointer] }),
			panics(func() { delete(pointers, pointer) }))
	}
}

func generic[K comparable](key K) {
	var nilMap map[K]int
	for _, m := range []map[K]int{nilMap, make(map[K]int)} {
		fmt.Println("generic", panics(func() { _ = m[key] }), panics(func() { _, _ = m[key] }), panics(func() { delete(m, key) }))
	}
}
