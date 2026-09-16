package parity

import (
	"fmt"
	"regexp"
	"strings"
)

var errorsAsCase = regexp.MustCompile(`^(TestAs/[0-9]+):As\(`)

// CanonicalNames preserves the upstream case index when errors tests embed
// process-specific pointer text in a name. Test assertions remain unchanged.
func CanonicalNames(pkg string, result Result) (Result, error) {
	if pkg != "errors" {
		return result, nil
	}
	out := Result{Outcomes: map[string]string{}, Exit: result.Exit}
	for name, status := range result.Outcomes {
		canonical := name
		if m := errorsAsCase.FindStringSubmatch(name); m != nil {
			canonical = m[1]
		}
		if strings.HasPrefix(name, "TestAsValidation/*string(") {
			canonical = "TestAsValidation/*string"
		}
		if _, ok := out.Outcomes[canonical]; ok {
			return out, fmt.Errorf("identity adapter collision: %s", canonical)
		}
		out.Outcomes[canonical] = status
	}
	return out, nil
}
