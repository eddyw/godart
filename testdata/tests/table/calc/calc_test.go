package calc

import "testing"

func TestClamp(t *testing.T) {
	cases := []struct {
		name string
		in   int
		low  int
		high int
		want int
	}{
		{name: "inside", in: 5, low: 0, high: 10, want: 5},
		{name: "below", in: -3, low: 0, high: 10, want: 0},
		{name: "above", in: 42, low: 0, high: 10, want: 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Clamp(c.in, c.low, c.high); got != c.want {
				t.Errorf("Clamp gives %d, want %d", got, c.want)
			}
		})
	}
}

func TestSum(t *testing.T) {
	for _, c := range []struct {
		in   []int
		want int
	}{
		{in: nil, want: 0},
		{in: []int{1, 2, 3}, want: 6},
	} {
		if got := Sum(c.in); got != c.want {
			t.Errorf("Sum(%v) gives %d, want %d", c.in, got, c.want)
		}
	}
}
