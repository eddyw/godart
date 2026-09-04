package main

import (
	"bytes"
	"fmt"
	"index/suffixarray"
	"regexp"
	"sort"
)

func main() {
	data := []byte("the quick brown fox jumps over the lazy dog")
	index := suffixarray.New(data)

	fmt.Println("bytes:", string(index.Bytes()) == string(data))

	hits := index.Lookup([]byte("the"), -1)
	sort.Ints(hits)
	fmt.Println("the:", hits)

	fmt.Println("one:", len(index.Lookup([]byte("quick"), 1)))
	fmt.Println("none:", index.Lookup([]byte("absent"), -1))

	re := regexp.MustCompile(`o[a-z]`)
	found := index.FindAllIndex(re, -1)
	fmt.Println("regexp:", found)

	var buf bytes.Buffer
	if err := index.Write(&buf); err != nil {
		fmt.Println("write:", err)
		return
	}
	fmt.Println("wrote:", buf.Len() > 0)

	var again suffixarray.Index
	if err := again.Read(&buf); err != nil {
		fmt.Println("read:", err)
		return
	}
	back := again.Lookup([]byte("lazy"), -1)
	fmt.Println("after read:", back, string(again.Bytes()) == string(data))

	big := make([]byte, 0, 600)
	for i := 0; i < 100; i++ {
		big = append(big, "banana"...)
	}
	bigIndex := suffixarray.New(big)
	fmt.Println("big:", len(bigIndex.Lookup([]byte("ana"), -1)))
}
