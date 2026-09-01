package app

import (
	"strings"
	"testing"

	"gart.dev/ui"
)

// The whole screen runs under `go test`. No Flutter, no Dart, and no gart.

func TestGreetingFollowsTheName(t *testing.T) {
	ui.Reset()
	a := New()
	if got := a.Greeting(); got != "Hello" {
		t.Fatalf("Greeting gave %q, want %q", got, "Hello")
	}
	a.SetName("Ada")
	if got := a.Greeting(); got != "Hello Ada" {
		t.Fatalf("Greeting gave %q, want %q", got, "Hello Ada")
	}
}

func TestTheCounterCountsAndResets(t *testing.T) {
	ui.Reset()
	a := New()
	a.Add()
	a.Add()
	if got := a.count.Peek(); got != 2 {
		t.Fatalf("the count is %d, want 2", got)
	}
	a.Reset()
	if got := a.count.Peek(); got != 0 {
		t.Fatalf("the count is %d after a reset, want 0", got)
	}
}

func TestTheCountMarksOnlyTheCounterBoundary(t *testing.T) {
	ui.Reset()
	a := New()
	greet := ui.NewWatch()
	count := ui.NewWatch()
	ui.RunWatch(greet, a.greetingRow)
	ui.RunWatch(count, a.counterRow)
	ui.TakeDirty()

	a.Add()
	got := ui.TakeDirty()
	if len(got) != 1 || got[0] != count {
		t.Fatalf("Add marked %v, want [%d]", got, count)
	}

	a.SetName("Ada")
	got = ui.TakeDirty()
	if len(got) != 1 || got[0] != greet {
		t.Fatalf("SetName marked %v, want [%d]", got, greet)
	}
}

func TestTheListLoads(t *testing.T) {
	ui.Reset()
	a := New()
	if !strings.Contains(bodyText(t, a), "press refresh") {
		t.Fatalf("the first body gave %q", bodyText(t, a))
	}

	a.load.Run()
	if !a.load.Busy() {
		t.Fatal("the command does not report busy after Run")
	}
	a.load.Wait()
	if a.load.Err() != nil {
		t.Fatalf("the load gave the error %v", a.load.Err())
	}
	list, ok := a.listBody().(ui.ListViewBuilder)
	if !ok {
		t.Fatalf("the body is %T, want a ListViewBuilder", a.listBody())
	}
	if list.Count != 100000 {
		t.Fatalf("the list holds %d rows, want 100000", list.Count)
	}
	row, ok := list.Item(9).(ui.Keyed)
	if !ok {
		t.Fatalf("row 9 is %T, want a Keyed", list.Item(9))
	}
	if row.Key != "9" {
		t.Fatalf("row 9 holds the key %q, want %q", row.Key, "9")
	}
}

func TestTheListShowsTheError(t *testing.T) {
	ui.Reset()
	a := New()
	a.SetFail(true)
	a.load.Run()
	a.load.Wait()
	if got := bodyText(t, a); got != "error: the server gave 503" {
		t.Fatalf("the body gave %q", got)
	}
}

// bodyText gives the text of a body that holds 1 centered line.
func bodyText(t *testing.T, a *App) string {
	t.Helper()
	center, ok := a.listBody().(ui.Center)
	if !ok {
		t.Fatalf("the body is %T, want a Center", a.listBody())
	}
	text, ok := center.Child.(ui.Text)
	if !ok {
		t.Fatalf("the child is %T, want a Text", center.Child)
	}
	return text.Data
}
