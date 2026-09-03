package ui

import "testing"

// TestAnEqualWriteMarksNoScope reads the equality test of a signal.
func TestAnEqualWriteMarksNoScope(t *testing.T) {
	Reset()
	s := NewSignalEq(1, Eq[int])
	runs := 0
	stop := Effect(func() { s.Get(); runs++ })
	s.Set(1)
	if runs != 1 {
		t.Fatalf("an equal write ran the effect %d times, and 1 is correct", runs)
	}
	s.Set(2)
	if runs != 2 {
		t.Fatalf("a different write ran the effect %d times, and 2 is correct", runs)
	}
	stop()
}

// TestAPlainSignalMarksEachWrite keeps the behavior of NewSignal.
func TestAPlainSignalMarksEachWrite(t *testing.T) {
	Reset()
	s := NewSignal(1)
	runs := 0
	stop := Effect(func() { s.Get(); runs++ })
	s.Set(1)
	if runs != 2 {
		t.Fatalf("the effect ran %d times, and 2 is correct", runs)
	}
	stop()
}

// TestAnEqualityTestOfASliceValue proves that 1 type serves a T that no ==
// operator accepts.
func TestAnEqualityTestOfASliceValue(t *testing.T) {
	Reset()
	same := func(a, b []int) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	s := NewSignalEq([]int{1, 2}, same)
	runs := 0
	stop := Effect(func() { s.Get(); runs++ })
	s.Set([]int{1, 2})
	if runs != 1 {
		t.Fatalf("an equal write ran the effect %d times, and 1 is correct", runs)
	}
	s.Set([]int{1, 3})
	if runs != 2 {
		t.Fatalf("a different write ran the effect %d times, and 2 is correct", runs)
	}
	stop()
}
