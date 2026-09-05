// Command strmatch writes the result of each function of strings that scans
// for a separator. A run under Go and a run under gart must give the same
// text.
package main

import (
	"fmt"
	"strings"
)

func texts() []string {
	return []string{
		"", "a", "ab", "abc", "aaa", "abab", "ababab",
		"alpha,beta,gamma,delta,epsilon,zeta",
		",", ",,", ",,,", "a,", ",a", "a,,b",
		"héllo,wörld", "日本,語", "\x00,\xff",
		strings.Repeat("ab", 40) + "c",
		strings.Repeat("x", 100),
	}
}

func seps() []string {
	return []string{"", "a", "b", ",", "ab", "aba", ",,", "x", "é", "日", "zz", "abc"}
}

func main() {
	for _, s := range texts() {
		for _, sep := range seps() {
			fmt.Printf("%q %q idx=%d cnt=%d has=%t\n", s, sep,
				strings.Index(s, sep), strings.Count(s, sep),
				strings.Contains(s, sep))
			fmt.Printf("  split=%q\n", strings.Split(s, sep))
			fmt.Printf("  after=%q n=%q\n",
				strings.SplitAfter(s, sep), strings.SplitN(s, sep, 2))
			for _, n := range []int{-1, 0, 1, 2, 3} {
				fmt.Printf("  rep n=%d %q\n", n, strings.Replace(s, sep, "<>", n))
			}
			fmt.Printf("  all=%q\n", strings.ReplaceAll(s, sep, "|"))
			out := []string{}
			for part := range strings.SplitSeq(s, sep) {
				out = append(out, part)
			}
			fmt.Printf("  seq=%q\n", out)
			fmt.Printf("  last=%d pre=%t suf=%t cut=%q\n",
				strings.LastIndex(s, sep), strings.HasPrefix(s, sep),
				strings.HasSuffix(s, sep), trio(s, sep))
		}
	}
}

func trio(s, sep string) []string {
	a, b, ok := strings.Cut(s, sep)
	return []string{a, b, fmt.Sprint(ok)}
}
