package parity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

type Result struct {
	Outcomes map[string]string `json:"outcomes"`
	Exit     int               `json:"exit"`
}

type Exception struct {
	Test   string `json:"test"`
	Native string `json:"native"`
	Gart   string `json:"gart"`
	Reason string `json:"reason"`
}

type Comparison struct {
	Partial  []string    `json:"partial"`
	Passed   []string    `json:"passed"`
	Excluded []Exception `json:"excluded"`
	Problems []string    `json:"problems"`
}

// NativeResults consumes the structured output of go test -json for 1 package.
func NativeResults(r io.Reader, exit int) (Result, error) {
	out := Result{Outcomes: map[string]string{}, Exit: exit}
	dec := json.NewDecoder(r)
	for {
		var e struct{ Action, Test string }
		err := dec.Decode(&e)
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}
		if e.Test != "" && (e.Action == "pass" || e.Action == "skip" || e.Action == "fail") {
			if _, ok := out.Outcomes[e.Test]; ok {
				return out, fmt.Errorf("duplicate native outcome: %s", e.Test)
			}
			out.Outcomes[e.Test] = e.Action
		}
	}
	return out, nil
}

var resultLine = regexp.MustCompile(`^\s*--- (PASS|SKIP|FAIL): (\S+) \([0-9.]+s\)$`)

// GartResults reads terminal test records, not log lines that contain markers.
func GartResults(r io.Reader, exit int) (Result, error) {
	out := Result{Outcomes: map[string]string{}, Exit: exit}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	for scanner.Scan() {
		m := resultLine.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		if _, ok := out.Outcomes[m[2]]; ok {
			return out, fmt.Errorf("duplicate Gart outcome: %s", m[2])
		}
		out.Outcomes[m[2]] = strings.ToLower(m[1])
	}
	return out, scanner.Err()
}

// Compare requires successful processes and accounts for all selected identities.
// An exception applies to exactly 1 observed identity and never counts as parity.
func Compare(native, gart Result, expected []string, exceptions []Exception) Comparison {
	var c Comparison
	if native.Exit != 0 {
		c.Problems = append(c.Problems, fmt.Sprintf("native exit %d", native.Exit))
	}
	if gart.Exit != 0 {
		c.Problems = append(c.Problems, fmt.Sprintf("Gart exit %d", gart.Exit))
	}
	if len(native.Outcomes) == 0 || len(gart.Outcomes) == 0 {
		c.Problems = append(c.Problems, "empty test execution")
	}
	if expected != nil {
		selected := map[string]bool{}
		for _, name := range expected {
			selected[name] = true
		}
		for name := range native.Outcomes {
			top, _, _ := strings.Cut(name, "/")
			if !selected[top] {
				c.Problems = append(c.Problems, "unexpected native test: "+name)
			}
		}
		for name := range gart.Outcomes {
			top, _, _ := strings.Cut(name, "/")
			if !selected[top] {
				c.Problems = append(c.Problems, "unexpected Gart test: "+name)
			}
		}
	}
	for _, name := range expected {
		if _, ok := native.Outcomes[name]; !ok {
			c.Problems = append(c.Problems, "missing native test: "+name)
		}
	}
	rules := map[string]Exception{}
	for _, e := range exceptions {
		if e.Reason == "" || e.Test == "" {
			c.Problems = append(c.Problems, "exception lacks identity or reason")
		}
		if _, ok := rules[e.Test]; ok {
			c.Problems = append(c.Problems, "duplicate exception: "+e.Test)
		}
		rules[e.Test] = e
	}
	names := map[string]bool{}
	for name := range native.Outcomes {
		names[name] = true
	}
	for name := range gart.Outcomes {
		names[name] = true
	}
	for name := range names {
		n, g := native.Outcomes[name], gart.Outcomes[name]
		if e, ok := rules[name]; ok {
			delete(rules, name)
			if n != e.Native || g != e.Gart {
				c.Problems = append(c.Problems, fmt.Sprintf("exception changed: %s: native=%s Gart=%s", name, n, g))
			} else {
				c.Excluded = append(c.Excluded, e)
			}
		} else if n == "pass" && g == "pass" {
			if native.Exit == 0 && gart.Exit == 0 {
				c.Passed = append(c.Passed, name)
			}
		} else {
			c.Problems = append(c.Problems, fmt.Sprintf("unclassified outcome: %s: native=%s Gart=%s", name, n, g))
		}
	}
	for name := range rules {
		c.Problems = append(c.Problems, "unused exception: "+name)
	}
	for i := 0; i < len(c.Passed); {
		name := c.Passed[i]
		partial := false
		for _, e := range c.Excluded {
			if strings.HasPrefix(e.Test, name+"/") {
				partial = true
				break
			}
		}
		if partial {
			c.Partial = append(c.Partial, name)
			c.Passed = append(c.Passed[:i], c.Passed[i+1:]...)
		} else {
			i++
		}
	}
	slices.Sort(c.Partial)
	slices.Sort(c.Passed)
	slices.Sort(c.Problems)
	slices.SortFunc(c.Excluded, func(a, b Exception) int { return strings.Compare(a.Test, b.Test) })
	return c
}
