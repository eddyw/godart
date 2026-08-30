// The program proves a parallel assignment whose right side reads a struct
// location that the left side writes. The stage must copy the value before
// the write, because a write to a field goes into its object. The level 6
// encoder of compress/flate swaps its hash chain this way, and the loss gave
// a worse ratio.
package main

import "fmt"

type entry struct{ off int32 }

type pair struct{ cur, prev entry }

func main() {
	table := make([]pair, 4)
	table[1] = pair{cur: entry{10}, prev: entry{5}}
	p := &table[1]
	p.cur, p.prev = entry{20}, p.cur
	fmt.Println(table[1])

	var arr [4]pair
	arr[2] = pair{cur: entry{7}, prev: entry{3}}
	q := &arr[2]
	q.cur, q.prev = entry{9}, q.cur
	fmt.Println(arr[2])

	a, b := 1, 2
	a, b = b, a
	fmt.Println(a, b)
}
