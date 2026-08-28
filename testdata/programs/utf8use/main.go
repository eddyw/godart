package main

import (
	"fmt"
	"unicode/utf8"
)

func main() {
	s := "héllo, 世界"
	fmt.Println(utf8.RuneCountInString(s), len(s))
	r, size := utf8.DecodeRuneInString(s[1:])
	fmt.Println(string(r), size)
	fmt.Println(utf8.ValidString(s), utf8.ValidString("\xff"))
	fmt.Println(utf8.RuneLen('a'), utf8.RuneLen('世'), utf8.RuneLen(-1))
	buf := make([]byte, 4)
	n := utf8.EncodeRune(buf, '界')
	fmt.Println(n, utf8.Valid(buf[:n]))
	fmt.Println(utf8.RuneCount([]byte(s)))
	lr, ls := utf8.DecodeLastRuneInString(s)
	fmt.Println(string(lr), ls)
	lb, lbs := utf8.DecodeLastRune([]byte(s))
	fmt.Println(string(lb), lbs)
	fmt.Println(utf8.FullRuneInString("世"), utf8.FullRuneInString("\xe4\xb8"))
	fmt.Println(utf8.RuneStart(s[0]), utf8.RuneStart(s[1]))
	fmt.Println(utf8.ValidRune('a'), utf8.ValidRune(0xD800), utf8.ValidRune(0x110000))
	out := utf8.AppendRune([]byte("x"), '世')
	fmt.Println(string(out), len(out))
	e, es := utf8.DecodeRuneInString("")
	fmt.Println(e == utf8.RuneError, es)
	// An overlong form and a truncated sequence must both give RuneError.
	b1, s1 := utf8.DecodeRune([]byte{0xC0, 0x80})
	b2, s2 := utf8.DecodeRune([]byte{0xE4, 0xB8})
	fmt.Println(b1 == utf8.RuneError, s1, b2 == utf8.RuneError, s2)
	for i, c := range "aé世" {
		fmt.Print(i, string(c), " ")
	}
	fmt.Println()
}
