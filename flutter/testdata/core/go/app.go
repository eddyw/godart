// Package app holds the whole application. It reads no Dart name, and a
// `go test` runs each part of it with no Flutter.
package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"example.dev/gocore/bindings/material"
	"gart.dev/ui"
)

// App holds the state of the screen. Each field is a signal, thus a write of 1
// field rebuilds only the part of the screen that reads that field.
type App struct {
	count ui.Signal[int]
	name  ui.Signal[string]
	rows  ui.Signal[[]string]
	fail  ui.Signal[bool]
	load  *ui.Command

	// shared holds the answer of the host after a share, and share runs the
	// call. A Go tree carries no call of Dart, thus ui.Call reaches the host.
	shared ui.Signal[string]
	share  *ui.Command

	// The 4 signals below hold the note section. note decides whether the
	// field stands in the tree, and the other 3 hold the state that the host
	// reports through a key of flutter/ui/identity.go.
	note  ui.Signal[bool]
	focus ui.Signal[bool]
	marks ui.Signal[string]
	at    ui.Signal[float64]
}

// New gives an application with an empty state.
func New() *App {
	a := &App{}
	a.load = ui.NewCommand(a.fetch)
	a.share = ui.NewCommand(a.shout)
	a.note.Set(true)
	return a
}

// fetch stands for the call of an HTTP API. It runs on a goroutine of its own,
// thus it may block. A build never waits.
func (a *App) fetch() error {
	time.Sleep(200 * time.Millisecond)
	if a.fail.Peek() {
		return errors.New("the server gave 503")
	}
	rows := make([]string, 100000)
	for i := range rows {
		rows[i] = fmt.Sprintf("row %d", i)
	}
	ui.Post(func() { a.rows.Set(rows) })
	return nil
}

// shout asks the host to share the greeting, and it holds the answer of the
// host. The call parks this goroutine, and the answer of the host wakes it.
func (a *App) shout() error {
	answer, err := ui.Call("share", a.Greeting())
	if err != nil {
		return err
	}
	ui.Post(func() { a.shared.Set(answer) })
	return nil
}

// Add counts 1 up.
func (a *App) Add() { a.count.Update(func(n int) int { return n + 1 }) }

// Reset counts back to 0.
func (a *App) Reset() { a.count.Set(0) }

// SetName writes the name that the greeting reads.
func (a *App) SetName(s string) { a.name.Set(s) }

// SetFail decides whether the next load fails. The switch of the screen writes
// it, and a test writes it too.
func (a *App) SetFail(v bool) { a.fail.Set(v) }

// ToggleNote takes the note field out of the tree, or it puts the field back.
// The host disposes the FocusNode of the key "note" when the field leaves.
func (a *App) ToggleNote() { a.note.Update(func(v bool) bool { return !v }) }

// SetNoteFocus records whether the note field holds the focus. The host calls
// it through the field OnFocus of the key.
func (a *App) SetNoteFocus(v bool) { a.focus.Set(v) }

// SetMarks records the states of the toggle button, for example "pressed".
func (a *App) SetMarks(on []string) { a.marks.Set(strings.Join(on, " ")) }

// SetOffset records the scroll offset of the list.
func (a *App) SetOffset(v float64) { a.at.Set(v) }

// Report gives the state that the host reported back for the note section.
func (a *App) Report() string {
	focus := "idle"
	if a.focus.Get() {
		focus = "focused"
	}
	marks := a.marks.Get()
	if marks == "" {
		marks = "none"
	}
	return "note " + focus + ", button " + marks
}

// Greeting gives the line that the name drives.
func (a *App) Greeting() string {
	name := a.name.Get()
	if name == "" {
		return "Hello"
	}
	return "Hello " + name
}

// Build gives the tree of the screen.
func (a *App) Build() ui.Node {
	return material.Scaffold{
		AppBar: material.AppBar{
			Title: ui.Text{Data: "This is Go!"},
			Actions: ui.Kids(
				material.IconButton{
					Icon:      ui.Icon{Icon: "refresh"},
					Tooltip:   ui.P("load the list"),
					OnPressed: a.load.Run,
				},
				material.IconButton{
					Icon:      ui.Icon{Icon: "share"},
					Tooltip:   ui.P("share the greeting"),
					OnPressed: a.share.Run,
				},
			),
		},
		Body: ui.Column{Children: ui.Kids(
			ui.Padding{Padding: ui.AllSides(16), Child: ui.Watch{Build: a.greetingRow}},
			ui.Padding{
				Padding: ui.Sides(16, 0),
				Child: ui.TextInput{
					Label:     "your name",
					Hint:      "type here",
					Value:     a.name.Peek(),
					OnChanged: a.SetName,
				},
			},
			ui.Padding{Padding: ui.AllSides(16), Child: ui.Watch{Build: a.counterRow}},
			ui.Padding{Padding: ui.Sides(16, 0), Child: ui.Watch{Build: a.shareRow}},
			ui.Padding{Padding: ui.Sides(16, 0), Child: ui.Watch{Build: a.noteSection}},
			ui.Padding{
				Padding: ui.Sides(16, 0),
				Child: ui.Row{Children: ui.Kids(
					ui.Text{Data: "make the load fail"},
					ui.Spacer{},
					ui.Watch{Build: a.failSwitch},
				)},
			},
			material.Divider{},
			ui.Expanded{Child: ui.Watch{Build: a.listPanel}},
		)},
		FloatingActionButton: material.IconButton{
			Icon:      ui.Icon{Icon: "add"},
			Tooltip:   ui.P("count 1 up"),
			OnPressed: a.Add,
		},
	}
}

// pixelPNG holds 1 transparent pixel in the PNG format. It stands for the
// bytes of an image that a Go program holds, for example the body of an HTTP
// answer or the content of a file.
var pixelPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func (a *App) greetingRow() ui.Node {
	return ui.Container{
		Padding: ui.P(ui.Sides(16, 12)),
		Decoration: &ui.Decoration{
			Radius: ui.P(ui.AllCorners(12)),
			Border: &ui.BorderSide{
				Color: ui.P(ui.RGB(0x90, 0xa4, 0xae)),
				Width: ui.P(1.0),
			},
			Gradient: &ui.Gradient{
				Kind:   "linear",
				Colors: []ui.Color{ui.RGB(0xe3, 0xf2, 0xfd), ui.RGB(0xbb, 0xde, 0xfb)},
				Begin:  "centerLeft",
				End:    "centerRight",
			},
			Shadows: []ui.Shadow{{
				Color:      ui.ARGB(0x33, 0, 0, 0),
				Offset:     ui.Offset{Dy: 2},
				BlurRadius: 4,
			}},
			Image: ui.P(ui.MemoryImage(pixelPNG)),
			Fit:   "cover",
		},
		Child: ui.Text{
			Data:  a.Greeting(),
			Style: &ui.TextStyle{Size: ui.P(22.0), Weight: "bold"},
		},
	}
}

func (a *App) counterRow() ui.Node {
	return ui.Row{
		MainAxisAlignment: ui.P("center"),
		Children: ui.Kids(
			ui.Text{
				Data:  fmt.Sprintf("count %d", a.count.Get()),
				Style: &ui.TextStyle{Size: ui.P(18.0)},
			},
			ui.SizedBox{Width: ui.P(16.0)},
			material.TextButton{Child: ui.Text{Data: "reset"}, OnPressed: a.Reset},
		),
	}
}

// noteSection holds a field whose FocusNode the host owns, and a button whose
// WidgetStatesController the host owns. The line under them holds the state
// that the host reported back, thus a change of the focus or of the press
// rebuilds this boundary alone.
func (a *App) noteSection() ui.Node {
	kids := ui.Kids(
		material.ElevatedButton{
			OnPressed:        a.ToggleNote,
			StatesController: &ui.StatesKey{Name: "toggle", OnStates: a.SetMarks},
			Child:            ui.Text{Data: a.noteLabel()},
		},
		ui.SizedBox{Width: ui.P(12.0)},
	)
	if a.note.Get() {
		kids = append(kids, ui.Expanded{Child: material.TextField{
			FocusNode: &ui.FocusKey{Name: "note", OnFocus: a.SetNoteFocus},
		}})
	} else {
		kids = append(kids, ui.Expanded{
			Child: ui.Text{Data: "the note field left the tree"},
		})
	}
	return ui.Column{
		CrossAxisAlignment: ui.P("start"),
		Children: ui.Kids(
			ui.Row{Children: kids},
			ui.Text{
				Data:  a.Report(),
				Style: &ui.TextStyle{Size: ui.P(12.0)},
			},
		),
	}
}

func (a *App) noteLabel() string {
	if a.note.Peek() {
		return "hide the note"
	}
	return "show the note"
}

// listPanel gives the list and the offset that the host reported. The offset
// stands in the boundary of the list, thus each scroll rebuilds the list from
// Go and the controller must keep its identity.
func (a *App) listPanel() ui.Node {
	body := a.listBody()
	list, ok := body.(ui.ListViewBuilder)
	if !ok {
		return body
	}
	return ui.Column{Children: ui.Kids(
		ui.Text{Data: fmt.Sprintf("offset %.0f", a.at.Get())},
		ui.Expanded{Child: list},
	)}
}

func (a *App) failSwitch() ui.Node {
	return material.Switch{Value: a.fail.Get(), OnChanged: a.SetFail}
}

func (a *App) listBody() ui.Node {
	if a.load.Busy() {
		return ui.Center{Child: material.CircularProgressIndicator{}}
	}
	if err := a.load.Err(); err != nil {
		return ui.Center{Child: ui.Text{Data: "error: " + err.Error()}}
	}
	rows := a.rows.Get()
	if len(rows) == 0 {
		return ui.Center{Child: ui.Text{Data: "press refresh to load 100,000 rows"}}
	}
	return ui.ListViewBuilder{
		Controller: &ui.ScrollKey{Name: "rows", OnScroll: a.SetOffset},
		ItemCount:  ui.P(len(rows)),
		ItemBuilder: func(i int) ui.Node {
			return ui.Keyed{Key: fmt.Sprint(i), Child: material.Card{
				Child: ui.Padding{
					Padding: ui.AllSides(12),
					Child:   ui.Text{Data: rows[i]},
				},
			}}
		},
	}
}

// root holds the 1 application of the process. The host reads it through the
// function below.
var root = New()

// Restart gives the application a fresh state. A widget test calls it between
// 2 cases.
func Restart() {
	ui.Reset()
	root = New()
}

// Tree gives the widget tree of the application. The Dart host calls it.
//
// The name differs from the method Build, because a top level name that a
// method of the package shadows takes a suffix in the Dart output.
func Tree() ui.Node { return root.Build() }

// shareRow gives the answer of the host, and an empty text before the first
// share. An error of the host reaches the same line.
func (a *App) shareRow() ui.Node {
	if err := a.share.Err(); err != nil {
		return ui.Text{Data: err.Error()}
	}
	return ui.Text{Data: a.shared.Get()}
}

func PrivateDefault(custom bool) ui.Node {
	button := material.FloatingActionButton{Child: ui.Text{Data: "brand"}}
	if custom {
		button.HeroTag = ui.P("brand")
	}
	return button
}
