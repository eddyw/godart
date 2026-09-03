package ui

import "testing"

// TestAComputedOfABoundaryDiesWithIt names the leak of a Computed that a build
// creates. The boundary owns the scope, thus a rebuild and a dispose each
// release it.
func TestAComputedOfABoundaryDiesWithIt(t *testing.T) {
	Reset()
	src := NewSignal(1)
	id := NewWatch()
	for i := 0; i < 5; i++ {
		RunWatch(id, func() Node {
			c := NewComputed(func() int { return src.Get() * 2 })
			_ = c.Get()
			return nil
		})
	}
	if got := liveScopes(); got != 2 {
		t.Fatalf("after 5 builds the graph holds %d scopes, and 2 is correct", got)
	}
	DisposeWatch(id)
	if got := liveScopes(); got != 0 {
		t.Fatalf("after the dispose the graph holds %d scopes, and 0 is correct", got)
	}
}

// TestAComputedOutsideABoundaryNeedsDrop keeps the manual rule. No scope owns
// such a value, thus the program releases it.
func TestAComputedOutsideABoundaryNeedsDrop(t *testing.T) {
	Reset()
	src := NewSignal(1)
	c := NewComputed(func() int { return src.Get() * 2 })
	_ = c.Get()
	if got := liveScopes(); got != 1 {
		t.Fatalf("the graph holds %d scopes, and 1 is correct", got)
	}
	c.Drop()
	if got := liveScopes(); got != 0 {
		t.Fatalf("after the drop the graph holds %d scopes, and 0 is correct", got)
	}
}

// TestADropTwiceKeepsTheIDList proves that a second Drop frees no identifier a
// second time. 2 scopes with 1 identifier would lose each mark of the first.
func TestADropTwiceKeepsTheIDList(t *testing.T) {
	Reset()
	c := NewComputed(func() int { return 1 })
	_ = c.Get()
	c.Drop()
	c.Drop()
	if got := len(freeIDs); got != 1 {
		t.Fatalf("the free list holds %d identifiers, and 1 is correct", got)
	}
}

// TestAnEffectOfABoundaryStopsWithIt applies the owner rule to an effect.
func TestAnEffectOfABoundaryStopsWithIt(t *testing.T) {
	Reset()
	src := NewSignal(0)
	runs := 0
	id := NewWatch()
	RunWatch(id, func() Node {
		Effect(func() { src.Get(); runs++ })
		return nil
	})
	if runs != 1 {
		t.Fatalf("the effect ran %d times, and 1 is correct", runs)
	}
	DisposeWatch(id)
	src.Set(1)
	if runs != 1 {
		t.Fatalf("the effect ran %d times after the dispose, and 1 is correct", runs)
	}
}

// liveScopes counts the scopes that the graph holds.
func liveScopes() int {
	n := 0
	for _, s := range scopes {
		if s != nil && s.alive {
			n++
		}
	}
	return n
}
