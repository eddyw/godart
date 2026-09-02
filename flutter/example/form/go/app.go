package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gart.dev/ui"
)

// Reply holds the answer of the server.
type Reply struct {
	ID       int       `json:"id"`
	Received time.Time `json:"received"`
}

// messagePath names the resource under the base URL.
const messagePath = "/messages"

// baseURL names the server. SetBaseURL points it at a local server.
var baseURL = "https://api.example.dev"

// SetBaseURL names the server that the next send reads. A test calls it, from
// Go and from Dart.
func SetBaseURL(u string) { baseURL = u }

// App holds the state of the screen. Each field is a signal, thus a write of 1
// field rebuilds only the part of the screen that reads that field.
type App struct {
	fields ui.Signal[Fields]
	tried  ui.Signal[bool]
	reply  ui.Signal[Reply]
	send   *ui.Command
}

// New gives an application with an empty form.
func New() *App {
	a := &App{}
	a.send = ui.NewCommand(a.post)
	return a
}

// Errors gives the report of each field of the current form.
func (a *App) Errors() Errors { return Validate(a.fields.Get()) }

// SetName writes the name field.
func (a *App) SetName(s string) { a.write(func(f *Fields) { f.Name = s }) }

// SetEmail writes the address field.
func (a *App) SetEmail(s string) { a.write(func(f *Fields) { f.Email = s }) }

// SetMessage writes the message field.
func (a *App) SetMessage(s string) { a.write(func(f *Fields) { f.Message = s }) }

func (a *App) write(set func(*Fields)) {
	a.fields.Update(func(f Fields) Fields {
		set(&f)
		return f
	})
}

// Submit validates the form and sends it. An invalid form reaches no server.
func (a *App) Submit() {
	a.tried.Set(true)
	if !Validate(a.fields.Peek()).Ok() {
		return
	}
	a.send.Run()
}

// post sends the fields and reads the answer. It runs on a goroutine of its
// own, thus it may block. A build never waits.
func (a *App) post() error {
	body, err := json.Marshal(a.fields.Peek())
	if err != nil {
		return err
	}
	resp, err := http.Post(baseURL+messagePath, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("the server gave %s", resp.Status)
	}
	var out Reply
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	ui.Post(func() { a.reply.Set(out) })
	return nil
}

// Build gives the tree of the screen.
func (a *App) Build() ui.Node {
	return ui.Scaffold{
		AppBar: ui.AppBar{Title: ui.Text{Data: "Go validates a form"}},
		Body: ui.Padding{
			Padding: ui.AllSides(16),
			Child:   ui.Watch{Build: a.body},
		},
	}
}

func (a *App) body() ui.Node {
	if a.send.Done() && a.send.Err() == nil {
		return a.sent()
	}
	f := a.fields.Get()
	e := Validate(f)
	return ui.Column{
		CrossAxisAlignment: ui.P("stretch"),
		Children: ui.Kids(
			a.field("name", "your name", f.Name, a.SetName, e.Name),
			a.field("email", "you@example.com", f.Email, a.SetEmail, e.Email),
			a.field("message", "your message", f.Message, a.SetMessage, e.Message),
			ui.SizedBox{Height: ui.P(8.0)},
			a.button(),
			ui.SizedBox{Height: ui.P(8.0)},
			a.report(e),
		),
	}
}

// field gives 1 labelled input and the report under it. The report shows after
// the first submit alone, thus an empty form holds no red text.
func (a *App) field(label, hint, value string, set func(string), report string) ui.Node {
	kids := ui.Kids(ui.TextInput{
		Label:     label,
		Hint:      hint,
		Value:     value,
		OnChanged: set,
	})
	if a.tried.Get() && report != "" {
		kids = append(kids, ui.Padding{
			Padding: ui.Sides(0, 4),
			Child: ui.Text{
				Data:  report,
				Style: &ui.TextStyle{Size: ui.P(12.0), Color: ui.P(ui.RGB(0xb3, 0x26, 0x1e))},
			},
		})
	}
	return ui.Padding{
		Padding: ui.Sides(0, 8),
		Child:   ui.Column{CrossAxisAlignment: ui.P("stretch"), Children: kids},
	}
}

func (a *App) button() ui.Node {
	if a.send.Busy() {
		return ui.Center{Child: ui.CircularProgressIndicator{}}
	}
	return ui.FilledButton{
		OnPressed: a.Submit,
		Child:     ui.Text{Data: "send"},
	}
}

func (a *App) report(e Errors) ui.Node {
	if err := a.send.Err(); err != nil {
		return ui.Text{Data: "error: " + err.Error()}
	}
	if a.tried.Get() && !e.Ok() {
		return ui.Text{Data: "the form holds an error"}
	}
	return ui.SizedBox{}
}

func (a *App) sent() ui.Node {
	r := a.reply.Get()
	return ui.Center{Child: ui.Column{
		MainAxisAlignment: ui.P("center"),
		Children: ui.Kids(
			ui.Text{
				Data:  "the server took the message",
				Style: &ui.TextStyle{Size: ui.P(18.0), Weight: "bold"},
			},
			ui.SizedBox{Height: ui.P(8.0)},
			ui.Text{Data: fmt.Sprintf("number %d, %s", r.ID, r.Received.Format(time.DateOnly))},
		),
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

// Sent reports whether the send finished. A widget test polls it while the
// real socket work of the goroutine runs.
func Sent() bool { return root.send.Done() || root.send.Err() != nil }

// Tree gives the widget tree of the application. The Dart host calls it.
//
// The name differs from the method Build, because a top level name that a
// method of the package shadows takes a suffix in the Dart output.
func Tree() ui.Node { return root.Build() }
