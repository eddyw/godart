package ui

import "testing"

// TestASliceThatGrowsRejectsTheMovedSignal names the copy that `go vet` misses.
// append copies each element when the slice grows, thus the 2 subscriber lists
// would drift apart.
func TestASliceThatGrowsRejectsTheMovedSignal(t *testing.T) {
	Reset()
	s := make([]Signal[int], 1, 1)
	s[0].Set(1)
	s = append(s, Signal[int]{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("the moved signal gave no panic")
		}
	}()
	s[0].Set(2)
}

// TestAnUnusedSignalMovesWithNoPanic keeps the normal case. A program builds a
// list of signals before it reads any 1 of them.
func TestAnUnusedSignalMovesWithNoPanic(t *testing.T) {
	Reset()
	s := make([]Signal[int], 1, 1)
	s = append(s, Signal[int]{})
	s[0].Set(1)
	if got := s[0].Peek(); got != 1 {
		t.Fatalf("the signal holds %d, and 1 is correct", got)
	}
}

// TestAPointerElementHoldsTheRule gives the form that a container must take.
func TestAPointerElementHoldsTheRule(t *testing.T) {
	Reset()
	s := []*Signal[int]{NewSignal(0)}
	runs := 0
	stop := Effect(func() { s[0].Get(); runs++ })
	s = append(s, NewSignal(0))
	s[0].Set(1)
	if runs != 2 {
		t.Fatalf("the effect ran %d times, and 2 is correct", runs)
	}
	stop()
}

// TestAStoreThatMovesPanics applies the rule to the second reactive value.
func TestAStoreThatMovesPanics(t *testing.T) {
	Reset()
	list := make([]Store, 1, 1)
	list[0].Notify()
	list = append(list, Store{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("the moved store gave no panic")
		}
	}()
	list[0].Notify()
}
