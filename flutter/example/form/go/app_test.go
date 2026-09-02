package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gart.dev/ui"
)

// The whole screen runs under `go test`. No Flutter, no Dart, and no gart.

var good = Fields{
	Name:    "Ada",
	Email:   "ada@example.com",
	Message: "the machine weaves algebraic patterns",
}

func TestValidateReadsEachField(t *testing.T) {
	cases := []struct {
		name  string
		in    Fields
		wants Errors
	}{
		{"a valid form", good, Errors{}},
		{"an empty form", Fields{}, Errors{
			Name:    "the name is empty",
			Email:   "the address is empty",
			Message: "the message is empty",
		}},
		{"a name of 1 character", Fields{Name: "A", Email: good.Email, Message: good.Message},
			Errors{Name: "the name needs 2 characters or more"}},
		{"a name of only spaces", Fields{Name: "   ", Email: good.Email, Message: good.Message},
			Errors{Name: "the name is empty"}},
		{"a name of 41 characters", Fields{Name: strings.Repeat("a", 41), Email: good.Email, Message: good.Message},
			Errors{Name: "the name holds more than 40 characters"}},
		{"an address with no at sign", Fields{Name: good.Name, Email: "ada.example.com", Message: good.Message},
			Errors{Email: "the address has no valid form"}},
		{"an address with no domain dot", Fields{Name: good.Name, Email: "ada@example", Message: good.Message},
			Errors{Email: "the address has no valid form"}},
		{"an address with 2 at signs", Fields{Name: good.Name, Email: "a@b@example.com", Message: good.Message},
			Errors{Email: "the address has no valid form"}},
		{"an address with a space", Fields{Name: good.Name, Email: "a b@example.com", Message: good.Message},
			Errors{Email: "the address has no valid form"}},
		{"a message of 501 characters", Fields{Name: good.Name, Email: good.Email, Message: strings.Repeat("m", 501)},
			Errors{Message: "the message holds more than 500 characters"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Validate(c.in); got != c.wants {
				t.Fatalf("Validate gave %+v, want %+v", got, c.wants)
			}
			if got := Validate(c.in).Ok(); got != (c.wants == Errors{}) {
				t.Fatalf("Ok gave %v", got)
			}
		})
	}
}

// serve starts a server that gives body, and it points the app at that server.
// It keeps the request that the client sent.
func serve(t *testing.T, status int, body string) *[]byte {
	t.Helper()
	var sent []byte
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent, _ = io.ReadAll(r.Body)
		if r.URL.Path != messagePath {
			t.Errorf("the client read %q, want %q", r.URL.Path, messagePath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	old := baseURL
	SetBaseURL(s.URL)
	t.Cleanup(func() {
		SetBaseURL(old)
		s.Close()
	})
	return &sent
}

func TestAnInvalidFormReachesNoServer(t *testing.T) {
	ui.Reset()
	sent := serve(t, http.StatusCreated, `{"id":1}`)
	a := New()

	a.Submit()
	if a.send.Busy() {
		t.Fatal("the command runs, and the form holds 3 errors")
	}
	if len(*sent) != 0 {
		t.Fatalf("the client sent %q", *sent)
	}
	if !a.tried.Peek() {
		t.Fatal("the first submit does not mark the form")
	}
}

func TestAValidFormSendsTheJSON(t *testing.T) {
	ui.Reset()
	sent := serve(t, http.StatusCreated, `{"id":42,"received":"2026-09-02T10:00:00Z"}`)
	a := New()
	a.SetName(good.Name)
	a.SetEmail(good.Email)
	a.SetMessage(good.Message)

	a.Submit()
	a.send.Wait()
	if err := a.send.Err(); err != nil {
		t.Fatalf("the send gave the error %v", err)
	}

	var got Fields
	if err := json.Unmarshal(*sent, &got); err != nil {
		t.Fatalf("the body holds no JSON: %v", err)
	}
	if got != good {
		t.Fatalf("the client sent %+v, want %+v", got, good)
	}
	if id := a.reply.Peek().ID; id != 42 {
		t.Fatalf("the reply holds the number %d, want 42", id)
	}
	if year := a.reply.Peek().Received.Year(); year != 2026 {
		t.Fatalf("the reply holds the year %d, want 2026", year)
	}
}

func TestAStatusErrorReachesTheScreen(t *testing.T) {
	ui.Reset()
	serve(t, http.StatusInternalServerError, "")
	a := New()
	a.SetName(good.Name)
	a.SetEmail(good.Email)
	a.SetMessage(good.Message)

	a.Submit()
	a.send.Wait()

	text, ok := a.report(a.Errors()).(ui.Text)
	if !ok {
		t.Fatalf("the report is %T, want a Text", a.report(a.Errors()))
	}
	if !strings.HasPrefix(text.Data, "error: the server gave 500") {
		t.Fatalf("the report gave %q", text.Data)
	}
}

func TestTheReportShowsAfterTheFirstSubmit(t *testing.T) {
	ui.Reset()
	a := New()
	if got := fieldReports(a); len(got) != 0 {
		t.Fatalf("the first screen holds the reports %v", got)
	}

	a.Submit()
	if got := fieldReports(a); len(got) != 3 {
		t.Fatalf("the screen holds %d reports, want 3", len(got))
	}

	a.SetName(good.Name)
	got := fieldReports(a)
	if len(got) != 2 {
		t.Fatalf("the screen holds %d reports, want 2", len(got))
	}
	for _, r := range got {
		if strings.Contains(r, "name") {
			t.Fatalf("the name report stays: %q", r)
		}
	}
}

func TestTheKeystrokeMarksTheBodyBoundary(t *testing.T) {
	ui.Reset()
	a := New()
	body := ui.NewWatch()
	ui.RunWatch(body, a.body)
	ui.TakeDirty()

	a.SetName("A")
	got := ui.TakeDirty()
	if len(got) != 1 || got[0] != body {
		t.Fatalf("the keystroke marked %v, want [%d]", got, body)
	}
}

// fieldReports gives the text under each field that holds a report.
func fieldReports(a *App) []string {
	var out []string
	col, ok := a.body().(ui.Column)
	if !ok {
		return out
	}
	for _, kid := range col.Children {
		pad, ok := kid.(ui.Padding)
		if !ok {
			continue
		}
		inner, ok := pad.Child.(ui.Column)
		if !ok || len(inner.Children) < 2 {
			continue
		}
		report, ok := inner.Children[1].(ui.Padding)
		if !ok {
			continue
		}
		if text, ok := report.Child.(ui.Text); ok {
			out = append(out, text.Data)
		}
	}
	return out
}
