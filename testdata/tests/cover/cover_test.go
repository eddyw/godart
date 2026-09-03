package cover

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{20, "high"},
		{5, "low"},
		{0, "zero"},
	}
	for _, c := range cases {
		if got := Classify(c.in); got != c.want {
			t.Errorf("Classify(%d) gives %q, and %q is the answer", c.in, got, c.want)
		}
	}
}

func TestShort(t *testing.T) {
	if testing.Short() {
		t.Skip("the short form skips this test")
	}
	if Classify(-1) != "negative" {
		t.Error("Classify(-1) gives the wrong text")
	}
}
