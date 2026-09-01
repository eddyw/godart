// Package app holds the whole application. It reads no Dart name, and a
// `go test` runs each part of it with no Flutter.
package app

import (
	"errors"
	"fmt"
	"time"

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
}

// New gives an application with an empty state.
func New() *App {
	a := &App{}
	a.load = ui.NewCommand(a.fetch)
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

// Add counts 1 up.
func (a *App) Add() { a.count.Update(func(n int) int { return n + 1 }) }

// Reset counts back to 0.
func (a *App) Reset() { a.count.Set(0) }

// SetName writes the name that the greeting reads.
func (a *App) SetName(s string) { a.name.Set(s) }

// SetFail decides whether the next load fails. The switch of the screen writes
// it, and a test writes it too.
func (a *App) SetFail(v bool) { a.fail.Set(v) }

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
	return ui.Scaffold{
		AppBar: ui.AppBar{
			Title: ui.Text{Data: "Go drives Flutter"},
			Actions: ui.Kids(
				ui.IconButton{
					Icon:      ui.Icon{Icon: "refresh"},
					Tooltip:   ui.P("load the list"),
					OnPressed: a.load.Run,
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
			ui.Padding{
				Padding: ui.Sides(16, 0),
				Child: ui.Row{Children: ui.Kids(
					ui.Text{Data: "make the load fail"},
					ui.Spacer{},
					ui.Watch{Build: a.failSwitch},
				)},
			},
			ui.Divider{},
			ui.Expanded{Child: ui.Watch{Build: a.listBody}},
		)},
		FloatingActionButton: ui.IconButton{
			Icon:      ui.Icon{Icon: "add"},
			Tooltip:   ui.P("count 1 up"),
			OnPressed: a.Add,
		},
	}
}

func (a *App) greetingRow() ui.Node {
	return ui.Text{
		Data:  a.Greeting(),
		Style: &ui.TextStyle{Size: ui.P(22.0), Weight: "bold"},
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
			ui.TextButton{Child: ui.Text{Data: "reset"}, OnPressed: a.Reset},
		),
	}
}

func (a *App) failSwitch() ui.Node {
	return ui.Switch{Value: a.fail.Get(), OnChanged: a.SetFail}
}

func (a *App) listBody() ui.Node {
	if a.load.Busy() {
		return ui.Center{Child: ui.CircularProgressIndicator{}}
	}
	if err := a.load.Err(); err != nil {
		return ui.Center{Child: ui.Text{Data: "error: " + err.Error()}}
	}
	rows := a.rows.Get()
	if len(rows) == 0 {
		return ui.Center{Child: ui.Text{Data: "press refresh to load 100,000 rows"}}
	}
	return ui.ListViewBuilder{
		ItemCount: ui.P(len(rows)),
		ItemBuilder: func(i int) ui.Node {
			return ui.Keyed{Key: fmt.Sprint(i), Child: ui.Card{
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
