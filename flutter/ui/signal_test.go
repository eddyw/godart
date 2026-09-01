package ui_test

import (
	"testing"

	"gart.dev/ui"
)

func TestSignalHoldsTheZeroValue(t *testing.T) {
	ui.Reset()
	var s ui.Signal[int]
	if got := s.Get(); got != 0 {
		t.Fatalf("Get gave %d, want 0", got)
	}
	s.Set(7)
	if got := s.Get(); got != 7 {
		t.Fatalf("Get gave %d, want 7", got)
	}
}

func TestEffectRunsOnAChangeOfAReadSignal(t *testing.T) {
	ui.Reset()
	var a, b ui.Signal[int]
	runs := 0
	stop := ui.Effect(func() {
		runs++
		a.Get()
	})
	if runs != 1 {
		t.Fatalf("the effect ran %d times, want 1", runs)
	}

	a.Set(1)
	if runs != 2 {
		t.Fatalf("the effect ran %d times after a write of a, want 2", runs)
	}

	b.Set(1)
	if runs != 2 {
		t.Fatalf("the effect ran %d times after a write of b, want 2", runs)
	}

	stop()
	a.Set(2)
	if runs != 2 {
		t.Fatalf("the effect ran %d times after stop, want 2", runs)
	}
}

func TestEffectDropsAnOldDependency(t *testing.T) {
	ui.Reset()
	var pick ui.Signal[bool]
	var a, b ui.Signal[int]
	runs := 0
	ui.Effect(func() {
		runs++
		if pick.Get() {
			a.Get()
		} else {
			b.Get()
		}
	})
	// The first run read pick and b.
	a.Set(1)
	if runs != 1 {
		t.Fatalf("a write of the unread signal ran the effect, runs=%d", runs)
	}
	b.Set(1)
	if runs != 2 {
		t.Fatalf("a write of the read signal did not run the effect, runs=%d", runs)
	}
	pick.Set(true)
	if runs != 3 {
		t.Fatalf("a write of pick did not run the effect, runs=%d", runs)
	}
	// The run after pick read a, and it dropped b.
	b.Set(2)
	if runs != 3 {
		t.Fatalf("the effect kept the old dependency, runs=%d", runs)
	}
	a.Set(2)
	if runs != 4 {
		t.Fatalf("the effect missed the new dependency, runs=%d", runs)
	}
}

func TestComputedRunsLateAndCaches(t *testing.T) {
	ui.Reset()
	var a ui.Signal[int]
	calls := 0
	double := ui.NewComputed(func() int {
		calls++
		return a.Get() * 2
	})
	if calls != 0 {
		t.Fatalf("the computed ran before a read, calls=%d", calls)
	}
	if got := double.Get(); got != 0 {
		t.Fatalf("Get gave %d, want 0", got)
	}
	if got := double.Get(); got != 0 || calls != 1 {
		t.Fatalf("the second read ran the function again, calls=%d", calls)
	}
	a.Set(4)
	if got := double.Get(); got != 8 {
		t.Fatalf("Get gave %d, want 8", got)
	}
	if calls != 2 {
		t.Fatalf("the function ran %d times, want 2", calls)
	}
}

func TestComputedFeedsAnEffect(t *testing.T) {
	ui.Reset()
	var a ui.Signal[int]
	sum := ui.NewComputed(func() int { return a.Get() + 1 })
	seen := 0
	ui.Effect(func() { seen = sum.Get() })
	if seen != 1 {
		t.Fatalf("the effect saw %d, want 1", seen)
	}
	a.Set(10)
	if seen != 11 {
		t.Fatalf("the effect saw %d, want 11", seen)
	}
}

func TestBatchRunsAScopeOneTime(t *testing.T) {
	ui.Reset()
	var a, b ui.Signal[int]
	runs := 0
	ui.Effect(func() {
		runs++
		a.Get()
		b.Get()
	})
	ui.Batch(func() {
		a.Set(1)
		b.Set(1)
	})
	if runs != 2 {
		t.Fatalf("the effect ran %d times, want 2", runs)
	}
}

func TestUntrackedRecordsNoDependency(t *testing.T) {
	ui.Reset()
	var a ui.Signal[int]
	runs := 0
	ui.Effect(func() {
		runs++
		ui.Untracked(func() { a.Get() })
	})
	a.Set(1)
	if runs != 1 {
		t.Fatalf("the effect ran %d times, want 1", runs)
	}
}

func TestPeekRecordsNoDependency(t *testing.T) {
	ui.Reset()
	var a ui.Signal[int]
	runs := 0
	ui.Effect(func() {
		runs++
		a.Peek()
	})
	a.Set(1)
	if runs != 1 {
		t.Fatalf("the effect ran %d times, want 1", runs)
	}
}

func TestUpdateReadsAndWrites(t *testing.T) {
	ui.Reset()
	s := ui.NewSignal(2)
	s.Update(func(v int) int { return v * 5 })
	if got := s.Peek(); got != 10 {
		t.Fatalf("Peek gave %d, want 10", got)
	}
}

func TestTrackedReportsTheScope(t *testing.T) {
	ui.Reset()
	if ui.Tracked() {
		t.Fatal("Tracked gave true outside a scope")
	}
	inside := false
	ui.Effect(func() { inside = ui.Tracked() })
	if !inside {
		t.Fatal("Tracked gave false inside an effect")
	}
}
