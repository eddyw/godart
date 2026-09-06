// The program proves the string rules of Go: an index gives a byte, a slice
// gives a string of bytes, a range gives a rune with its byte offset, len
// counts the bytes, and a byte that carries no rune gives the replacement
// rune with a width of 1.
package main

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

var inputs = []string{
	"",
	"a",
	"abc",
	"héllo",
	"日本語",
	"a\x80b",
	"\xff\xfe",
	"é",
	"\U0001F600",
	"a\xc3",
	"\xed\xa0\x80",
	"\xf4\x90\x80\x80",
	"\xc0\x80",
}

func lengths() {
	fmt.Printf("%-14s %-5s %-6s %-6s %s\n", "input", "len", "runes", "valid", "bytes")
	for _, s := range inputs {
		fmt.Printf("%-14q %-5d %-6d %-6v %v\n",
			s, len(s), utf8.RuneCountInString(s), utf8.ValidString(s), []byte(s))
	}
}

func indexing() {
	s := "héllo"
	for i := 0; i < len(s); i++ {
		fmt.Printf("s[%d] = %d %q\n", i, s[i], s[i])
	}
	fmt.Println("s[0:2]", strconv.Quote(s[0:2]))
	fmt.Println("s[1:3]", strconv.Quote(s[1:3]))
	fmt.Println("s[:0] ", strconv.Quote(s[:0]))
	fmt.Println("s[6:] ", strconv.Quote(s[6:]))
	fmt.Println("s[2:2]", strconv.Quote(s[2:2]))
	fmt.Println("len of s[1:3]", len(s[1:3]))
	// A slice of a slice keeps the byte offsets of the new value.
	t := s[1:]
	fmt.Println("t     ", strconv.Quote(t))
	fmt.Println("t[0:2]", strconv.Quote(t[0:2]))
}

func ranges() {
	for _, s := range inputs {
		fmt.Printf("range %-14q:", s)
		for i, r := range s {
			fmt.Printf(" [%d]=%U", i, r)
		}
		fmt.Println()
	}
	fmt.Println("-- the index after the loop --")
	last := -1
	for i := range "日本語" {
		last = i
	}
	fmt.Println("last index", last)
	fmt.Println("-- byte by byte --")
	s := "a\x80b"
	for i := 0; i < len(s); i++ {
		r, size := utf8.DecodeRuneInString(s[i:])
		fmt.Printf("at %d rune=%U size=%d\n", i, r, size)
	}
	fmt.Println("-- the last rune --")
	for _, s := range inputs {
		r, size := utf8.DecodeLastRuneInString(s)
		fmt.Printf("%-14q last=%U size=%d\n", s, r, size)
	}
}

func joins() {
	a := "foo"
	b := "bar"
	fmt.Println("a+b   ", a+b)
	c := a
	c += b
	c += ""
	fmt.Println("c     ", c)
	fmt.Println("a     ", a)
	out := ""
	for i := 0; i < 5; i++ {
		out += strconv.Itoa(i) + ","
	}
	fmt.Println("loop  ", out)
	fmt.Println("empty+", ""+"")
	fmt.Println("len   ", len(a+b+c))
	var zero string
	fmt.Println("zero  ", strconv.Quote(zero), len(zero), zero == "")
}

func compares() {
	pairs := [][2]string{
		{"a", "b"}, {"b", "a"}, {"a", "a"}, {"", "a"}, {"a", ""},
		{"abc", "abd"}, {"abc", "ab"}, {"A", "a"}, {"\x00", ""},
		{"héllo", "hello"}, {"\xff", "\x7f"}, {"日", "本"},
	}
	for _, p := range pairs {
		x, y := p[0], p[1]
		fmt.Printf("%-8q %-8q < %-6v <= %-6v == %-6v >= %-6v > %v\n",
			x, y, x < y, x <= y, x == y, x >= y, x > y)
	}
	// A string is a value, thus 2 equal values compare equal.
	m := map[string]int{"a": 1, "日本": 2}
	fmt.Println("map a  ", m["a"])
	fmt.Println("map 日本", m["日本"])
	fmt.Println("map gap", m["b"])
}

// runeWidth proves the byte count of the encoding of a rune.
func runeWidth() {
	runes := []rune{0, 'a', 0x7F, 0x80, 0x7FF, 0x800, 0xFFFF, 0x10000,
		0x10FFFF, 0xD800, -1, 0x110000, 0xFFFD}
	for _, r := range runes {
		buf := make([]byte, 4)
		n := utf8.EncodeRune(buf, r)
		fmt.Printf("%-10d len=%d valid=%-6v bytes=%v text=%q\n",
			r, utf8.RuneLen(r), utf8.ValidRune(r), buf[:n], string(r))
	}
}

func main() {
	fmt.Println("== the length ==")
	lengths()
	fmt.Println("== the index and the slice ==")
	indexing()
	fmt.Println("== the range ==")
	ranges()
	fmt.Println("== the concatenation ==")
	joins()
	fmt.Println("== the comparison ==")
	compares()
	fmt.Println("== the width of a rune ==")
	runeWidth()
}
