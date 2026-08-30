// The file holds no build tag, because the compiler and the harness both read
// it. The compiler builds it as a package of the module, and the harness builds
// it as a part of the overlay of testing. Thus 1 rule filters the skip line of
// the driver and the test of a binary.
package testing

import "strings"

// NoTestsToRun is the warning of a run that holds no test. The harness writes
// the line, and the driver reads it and marks the package.
const NoTestsToRun = "testing: warning: no tests to run"

// MatchName reports whether a test name passes the filter of -test.run or of
// -test.bench. An empty pattern passes each name.
//
// The name holds 1 element for each level of a subtest, and the elements stand
// between the slashes. A name that holds fewer elements than the pattern
// passes, because a subtest of that name can still match.
//
// matchString matches 1 expression against 1 element. The package testing
// imports no regexp, thus the caller gives the function.
func MatchName(name, pattern string, matchString func(pat, str string) (bool, error)) bool {
	if pattern == "" || matchString == nil {
		return true
	}
	elems := strings.Split(name, "/")
	for _, alt := range alternatives(pattern) {
		if matchElems(elems, alt, matchString) {
			return true
		}
	}
	return false
}

func matchElems(elems, alt []string, matchString func(pat, str string) (bool, error)) bool {
	for i, elem := range elems {
		if i >= len(alt) {
			break
		}
		ok, err := matchString(alt[i], elem)
		if err != nil || !ok {
			return false
		}
	}
	return true
}

// alternatives splits a pattern into its alternatives, and each alternative
// into 1 expression for each element of a test name. A vertical bar separates
// 2 alternatives, and a slash separates 2 elements.
//
// A bar or a slash inside a bracket group or inside a parenthesis group belongs
// to the expression. A backslash escapes the next byte. The rule comes from
// splitRegexp of testing/match.go of Go.
func alternatives(pattern string) [][]string {
	var out [][]string
	var elems []string
	brackets, parens, start := 0, 0, 0
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '[':
			brackets++
		case ']':
			if brackets--; brackets < 0 {
				brackets = 0
			}
		case '(':
			if brackets == 0 {
				parens++
			}
		case ')':
			if brackets == 0 {
				parens--
			}
		case '\\':
			i++
		case '/':
			if brackets == 0 && parens == 0 {
				elems = append(elems, pattern[start:i])
				start = i + 1
			}
		case '|':
			if brackets == 0 && parens == 0 {
				elems = append(elems, pattern[start:i])
				start = i + 1
				out = append(out, elems)
				elems = nil
			}
		}
	}
	return append(out, append(elems, pattern[start:]))
}
