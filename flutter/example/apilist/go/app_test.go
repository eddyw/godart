package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gart.dev/ui"
	"gart.dev/ui/material"
)

// The whole screen runs under `go test`. No Flutter, no Dart, and no gart.

const twoIssues = `[
  {"number":71234,"title":"spec: clarify the range over a function",
   "state":"open","created_at":"2026-03-04T09:15:00Z",
   "user":{"login":"griesemer"}},
  {"number":71987,"title":"net/http: the client drops a trailer",
   "state":"closed","created_at":"2026-05-21T17:40:00Z",
   "user":{"login":"neild"}}
]`

// serve starts a server that gives body under the path that the app names, and
// it points the app at that server.
func serve(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	seen := make(chan string, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case seen <- r.URL.RequestURI():
		default:
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
		if got := <-seen; got != issuePath {
			t.Errorf("the client read %q, want %q", got, issuePath)
		}
	})
	return s
}

func TestTheFirstBodyAsksForARefresh(t *testing.T) {
	ui.Reset()
	a := New()
	if got := centerText(t, a); got != "press refresh to read the API" {
		t.Fatalf("the first body gave %q", got)
	}
}

func TestTheLoadDecodesEachField(t *testing.T) {
	ui.Reset()
	serve(t, http.StatusOK, twoIssues)
	a := New()

	a.Load()
	if !a.load.Busy() {
		t.Fatal("the command does not report busy after Load")
	}
	a.load.Wait()
	ui.Pump()
	if err := a.load.Err(); err != nil {
		t.Fatalf("the load gave the error %v", err)
	}

	list := a.Issues()
	if len(list) != 2 {
		t.Fatalf("the load gave %d issues, want 2", len(list))
	}
	first := list[0]
	if first.Number != 71234 {
		t.Errorf("the number is %d, want 71234", first.Number)
	}
	if first.State != "open" {
		t.Errorf("the state is %q, want %q", first.State, "open")
	}
	if first.User.Login != "griesemer" {
		t.Errorf("the author is %q, want %q", first.User.Login, "griesemer")
	}
	if got := first.Created.Year(); got != 2026 {
		t.Errorf("the year is %d, want 2026", got)
	}
	if got := Subtitle(first); got != "#71234  open  griesemer  2026-03-04" {
		t.Errorf("Subtitle gave %q", got)
	}
}

func TestTheBodyHoldsOneRowForEachIssue(t *testing.T) {
	ui.Reset()
	serve(t, http.StatusOK, twoIssues)
	a := New()
	a.Load()
	a.load.Wait()
	ui.Pump()

	list, ok := a.body().(ui.ListViewBuilder)
	if !ok {
		t.Fatalf("the body is %T, want a ListViewBuilder", a.body())
	}
	if list.ItemCount == nil || *list.ItemCount != 2 {
		t.Fatalf("the list holds %v rows, want 2", list.ItemCount)
	}
	keyed, ok := list.ItemBuilder(1).(ui.Keyed)
	if !ok {
		t.Fatalf("row 1 is %T, want a Keyed", list.ItemBuilder(1))
	}
	if keyed.Key != "71987" {
		t.Fatalf("row 1 holds the key %q, want %q", keyed.Key, "71987")
	}
	if got := rowText(t, keyed); !strings.Contains(got, "the client drops a trailer") {
		t.Fatalf("row 1 gave %q", got)
	}
}

func TestAStatusErrorReachesTheScreen(t *testing.T) {
	ui.Reset()
	serve(t, http.StatusServiceUnavailable, "")
	a := New()
	a.Load()
	a.load.Wait()

	if got := errorText(t, a); !strings.HasPrefix(got, "error: the server gave 503") {
		t.Fatalf("the body gave %q", got)
	}
}

func TestBrokenJSONReachesTheScreen(t *testing.T) {
	ui.Reset()
	serve(t, http.StatusOK, "{not json}")
	a := New()
	a.Load()
	a.load.Wait()

	if got := errorText(t, a); !strings.HasPrefix(got, "error: ") {
		t.Fatalf("the body gave %q", got)
	}
}

func TestTheLoadMarksTheBodyBoundary(t *testing.T) {
	ui.Reset()
	serve(t, http.StatusOK, twoIssues)
	a := New()
	body := ui.NewWatch()
	ui.RunWatch(body, a.body)
	ui.TakeDirty()

	a.Load()
	a.load.Wait()
	ui.Pump()

	got := ui.TakeDirty()
	if len(got) != 1 || got[0] != body {
		t.Fatalf("the load marked %v, want [%d]", got, body)
	}
}

// centerText gives the text of a body that holds 1 centered line.
func centerText(t *testing.T, a *App) string {
	t.Helper()
	center, ok := a.body().(ui.Center)
	if !ok {
		t.Fatalf("the body is %T, want a Center", a.body())
	}
	text, ok := center.Child.(ui.Text)
	if !ok {
		t.Fatalf("the child is %T, want a Text", center.Child)
	}
	return text.Data
}

// errorText gives the first line of a body that reports an error.
func errorText(t *testing.T, a *App) string {
	t.Helper()
	center, ok := a.body().(ui.Center)
	if !ok {
		t.Fatalf("the body is %T, want a Center", a.body())
	}
	col, ok := center.Child.(ui.Column)
	if !ok {
		t.Fatalf("the child is %T, want a Column", center.Child)
	}
	text, ok := col.Children[0].(ui.Text)
	if !ok {
		t.Fatalf("the first child is %T, want a Text", col.Children[0])
	}
	return text.Data
}

// rowText joins the 2 lines of 1 row.
func rowText(t *testing.T, keyed ui.Keyed) string {
	t.Helper()
	card, ok := keyed.Child.(material.Card)
	if !ok {
		t.Fatalf("the child is %T, want a Card", keyed.Child)
	}
	pad, ok := card.Child.(ui.Padding)
	if !ok {
		t.Fatalf("the child is %T, want a Padding", card.Child)
	}
	col, ok := pad.Child.(ui.Column)
	if !ok {
		t.Fatalf("the child is %T, want a Column", pad.Child)
	}
	var b strings.Builder
	for _, kid := range col.Children {
		if text, ok := kid.(ui.Text); ok {
			b.WriteString(text.Data)
			b.WriteString(" ")
		}
	}
	return b.String()
}
