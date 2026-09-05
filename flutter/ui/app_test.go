package ui_test

import (
	"fmt"
	"slices"
	"testing"

	"gart.dev/ui"
	"gart.dev/ui/material"
)

// counter is the proof program of milestone 1. It holds its state in a signal,
// and it needs no Flutter and no Dart to run.
type counter struct {
	n ui.Signal[int]
}

func (c *counter) Add() { c.n.Update(func(v int) int { return v + 1 }) }

func (c *counter) Build() ui.Node {
	return material.Scaffold{
		AppBar: material.AppBar{Title: ui.Text{Data: "Counter"}},
		Body: ui.Center{Child: ui.Watch{Build: func() ui.Node {
			return ui.Text{Data: fmt.Sprintf("count %d", c.n.Get())}
		}}},
		FloatingActionButton: material.IconButton{
			Icon:      ui.Icon{Icon: "add"},
			OnPressed: c.Add,
		},
	}
}

func TestCounterRunsWithNoFlutter(t *testing.T) {
	ui.Reset()
	h := newHost(&counter{})

	if got, want := h.Frame(), "Counter count 0 <add>"; got != want {
		t.Fatalf("the first frame gave %q, want %q", got, want)
	}

	h.Tap("add")
	h.Settle()
	if got, want := h.Screen(), "Counter count 1 <add>"; got != want {
		t.Fatalf("the frame after 1 tap gave %q, want %q", got, want)
	}

	h.Tap("add")
	h.Tap("add")
	h.Settle()
	if got, want := h.Screen(), "Counter count 3 <add>"; got != want {
		t.Fatalf("the frame after 3 taps gave %q, want %q", got, want)
	}
}

func TestOneWriteMarksOneBoundary(t *testing.T) {
	ui.Reset()
	var a, b ui.Signal[int]
	idA := ui.NewWatch()
	idB := ui.NewWatch()
	ui.RunWatch(idA, func() ui.Node { return ui.Text{Data: fmt.Sprint(a.Get())} })
	ui.RunWatch(idB, func() ui.Node { return ui.Text{Data: fmt.Sprint(b.Get())} })
	ui.TakeDirty()

	a.Set(1)
	got := ui.TakeDirty()
	if len(got) != 1 || got[0] != idA {
		t.Fatalf("TakeDirty gave %v, want [%d]", got, idA)
	}

	b.Set(1)
	got = ui.TakeDirty()
	if len(got) != 1 || got[0] != idB {
		t.Fatalf("TakeDirty gave %v, want [%d]", got, idB)
	}
}

func TestOneRowOfManyRebuildsAlone(t *testing.T) {
	ui.Reset()
	const rows = 500
	cells := make([]ui.Signal[int], rows)
	ids := make([]int, rows)
	for i := range rows {
		ids[i] = ui.NewWatch()
		ui.RunWatch(ids[i], func() ui.Node {
			return ui.Text{Data: fmt.Sprint(cells[i].Get())}
		})
	}
	ui.TakeDirty()

	cells[317].Set(9)
	got := ui.TakeDirty()
	if len(got) != 1 || got[0] != ids[317] {
		t.Fatalf("a write of 1 row of %d marked %v, want [%d]", rows, got, ids[317])
	}
}

func TestDirtyMarksTheWholeTree(t *testing.T) {
	ui.Reset()
	ui.Dirty()
	got := ui.TakeDirty()
	if len(got) != 1 || got[0] != ui.RootID {
		t.Fatalf("TakeDirty gave %v, want [%d]", got, ui.RootID)
	}
	if rest := ui.TakeDirty(); len(rest) != 0 {
		t.Fatalf("TakeDirty gave %v after a take, want an empty list", rest)
	}
}

func TestTakeDirtyHoldsEachIDOneTime(t *testing.T) {
	ui.Reset()
	var a ui.Signal[int]
	id := ui.NewWatch()
	ui.RunWatch(id, func() ui.Node { return ui.Text{Data: fmt.Sprint(a.Get())} })
	ui.TakeDirty()

	a.Set(1)
	a.Set(2)
	a.Set(3)
	if got := ui.TakeDirty(); len(got) != 1 {
		t.Fatalf("3 writes gave %v, want 1 id", got)
	}
}

func TestDisposeStopsTheBoundaryAndReusesTheID(t *testing.T) {
	ui.Reset()
	var a ui.Signal[int]
	id := ui.NewWatch()
	ui.RunWatch(id, func() ui.Node { return ui.Text{Data: fmt.Sprint(a.Get())} })
	ui.TakeDirty()

	ui.DisposeWatch(id)
	a.Set(1)
	if got := ui.TakeDirty(); len(got) != 0 {
		t.Fatalf("a dropped boundary gave %v, want an empty list", got)
	}
	if next := ui.NewWatch(); next != id {
		t.Fatalf("the next boundary took the id %d, want the free id %d", next, id)
	}
}

func TestThemeRebuildsOnlyTheReaderOfTheTheme(t *testing.T) {
	ui.Reset()
	var a ui.Signal[int]
	idPlain := ui.NewWatch()
	idThemed := ui.NewWatch()
	ui.RunWatch(idPlain, func() ui.Node { return ui.Text{Data: fmt.Sprint(a.Get())} })
	ui.RunWatch(idThemed, func() ui.Node {
		return ui.Text{Data: "x", Style: &ui.TextStyle{Size: ui.P(ui.Theme().TextSize)}}
	})
	ui.TakeDirty()

	ui.SetTheme(ui.ThemeData{Dark: true, TextSize: 18})
	got := ui.TakeDirty()
	if len(got) != 1 || got[0] != idThemed {
		t.Fatalf("a theme change marked %v, want [%d]", got, idThemed)
	}
	if ui.Theme().TextSize != 18 {
		t.Fatalf("Theme gave %v, want a text size of 18", ui.Theme())
	}
}

func TestMediaIsASignal(t *testing.T) {
	ui.Reset()
	id := ui.NewWatch()
	ui.RunWatch(id, func() ui.Node {
		return ui.SizedBox{Width: ui.P(ui.Media().Width)}
	})
	ui.TakeDirty()
	ui.SetMedia(ui.MediaData{Width: 390, Height: 844})
	if got := ui.TakeDirty(); !slices.Contains(got, id) {
		t.Fatalf("a media change marked %v, want the id %d", got, id)
	}
}

// listApp proves the lazy list and the per row boundary together.
type listApp struct {
	rows []ui.Signal[string]
}

func newListApp(n int) *listApp {
	a := &listApp{rows: make([]ui.Signal[string], n)}
	for i := range a.rows {
		a.rows[i].Set(fmt.Sprintf("row %d", i))
	}
	return a
}

func (a *listApp) Build() ui.Node {
	return ui.ListViewBuilder{
		ItemCount: ui.P(len(a.rows)),
		ItemBuilder: func(i int) ui.Node {
			return ui.Keyed{Key: fmt.Sprint(i), Child: ui.Watch{Build: func() ui.Node {
				return ui.Text{Data: a.rows[i].Get()}
			}}}
		},
	}
}

func TestLazyListWithOneBoundaryForEachRow(t *testing.T) {
	ui.Reset()
	app := newListApp(4)
	h := newHost(app)
	if got, want := h.Frame(), "row 0 row 1 row 2 row 3"; got != want {
		t.Fatalf("the first frame gave %q, want %q", got, want)
	}
	ui.TakeDirty()

	app.rows[2].Set("changed")
	if got := ui.TakeDirty(); len(got) != 1 {
		t.Fatalf("a write of 1 row marked %v, want 1 id", got)
	}
	h.Frame()
	if got, want := h.Screen(), "row 0 row 1 changed row 3"; got != want {
		t.Fatalf("the frame gave %q, want %q", got, want)
	}
}
