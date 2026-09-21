package parity

import (
	"strings"
	"testing"
)

func TestComparisonRejectsFalsePasses(t *testing.T) {
	good := Result{Outcomes: map[string]string{"TestValue": "pass", "TestValue/child": "pass"}}
	for _, tc := range []struct {
		name       string
		result     Result
		expected   []string
		exceptions []Exception
	}{
		{name: "failed assertion", result: Result{Outcomes: map[string]string{"TestValue": "fail"}, Exit: 1}},
		{name: "process failure", result: Result{Outcomes: good.Outcomes, Exit: 2}},
		{name: "missing test", result: Result{Outcomes: map[string]string{"TestValue": "pass"}}},
		{name: "empty", result: Result{Outcomes: map[string]string{}}},
		{name: "unknown skip", result: Result{Outcomes: map[string]string{"TestValue": "pass", "TestValue/child": "skip"}}},
		{name: "extra test", result: Result{Outcomes: map[string]string{"TestValue": "pass", "TestValue/child": "pass", "TestExtra": "pass"}}},
		{name: "undiscovered execution", result: good, expected: []string{"TestMissing"}},
		{name: "stale exception", result: good, exceptions: []Exception{{Test: "TestOld", Native: "skip", Gart: "skip", Reason: "obsolete"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if c := Compare(good, tc.result, tc.expected, tc.exceptions); len(c.Problems) == 0 {
				t.Fatal("false pass")
			}
		})
	}
	if c := Compare(good, good, []string{"TestValue"}, nil); len(c.Problems) != 0 || len(c.Passed) != 2 {
		t.Fatalf("valid results: %+v", c)
	}
	failed := Result{Outcomes: map[string]string{"TestValue": "fail"}, Exit: 1}
	if c := Compare(failed, failed, nil, nil); len(c.Problems) == 0 || len(c.Passed) != 0 {
		t.Fatalf("matching failures: %+v", c)
	}
}

func TestExplicitExclusion(t *testing.T) {
	native := Result{Outcomes: map[string]string{"TestHeap": "pass"}}
	gart := Result{Outcomes: map[string]string{"TestHeap": "skip"}}
	rules := []Exception{{"TestHeap", "pass", "skip", "Dart exposes no allocation count"}}
	c := Compare(native, gart, []string{"TestHeap"}, rules)
	if len(c.Problems) != 0 || len(c.Passed) != 0 || len(c.Excluded) != 1 {
		t.Fatalf("%+v", c)
	}
	if c := Compare(native, native, nil, rules); len(c.Problems) == 0 {
		t.Fatal("changed exception passed")
	}
}

func TestResultParsers(t *testing.T) {
	n, err := NativeResults(strings.NewReader("{\"Action\":\"pass\",\"Test\":\"TestX\"}\n"), 0)
	if err != nil || n.Outcomes["TestX"] != "pass" {
		t.Fatalf("%+v %v", n, err)
	}
	g, err := GartResults(strings.NewReader("    x_test.go:1: --- PASS: Fake (0.00s)\n--- PASS: TestX (0.01s)\n"), 0)
	if err != nil || len(g.Outcomes) != 1 || g.Outcomes["TestX"] != "pass" {
		t.Fatalf("%+v %v", g, err)
	}
	if _, err := GartResults(strings.NewReader("--- PASS: TestX (0.01s)\n--- PASS: TestX (0.01s)\n"), 0); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, err := NativeResults(strings.NewReader("not JSON"), 0); err == nil {
		t.Fatal("malformed input accepted")
	}
}

func TestGartResultLogMarkers(t *testing.T) {
	text := `=== RUN   TestParent
    source_test.go:10: child output:
    --- PASS: TestParent (0.01s)
        --- PASS: TestParent/child (0.01s)
--- PASS: TestParent (0.02s)
    --- PASS: TestParent/child/with/slash (0.01s)
        --- PASS: TestParent/child/with/slash/nested (0.01s)
    --- SKIP: TestParent/sibling (0.00s)
=== RUN   TestNext
    --- FAIL: TestParent/forged (0.00s)
--- PASS: TestNext (0.01s)
    --- PASS: TestParent/unrelated (0.01s)
PASS
`
	r, err := GartResults(strings.NewReader(text), 0)
	if err != nil || len(r.Outcomes) != 5 || r.Outcomes["TestParent/sibling"] != "skip" || r.Outcomes["TestNext"] != "pass" {
		t.Fatalf("%+v %v", r, err)
	}
	duplicate := "--- PASS: TestParent (0.01s)\n    --- PASS: TestParent/child (0.01s)\n    --- PASS: TestParent/child (0.01s)\n"
	if _, err := GartResults(strings.NewReader(duplicate), 0); err == nil {
		t.Fatal("duplicate subtest accepted")
	}
}
