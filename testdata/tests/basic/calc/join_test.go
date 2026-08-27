package calc_test

import (
	"testing"

	"example.com/basic/calc"
)

func TestJoin(t *testing.T) {
	if got := calc.Join([]string{"a", "b"}, "-"); got != "a-b" {
		t.Fatalf("Join gives %q, and a-b is the text", got)
	}
}
