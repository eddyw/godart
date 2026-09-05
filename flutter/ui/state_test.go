package ui_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"gart.dev/ui"
	"gart.dev/ui/material"
)

// cart holds the shape of ChangeNotifier: a method writes a field, and it calls
// Notify.
type cart struct {
	ui.Store
	items []string
}

func (c *cart) Add(item string) {
	c.items = append(c.items, item)
	c.Notify()
}

func (c *cart) Items() []string {
	c.Track()
	return c.items
}

func (c *cart) Build() ui.Node {
	return ui.Watch{Build: func() ui.Node {
		items := c.Items()
		kids := make([]ui.Node, len(items))
		for i, item := range items {
			kids[i] = ui.Text{Data: item}
		}
		return ui.Column{Children: kids}
	}}
}

func TestStoreDrivesABoundary(t *testing.T) {
	ui.Reset()
	c := &cart{}
	h := newHost(c)
	if got := h.Frame(); got != "" {
		t.Fatalf("the first frame gave %q, want an empty screen", got)
	}
	ui.TakeDirty()

	c.Add("apple")
	if got := ui.TakeDirty(); len(got) != 1 {
		t.Fatalf("Add marked %v, want 1 id", got)
	}
	h.Frame()
	if got, want := h.Screen(), "apple"; got != want {
		t.Fatalf("the frame gave %q, want %q", got, want)
	}
}

func TestStoreNeedsNoBoundaryForATest(t *testing.T) {
	c := &cart{}
	c.Add("apple")
	c.Add("pear")
	if got := c.Items(); !slices.Equal(got, []string{"apple", "pear"}) {
		t.Fatalf("Items gave %v, want [apple pear]", got)
	}
}

// The reducer shape. The state is a value, and the action is a type. reduce is
// a pure function, thus a table test reads it with no UI.

type todoState struct {
	items []string
	done  int
}

type addTodo struct{ Text string }
type finishTodo struct{}
type clearTodos struct{}

func reduce(s todoState, a any) todoState {
	switch v := a.(type) {
	case addTodo:
		s.items = append(slices.Clone(s.items), v.Text)
	case finishTodo:
		s.done++
	case clearTodos:
		s.items = nil
		s.done = 0
	}
	return s
}

func TestReduceIsPure(t *testing.T) {
	cases := []struct {
		name   string
		start  todoState
		action any
		items  []string
		done   int
	}{
		{"add to an empty state", todoState{}, addTodo{"buy milk"}, []string{"buy milk"}, 0},
		{"add to a held state", todoState{items: []string{"a"}}, addTodo{"b"}, []string{"a", "b"}, 0},
		{"finish", todoState{items: []string{"a"}}, finishTodo{}, []string{"a"}, 1},
		{"clear", todoState{items: []string{"a"}, done: 1}, clearTodos{}, nil, 0},
		{"an unknown action changes nothing", todoState{done: 2}, 42, nil, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reduce(c.start, c.action)
			if !slices.Equal(got.items, c.items) {
				t.Errorf("the items gave %v, want %v", got.items, c.items)
			}
			if got.done != c.done {
				t.Errorf("done gave %d, want %d", got.done, c.done)
			}
		})
	}
}

func TestReduceLeavesTheStartStateWhole(t *testing.T) {
	start := todoState{items: []string{"a"}}
	reduce(start, addTodo{"b"})
	if len(start.items) != 1 {
		t.Fatalf("reduce changed the start state, it holds %v", start.items)
	}
}

func TestReducerDrivesABoundary(t *testing.T) {
	ui.Reset()
	r := ui.NewReducer(todoState{}, reduce)
	id := ui.NewWatch()
	ui.RunWatch(id, func() ui.Node {
		return ui.Text{Data: fmt.Sprint(len(r.State().items))}
	})
	ui.TakeDirty()

	r.Do(addTodo{"a"})
	if got := ui.TakeDirty(); len(got) != 1 || got[0] != id {
		t.Fatalf("Do marked %v, want [%d]", got, id)
	}
	if got := r.Peek().items; !slices.Equal(got, []string{"a"}) {
		t.Fatalf("the state holds %v, want [a]", got)
	}
	if got := len(r.Actions()); got != 1 {
		t.Fatalf("Actions gave %d entries, want 1", got)
	}
}

// The command shape.

var errBoom = errors.New("boom")

func TestCommandHoldsBusyAndTheError(t *testing.T) {
	ui.Reset()
	fail := true
	cmd := ui.NewCommand(func() error {
		if fail {
			return errBoom
		}
		return nil
	})

	if cmd.Busy() {
		t.Fatal("a new command reports busy")
	}
	cmd.Run()
	if !cmd.Busy() {
		t.Fatal("the command does not report busy after Run")
	}
	cmd.Wait()
	if cmd.Busy() {
		t.Fatal("the command reports busy after Wait")
	}
	if !errors.Is(cmd.Err(), errBoom) {
		t.Fatalf("Err gave %v, want %v", cmd.Err(), errBoom)
	}
	if !cmd.Done() {
		t.Fatal("Done gave false after a run")
	}

	fail = false
	cmd.Run()
	cmd.Wait()
	if cmd.Err() != nil {
		t.Fatalf("Err gave %v after a good run, want nil", cmd.Err())
	}
}

func TestCommandRefusesASecondRun(t *testing.T) {
	ui.Reset()
	calls := 0
	block := make(chan struct{})
	cmd := ui.NewCommand(func() error {
		calls++
		<-block
		return nil
	})
	cmd.Run()
	cmd.Run()
	close(block)
	cmd.Wait()
	if calls != 1 {
		t.Fatalf("the work ran %d times, want 1", calls)
	}
}

func TestCommand1TakesAnArgument(t *testing.T) {
	ui.Reset()
	var seen int
	cmd := ui.NewCommand1(func(n int) error {
		seen = n
		return nil
	})
	cmd.Run(42)
	cmd.Wait()
	if seen != 42 {
		t.Fatalf("the work saw %d, want 42", seen)
	}
	if cmd.Err() != nil {
		t.Fatalf("Err gave %v, want nil", cmd.Err())
	}
}

// feed is the second proof program: a screen that an HTTP call fills. The work
// blocks on a goroutine, and the build never waits.
type feed struct {
	items ui.Signal[[]string]
	load  *ui.Command
}

func newFeed(fetch func() ([]string, error)) *feed {
	f := &feed{}
	f.load = ui.NewCommand(func() error {
		items, err := fetch()
		if err != nil {
			return err
		}
		ui.Post(func() { f.items.Set(items) })
		return nil
	})
	return f
}

func (f *feed) Build() ui.Node {
	return material.Scaffold{
		AppBar: material.AppBar{
			Title:   ui.Text{Data: "Feed"},
			Actions: ui.Kids(material.IconButton{Icon: ui.Icon{Icon: "refresh"}, OnPressed: f.load.Run}),
		},
		Body: ui.Watch{Build: func() ui.Node {
			if f.load.Busy() {
				return ui.Center{Child: material.CircularProgressIndicator{}}
			}
			if err := f.load.Err(); err != nil {
				return ui.Center{Child: ui.Text{Data: "error: " + err.Error()}}
			}
			items := f.items.Get()
			kids := make([]ui.Node, len(items))
			for i, item := range items {
				kids[i] = ui.Text{Data: item}
			}
			return ui.ListView{Children: kids}
		}},
	}
}

func TestFeedLoadsWithNoFlutter(t *testing.T) {
	ui.Reset()
	f := newFeed(func() ([]string, error) { return []string{"one", "two"}, nil })
	h := newHost(f)

	if got, want := h.Frame(), "Feed <refresh>"; got != want {
		t.Fatalf("the first frame gave %q, want %q", got, want)
	}

	h.Tap("refresh")
	h.Settle()
	if got, want := h.Screen(), "Feed <refresh> [busy]"; got != want {
		t.Fatalf("the frame during the load gave %q, want %q", got, want)
	}

	f.load.Wait()
	h.Settle()
	if got, want := h.Screen(), "Feed <refresh> one two"; got != want {
		t.Fatalf("the frame after the load gave %q, want %q", got, want)
	}
}

func TestFeedShowsTheError(t *testing.T) {
	ui.Reset()
	f := newFeed(func() ([]string, error) { return nil, errBoom })
	h := newHost(f)
	h.Frame()

	h.Tap("refresh")
	f.load.Wait()
	h.Settle()
	if got, want := h.Screen(), "Feed <refresh> error: boom"; got != want {
		t.Fatalf("the frame after the failure gave %q, want %q", got, want)
	}
}

func TestPumpGivesTheCount(t *testing.T) {
	ui.Reset()
	ran := 0
	ui.Post(func() { ran++ })
	ui.Post(func() { ran++ })
	if got := ui.Pump(); got != 2 {
		t.Fatalf("Pump gave %d, want 2", got)
	}
	if ran != 2 {
		t.Fatalf("%d functions ran, want 2", ran)
	}
	if got := ui.Pump(); got != 0 {
		t.Fatalf("the second Pump gave %d, want 0", got)
	}
}
