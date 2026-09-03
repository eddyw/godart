// casefold reads the case mapping and the simple case folding of Go.
package main

import (
	"fmt"
	"strings"
	"unicode"
)

var texts = []string{
	"ß", "ﬁ", "İ", "ı", "ǅ", "Σσς", "µ", "ǰ", "\xff", "aA1", "",
	"héllo, 世界\xff", "ABC", "abc",
}

var pairs = [][2]string{
	{"Go", "GO"}, {"K", "K"}, {"ß", "SS"}, {"ſ", "s"}, {"İ", "i"},
	{"Σ", "σ"}, {"Σ", "ς"}, {"σ", "ς"}, {"ǅ", "ǆ"}, {"ǅ", "Ǆ"},
	{"ﬁ", "fi"}, {"a\xffb", "A\xffB"}, {"", ""}, {"a", "ab"},
	{"Å", "Å"}, {"θ", "ϑ"}, {"Ω", "Ω"}, {"a\xffb", "a\xffb"},
}

func main() {
	for _, s := range texts {
		fmt.Printf("%q up=%q low=%q title=%q\n",
			s, strings.ToUpper(s), strings.ToLower(s), strings.ToTitle(s))
	}
	for _, p := range pairs {
		fmt.Printf("%q %q %v\n", p[0], p[1], strings.EqualFold(p[0], p[1]))
	}
	fmt.Println(string(unicode.ToUpper('ß')), string(unicode.ToLower('İ')))
	fmt.Println(strings.Map(func(r rune) rune { return r + 1 }, "abc"))
	fmt.Printf("%q\n", strings.Map(unicode.ToUpper, "a\xffb"))
}
