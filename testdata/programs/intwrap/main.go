package main

import "fmt"

func main() {
	x := int64(1) << 62
	fmt.Println(x + x)
	var u uint64 = 1<<64 - 1
	fmt.Println(u >> 40)
	fmt.Println(u / 3)
	h := uint64(14695981039346656037)
	for _, b := range []byte("godart") {
		h ^= uint64(b)
		h *= 1099511628211
	}
	fmt.Println(h)
	a := int64(-9223372036854775808)
	fmt.Println(a / -1)
	fmt.Println(a%7, int64(7)%-3)
	p1 := int64(6364136223846793005)
	p2 := int64(1442695040888963407)
	fmt.Println(p1 * p2)
	var i8 int8 = 127
	i8++
	fmt.Println(i8)
	fmt.Printf("%x %b %o\n", u, uint32(0xF0F0), int64(-255))
	m := map[int64]string{1 << 60: "high"}
	fmt.Println(m[1<<60], len(m))
	g := 3.9
	g2 := -2.7
	fmt.Println(int(g), int64(g2))
	f := 2.5
	fmt.Println(int64(f * 4))
	s := []int{5, 4, 3, 2, 1}
	total := 0
	for i, v := range s {
		total += i * v
	}
	fmt.Println(total, s[1:3], cap(s[:2]))
	one := 1
	small := int32(1<<31 - 1)
	small += int32(one)
	fmt.Println(uint64(one)<<63, small)
}
