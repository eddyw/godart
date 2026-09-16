// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testing

import (
	"fmt"
	"strconv"
	"strings"
)

type subtestNames struct{ subNames map[string]int32 }

// unique creates a unique name for the given parent and subname by affixing it
// with one or more counts, if necessary.
func (m *subtestNames) unique(parent, subname string) string {
	base := parent + "/" + subname

	for {
		n := m.subNames[base]
		if n < 0 {
			panic("subtest count overflow")
		}
		m.subNames[base] = n + 1

		if n == 0 && subname != "" {
			prefix, nn := parseSubtestNumber(base)
			if len(prefix) < len(base) && nn < m.subNames[prefix] {
				// This test is explicitly named like "parent/subname#NN",
				// and #NN was already used for the NNth occurrence of "parent/subname".
				// Loop to add a disambiguating suffix.
				continue
			}
			return base
		}

		name := fmt.Sprintf("%s#%02d", base, n)
		if m.subNames[name] != 0 {
			// This is the nth occurrence of base, but the name "parent/subname#NN"
			// collides with the first occurrence of a subtest *explicitly* named
			// "parent/subname#NN". Try the next number.
			continue
		}

		return name
	}
}

// parseSubtestNumber splits a subtest name into a "#%02d"-formatted int32
// suffix (if present), and a prefix preceding that suffix (always).
func parseSubtestNumber(s string) (prefix string, nn int32) {
	i := strings.LastIndex(s, "#")
	if i < 0 {
		return s, 0
	}

	prefix, suffix := s[:i], s[i+1:]
	if len(suffix) < 2 || (len(suffix) > 2 && suffix[0] == '0') {
		// Even if suffix is numeric, it is not a possible output of a "%02" format
		// string: it has either too few digits or too many leading zeroes.
		return s, 0
	}
	if suffix == "00" {
		if !strings.HasSuffix(prefix, "/") {
			// We only use "#00" as a suffix for subtests named with the empty
			// string — it isn't a valid suffix if the subtest name is non-empty.
			return s, 0
		}
	}

	n, err := strconv.ParseInt(suffix, 10, 32)
	if err != nil || n < 0 {
		return s, 0
	}
	return prefix, int32(n)
}

// rewrite rewrites a subname to having only printable characters and no white
// space.
func subName(s string) string {
	b := []byte{}
	for _, r := range s {
		switch {
		case isSpace(r):
			b = append(b, '_')
		case !strconv.IsPrint(r):
			s := strconv.QuoteRune(r)
			b = append(b, s[1:len(s)-1]...)
		default:
			b = append(b, string(r)...)
		}
	}
	return string(b)
}

func isSpace(r rune) bool {
	if r < 0x2000 {
		switch r {
		// Note: not the same as Unicode Z class.
		case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0, 0x1680:
			return true
		}
	} else {
		if r <= 0x200a {
			return true
		}
		switch r {
		case 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
			return true
		}
	}
	return false
}
