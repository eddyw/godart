package main

import "fmt"

type index interface{ ~int64 | ~uint64 }

func report(name string, f func()) {
	defer func() { fmt.Println(name, recover()) }()
	f()
}
func mixed[C ~string | ~[]byte, I index](v C, i I) {
	report("mixed index", func() { _ = v[i] })
	report("mixed low", func() { _ = v[i:1] })
	report("mixed high", func() { _ = v[:i] })
}
func indexes[I index](i I) {
	a := [2]byte{1, 2}
	s := a[:]
	p := &a
	text := "ab"
	report("string", func() { _ = text[i] })
	report("array ptr", func() { _ = &a[i] })
	report("slice ptr", func() { _ = &s[i] })
	report("pointer ptr", func() { _ = &p[i] })
	report("string low", func() { _ = text[i:1] })
	report("string high", func() { _ = text[:i] })
	report("array low", func() { _ = a[i:1] })
	report("array high", func() { _ = a[:i] })
	report("slice low", func() { _ = s[i:1] })
	report("slice high", func() { _ = s[:i] })
	report("slice max", func() { _ = s[:1:i] })
	mixed(text, i)
	mixed(s, i)
}
func main() {
	indexes(int64(-1))
	indexes(^uint64(0))
	indexes(int64(1))
	indexes(uint64(1))
	var i uint64 = ^uint64(0)
	a := [2]byte{}
	report("concrete address", func() { _ = &a[i] })
}
