package main

import "fmt"

type table [4]int

func fill(p *[4]int) {
	for i := range p {
		p[i] = i * i
	}
}

func total(p *table) int {
	n := 0
	for _, v := range p {
		n += v
	}
	return n
}

func main() {
	var arr [4]int
	fill(&arr)
	fmt.Println(arr[0], arr[1], arr[2], arr[3])

	var t table
	t[1] = 5
	fmt.Println(total(&t), len(&t))

	p := new([3]string)
	p[0] = "a"
	fmt.Println(p[0], p[1] == "", len(p))

	// A pointer to an array keeps the identity of the array.
	q := &arr
	q[0] = 99
	fmt.Println(arr[0])

	// A slice of a pointer to an array shares the storage.
	s := q[1:3]
	s[0] = 7
	fmt.Println(arr[1], len(s), cap(s))

	fmt.Println(&arr == &arr, &arr == q)
}
