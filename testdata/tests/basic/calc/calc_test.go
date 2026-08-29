package calc

import "testing"

// doubleCase is 1 row of the table of TestDouble. An unnamed struct type is
// outside the subset, thus the table needs a named type.
type doubleCase struct {
	name string
	in   int
	want int
}

func TestAdd(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Errorf("Add(2, 3) = %d, and 5 is the sum", got)
	}
}

func TestDouble(t *testing.T) {
	cases := []doubleCase{
		{name: "zero", in: 0, want: 0},
		{name: "one", in: 1, want: 2},
		{name: "minus one", in: -1, want: -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Double(c.in); got != c.want {
				t.Errorf("Double(%d) = %d, and %d is the result", c.in, got, c.want)
			}
		})
	}
}

func TestSkipped(t *testing.T) {
	t.Skip("the case waits for a decision")
}

func TestCleanup(t *testing.T) {
	t.Cleanup(func() { t.Log("the cleanup ran") })
	t.Log("the body ran")
}

func wantAdd(t *testing.T, a, b, want int) {
	t.Helper()
	if got := Add(a, b); got != want {
		t.Errorf("Add(%d, %d) = %d, and %d is the result", a, b, got, want)
	}
	t.Log("the helper ran")
}

func TestHelperPosition(t *testing.T) {
	wantAdd(t, 1, 2, 3)
}
