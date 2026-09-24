package main

import (
	"errors"
	"fmt"
	"syscall"
)

func recovered(name string, call func()) {
	defer func() { fmt.Printf("%s panic=%v\n", name, recover()) }()
	call()
}

func pointers(name string, values []*byte) {
	fmt.Print(name, " ", len(values), " ", cap(values))
	for _, value := range values {
		if value == nil {
			fmt.Print(" nil")
		} else {
			fmt.Print(" ", *value)
		}
	}
	fmt.Println()
	for i, value := range values {
		for _, other := range values[:i] {
			if value != nil && value == other {
				panic("distinct strings share a byte")
			}
		}
	}
}

func main() {
	for _, s := range []string{"", "hello", "世界", "\xff\xfe", "\x00", "a\x00b", "ab\x00"} {
		b, err := syscall.ByteSliceFromString(s)
		fmt.Printf("slice %x %x %d %d %t %t\n", s, b, len(b), cap(b), b == nil, errors.Is(err, syscall.EINVAL))
		p, err := syscall.BytePtrFromString(s)
		fmt.Println("pointer", p == nil, errors.Is(err, syscall.EINVAL))
		if p != nil {
			fmt.Println("first", *p)
			*p = 42
			q, err := syscall.BytePtrFromString(s)
			fmt.Println("fresh", *q, p == q, err == nil)
		}
		recovered("StringByteSlice", func() {
			fmt.Printf("bytes %x\n", syscall.StringByteSlice(s))
		})
		recovered("StringBytePtr", func() {
			fmt.Println("byte", *syscall.StringBytePtr(s))
		})
	}
	for _, values := range [][]string{nil, {}, {""}, {"", ""}, {"hello", "世界", "\xff", ""}, {"a", "\x00"}, {"a\x00b", "c"}} {
		p, err := syscall.SlicePtrFromStrings(values)
		fmt.Printf("strings %x %t %t\n", values, p == nil, errors.Is(err, syscall.EINVAL))
		pointers("slice", p)
		recovered("StringSlicePtr", func() {
			pointers("legacy", syscall.StringSlicePtr(values))
		})
	}
}
