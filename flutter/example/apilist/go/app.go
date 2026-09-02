// Package app holds the whole application. It reads an HTTP API, it decodes
// the answer into a Go struct, and it gives a list of widgets. It reads no
// Dart name, and a `go test` runs each part of it with no Flutter.
package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gart.dev/ui"
)

// Issue holds the part of a GitHub issue that the screen reads.
type Issue struct {
	Number  int       `json:"number"`
	Title   string    `json:"title"`
	State   string    `json:"state"`
	Created time.Time `json:"created_at"`
	User    User      `json:"user"`
}

// User holds the author of an issue.
type User struct {
	Login string `json:"login"`
}

// issuePath names the resource under the base URL.
const issuePath = "/repos/golang/go/issues?per_page=20"

// baseURL names the server. SetBaseURL points it at a local server.
var baseURL = "https://api.github.com"

// SetBaseURL names the server that the next load reads. A test calls it, from
// Go and from Dart.
func SetBaseURL(u string) { baseURL = u }

// App holds the state of the screen. Each field is a signal, thus a write of 1
// field rebuilds only the part of the screen that reads that field.
type App struct {
	issues ui.Signal[[]Issue]
	load   *ui.Command
}

// New gives an application with an empty list.
func New() *App {
	a := &App{}
	a.load = ui.NewCommand(a.fetch)
	return a
}

// fetch reads the API and decodes the answer. It runs on a goroutine of its
// own, thus it may block. A build never waits.
func (a *App) fetch() error {
	resp, err := http.Get(baseURL + issuePath)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the server gave %s", resp.Status)
	}
	var list []Issue
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}
	ui.Post(func() { a.issues.Set(list) })
	return nil
}

// Load starts a read of the API.
func (a *App) Load() { a.load.Run() }

// Issues gives the list that the last load wrote.
func (a *App) Issues() []Issue { return a.issues.Peek() }

// Subtitle gives the line under the title of an issue.
func Subtitle(it Issue) string {
	return fmt.Sprintf("#%d  %s  %s  %s",
		it.Number, it.State, it.User.Login, it.Created.Format(time.DateOnly))
}

// Build gives the tree of the screen.
func (a *App) Build() ui.Node {
	return ui.Scaffold{
		AppBar: ui.AppBar{
			Title: ui.Text{Data: "golang/go issues"},
			Actions: ui.Kids(
				ui.IconButton{
					Icon:        ui.Icon{Icon: "refresh"},
					Tooltip:     ui.P("read the API"),
					MouseCursor: ui.P("click"),
					OnPressed:   a.Load,
				},
			),
		},
		Body: ui.Watch{Build: a.body},
	}
}

func (a *App) body() ui.Node {
	if a.load.Busy() {
		return ui.Center{Child: ui.CircularProgressIndicator{}}
	}
	if err := a.load.Err(); err != nil {
		return ui.Center{Child: ui.Column{
			MainAxisAlignment: ui.P("center"),
			Children: ui.Kids(
				ui.Text{Data: "error: " + err.Error()},
				ui.TextButton{Child: ui.Text{Data: "try again"}, OnPressed: a.Load},
			),
		}}
	}
	list := a.issues.Get()
	if len(list) == 0 {
		return ui.Center{Child: ui.Text{Data: "press refresh to read the API"}}
	}
	return ui.ListViewBuilder{
		Physics:     ui.P("bouncing"),
		ItemCount:   ui.P(len(list)),
		ItemBuilder: func(i int) ui.Node { return row(list[i]) },
	}
}

func row(it Issue) ui.Node {
	return ui.Keyed{Key: fmt.Sprint(it.Number), Child: ui.Card{
		Margin: ui.P(ui.Sides(12, 6)),
		Shape: &ui.Shape{
			Kind:   "rounded",
			Radius: ui.AllCorners(12),
			Side:   &ui.BorderSide{Width: ui.P(1.0)},
		},
		Child: ui.Padding{
			Padding: ui.AllSides(12),
			Child: ui.Column{
				CrossAxisAlignment: ui.P("start"),
				Children: ui.Kids(
					ui.Text{
						Data:  it.Title,
						Style: &ui.TextStyle{Size: ui.P(16.0), Weight: "semibold"},
					},
					ui.SizedBox{Height: ui.P(4.0)},
					ui.Text{Data: Subtitle(it)},
				),
			},
		},
	}}
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

// Loaded reports whether the last load finished. A widget test polls it while
// the real socket work of the goroutine runs.
func Loaded() bool { return root.load.Done() }

// Tree gives the widget tree of the application. The Dart host calls it.
//
// The name differs from the method Build, because a top level name that a
// method of the package shadows takes a suffix in the Dart output.
func Tree() ui.Node { return root.Build() }
