package main

import (
	"container/heap"
	"container/ring"
	"encoding/ascii85"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"hash/fnv"
	"strings"
	"sync"
)

// counterOf holds a function literal inside the initializer of a package
// variable. The literal declares locals of its own.
var counterOf = sync.OnceValue(func() int {
	total := 0
	for i := 1; i <= 4; i++ {
		total += i * i
	}
	return total
})

var greet = sync.OnceFunc(func() { fmt.Println("once") })

type intHeap []int

func (h intHeap) Len() int           { return len(h) }
func (h intHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h intHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *intHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *intHeap) Pop() any          { old := *h; n := len(old); v := old[n-1]; *h = old[:n-1]; return v }

func heaps() {
	// The address of a composite literal of a type that is not a struct
	// stays outside the subset, thus the heap takes a variable first.
	h := intHeap{5, 2, 9, 1, 7}
	heap.Init(&h)
	heap.Push(&h, 3)
	out := []int{}
	for h.Len() > 0 {
		out = append(out, heap.Pop(&h).(int))
	}
	fmt.Println("heap:", out)
}

func rings() {
	r := ring.New(5)
	for i := 0; i < r.Len(); i++ {
		r.Value = i * 2
		r = r.Next()
	}
	sum := 0
	r.Do(func(v any) { sum += v.(int) })
	fmt.Println("ring:", r.Len(), sum, r.Move(2).Value)
}

func hashes() {
	data := []byte("the quick brown fox")
	f32 := fnv.New32a()
	f32.Write(data)
	f64 := fnv.New64a()
	f64.Write(data)
	fmt.Println("fnv:", f32.Sum32(), f64.Sum64())
	fmt.Println("crc32:", crc32.ChecksumIEEE(data),
		crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
	fmt.Println("adler32:", adler32.Checksum(data))
}

func codecs() {
	data := []byte{0x00, 0x7f, 0x80, 0xff, 0x10}
	text := hex.EncodeToString(data)
	back, err := hex.DecodeString(text)
	fmt.Println("hex:", text, back, err)
	fmt.Println("hex dump:", strings.TrimSpace(hex.Dump(data)) != "")

	buf := make([]byte, ascii85.MaxEncodedLen(len(data)))
	n := ascii85.Encode(buf, data)
	fmt.Println("ascii85:", string(buf[:n]))

	r := csv.NewReader(strings.NewReader("a,b,c\n1,\"x,y\",3\n"))
	rows, err := r.ReadAll()
	fmt.Println("csv:", rows, err)

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	w.WriteAll([][]string{{"p", "q"}, {"1", "two words"}})
	fmt.Printf("csv out: %q\n", sb.String())
}

// blocking proves a closure that can block inside a generic function. The
// frame class of such a closure stands at package level, thus it names the type
// parameters of the function.
func blocking() {
	ch := make(chan int, 1)
	ch <- 41
	g := sync.OnceValue(func() int { return <-ch + 1 })
	h := sync.OnceValues(func() (string, int) { return "n", <-ch })
	fmt.Println("lazy:", g(), g())
	ch <- 5
	a, b := h()
	fmt.Println("lazy pair:", a, b)
}

func main() {
	fmt.Println("once value:", counterOf(), counterOf())
	greet()
	greet()
	blocking()
	heaps()
	rings()
	hashes()
	codecs()
}
