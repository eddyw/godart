package ui_test

import (
	"fmt"
	"strings"

	"gart.dev/ui"
)

// host is a test host. It does the work that the Dart host does: it walks the
// Go tree, it gives each Watch a boundary that survives a rebuild, and it
// rebuilds only the boundaries that TakeDirty names.
//
// The real host keys a boundary on the Flutter element. This one keys it on the
// order of the walk, which is enough for a test.
type host struct {
	root    ui.Root
	ids     []int
	next    int
	out     strings.Builder
	taps    map[string]func()
	changes map[string]func(string)
	builds  int
	watches int
}

func newHost(root ui.Root) *host {
	return &host{root: root}
}

// Frame builds the whole tree.
func (h *host) Frame() string {
	h.next = 0
	h.taps = map[string]func(){}
	h.changes = map[string]func(string){}
	h.out.Reset()
	h.builds++
	h.walk(h.root.Build())
	return h.out.String()
}

// Screen gives the text of the last frame.
func (h *host) Screen() string { return h.out.String() }

// Settle runs each pending rebuild until no boundary stays dirty. It gives the
// number of boundaries that rebuilt, and it counts the whole tree as 1.
func (h *host) Settle() int {
	n := 0
	for range 100 {
		dirty := ui.TakeDirty()
		if len(dirty) == 0 {
			return n
		}
		n += len(dirty)
		h.Frame()
	}
	panic("the tree never settled")
}

// Tap runs the handler of the button that holds the label.
func (h *host) Tap(label string) {
	fn, ok := h.taps[label]
	if !ok {
		panic("no button with the label " + label)
	}
	fn()
}

// Type runs the change handler of the field that holds the label.
func (h *host) Type(label, text string) {
	fn, ok := h.changes[label]
	if !ok {
		panic("no field with the label " + label)
	}
	fn(text)
}

func (h *host) write(text string) {
	if h.out.Len() > 0 {
		h.out.WriteString(" ")
	}
	h.out.WriteString(text)
}

func (h *host) each(list []ui.Node) {
	for _, n := range list {
		h.walk(n)
	}
}

func (h *host) walk(n ui.Node) {
	if n == nil {
		return
	}
	switch v := n.(type) {
	case ui.Watch:
		i := h.next
		h.next++
		if i == len(h.ids) {
			h.ids = append(h.ids, ui.NewWatch())
		}
		h.watches++
		h.walk(ui.RunWatch(h.ids[i], v.Build))
	case ui.Scaffold:
		h.walk(v.AppBar)
		h.walk(v.Body)
		h.walk(v.Fab)
	case ui.AppBar:
		h.walk(v.Title)
		h.each(v.Actions)
	case ui.Column:
		h.each(v.Children)
	case ui.Row:
		h.each(v.Children)
	case ui.Stack:
		h.each(v.Children)
	case ui.ListView:
		h.each(v.Children)
	case ui.ListViewBuilder:
		for i := range v.Count {
			h.walk(v.Item(i))
		}
	case ui.Center:
		h.walk(v.Child)
	case ui.Padding:
		h.walk(v.Child)
	case ui.SizedBox:
		h.walk(v.Child)
	case ui.Expanded:
		h.walk(v.Child)
	case ui.Container:
		h.walk(v.Child)
	case ui.Card:
		h.walk(v.Child)
	case ui.Keyed:
		h.walk(v.Child)
	case ui.InkWell:
		h.walk(v.Child)
	case ui.Text:
		h.write(v.Data)
	case ui.Icon:
		h.write("[" + v.Name + "]")
	case ui.Progress:
		h.write("[busy]")
	case ui.Divider:
		h.write("---")
	case ui.ElevatedButton:
		h.button(v.Child, v.OnPressed)
	case ui.TextButton:
		h.button(v.Child, v.OnPressed)
	case ui.IconButton:
		h.write("<" + v.Icon + ">")
		if v.OnPressed != nil {
			h.taps[v.Icon] = v.OnPressed
		}
	case ui.TextField:
		h.write(fmt.Sprintf("{%s=%q}", v.Label, v.Value))
		if v.OnChanged != nil {
			h.changes[v.Label] = v.OnChanged
		}
	case ui.Switch:
		h.write(fmt.Sprintf("(%t)", v.Value))
	case ui.Checkbox:
		h.write(fmt.Sprintf("[%t]", v.Value))
	default:
		panic(fmt.Sprintf("the test host holds no case for %T", n))
	}
}

func (h *host) button(child ui.Node, tap func()) {
	label := ""
	if t, ok := child.(ui.Text); ok {
		label = t.Data
	}
	h.write("<" + label + ">")
	if tap != nil {
		h.taps[label] = tap
	}
}
