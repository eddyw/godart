package analyze

import (
	"fmt"
	"testing"
)

func TestRecursiveComponents(t *testing.T) {
	for _, order := range [][]any{{"a", "b", "c"}, {"a", "c", "b"}, {"b", "a", "c"}, {"b", "c", "a"}, {"c", "a", "b"}, {"c", "b", "a"}} {
		for _, reversed := range []bool{false, true} {
			t.Run(fmt.Sprint(order, reversed), func(t *testing.T) {
				edges := []any{"b", "c"}
				if reversed {
					edges = []any{"c", "b"}
				}
				a := &analyzer{result: &Result{blocking: map[any]bool{}}, summaries: map[any]*summary{
					"a":      {sites: []*callSite{{targets: edges}}},
					"b":      {sites: []*callSite{{targets: []any{"a"}}}},
					"c":      {sites: []*callSite{{targets: []any{"b"}}}},
					"self":   {sites: []*callSite{{targets: []any{"self"}}}},
					"caller": {sites: []*callSite{{targets: []any{"a"}}}},
					"leaf":   {},
				}, order: append(append([]any{}, order...), "self", "caller", "leaf")}
				a.recursive()
				for _, key := range []any{"a", "b", "c", "self", "caller", "leaf"} {
					want := key != "caller" && key != "leaf"
					if a.result.blocking[key] != want {
						t.Errorf("recursive(%v) = %v, want %v", key, a.result.blocking[key], want)
					}
				}
			})
		}
	}
}
