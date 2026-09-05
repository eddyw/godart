package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.dev/gocompass/data"
	"gart.dev/ui"
)

// The whole application runs under `go test`. No Flutter, no Dart, and no
// gart. The server of the test answers the 9 routes of the sample.

const token = " e1c37dfd973353b78bb71df050e2c6e72d53034e148920383968ae49b96f1fd2"

type fake struct {
	*httptest.Server
	posted []data.BookingAPI
	gone   []int
}

func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var in data.LoginRequest
		json.NewDecoder(r.Body).Decode(&in)
		if in.Email != "email@example.com" || in.Password != "password" {
			http.Error(w, "Invalid credentials", 401)
			return
		}
		write(w, data.LoginResponse{Token: token, UserID: "123"})
	})
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]string{"id": "123", "name": "Sofie",
			"email": "email@example.com", "picture": "assets/user.jpg"})
	})
	mux.HandleFunc("GET /continent", func(w http.ResponseWriter, r *http.Request) {
		write(w, []map[string]string{
			{"name": "Europe", "imageUrl": "http://x/europe.jpg"},
			{"name": "Asia", "imageUrl": "http://x/asia.jpg"},
		})
	})
	mux.HandleFunc("GET /destination", func(w http.ResponseWriter, r *http.Request) {
		write(w, []map[string]any{
			{"ref": "alps", "name": "Alps", "country": "Switzerland",
				"continent": "Europe", "knownFor": "snow",
				"tags": []string{"Mountain"}, "imageUrl": "http://x/alps.jpg"},
			{"ref": "rome", "name": "Rome", "country": "Italy",
				"continent": "Europe", "knownFor": "history",
				"tags": []string{"City"}, "imageUrl": "http://x/rome.jpg"},
			{"ref": "kyoto", "name": "Kyoto", "country": "Japan",
				"continent": "Asia", "knownFor": "temples",
				"tags": []string{"City"}, "imageUrl": "http://x/kyoto.jpg"},
		})
	})
	mux.HandleFunc("GET /destination/{ref}/activity", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("ref") != "alps" {
			write(w, []any{})
			return
		}
		write(w, []map[string]any{
			{"ref": "ski", "destinationRef": "alps", "name": "Ski",
				"locationName": "Zermatt", "duration": 6, "timeOfDay": "morning",
				"price": 3, "imageUrl": "http://x/ski.jpg"},
			{"ref": "fondue", "destinationRef": "alps", "name": "Fondue",
				"locationName": "Zermatt", "duration": 2.5, "timeOfDay": "evening",
				"price": 2, "imageUrl": "http://x/fondue.jpg"},
		})
	})
	mux.HandleFunc("GET /booking", func(w http.ResponseWriter, r *http.Request) {
		id := 0
		write(w, []data.BookingAPI{{
			ID:             &id,
			Name:           "Alps, Europe",
			StartDate:      time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC),
			DestinationRef: "alps",
			ActivitiesRef:  []string{"ski"},
		}})
	})
	mux.HandleFunc("GET /booking/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := 0
		write(w, data.BookingAPI{
			ID:             &id,
			Name:           "Alps, Europe",
			StartDate:      time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC),
			DestinationRef: "alps",
			ActivitiesRef:  []string{"ski"},
		})
	})
	mux.HandleFunc("POST /booking", func(w http.ResponseWriter, r *http.Request) {
		var in data.BookingAPI
		json.NewDecoder(r.Body).Decode(&in)
		f.posted = append(f.posted, in)
		id := 7
		in.ID = &id
		w.WriteHeader(201)
		write(w, in)
	})
	mux.HandleFunc("DELETE /booking/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.gone = append(f.gone, 1)
		w.WriteHeader(204)
	})
	f.Server = httptest.NewServer(auth(mux))
	old := baseURL
	SetBaseURL(f.URL)
	t.Cleanup(func() {
		SetBaseURL(old)
		f.Close()
	})
	return f
}

// auth rejects each request outside the login that carries no token, exactly
// as the server of the sample does.
func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" && r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "Unauthorized", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// login runs the login command and it applies the result.
func login(t *testing.T, a *App) {
	t.Helper()
	a.login.Login.Run()
	a.login.Login.Wait()
	ui.Pump()
	if err := a.login.Login.Err(); err != nil {
		t.Fatalf("the login gave the error %v", err)
	}
	a.home.Load.Wait()
	ui.Pump()
}

func TestTheFirstScreenIsTheLogin(t *testing.T) {
	ui.Reset()
	newFake(t)
	a := New()
	if got := a.Stack(); len(got) != 1 || got[0] != screenLogin {
		t.Fatalf("the stack holds %v", got)
	}
	if got := text(a.Build()); !strings.Contains(got, "[Email=email@example.com]") {
		t.Fatalf("the login screen gave %q", got)
	}
}

func TestTheLoginOpensTheHome(t *testing.T) {
	ui.Reset()
	newFake(t)
	a := New()
	login(t, a)
	if got := a.Stack(); len(got) != 1 || got[0] != screenHome {
		t.Fatalf("the stack holds %v", got)
	}
	got := text(a.Build())
	if !strings.Contains(got, "Sofie's Trips") {
		t.Fatalf("the home screen gave %q", got)
	}
	if !strings.Contains(got, "Alps, Europe") || !strings.Contains(got, "20 - 25 Jul") {
		t.Fatalf("the home screen holds no booking: %q", got)
	}
}

func TestAWrongPasswordReachesTheScreen(t *testing.T) {
	ui.Reset()
	newFake(t)
	a := New()
	a.login.password.Set("wrong")
	a.login.Login.Run()
	a.login.Login.Wait()
	ui.Pump()
	if a.login.Login.Err() == nil {
		t.Fatal("the login gave no error")
	}
	if got := text(a.Build()); !strings.Contains(got, "Error while trying to login") {
		t.Fatalf("the login screen gave %q", got)
	}
	if got := a.Stack(); got[0] != screenLogin {
		t.Fatalf("the stack holds %v", got)
	}
}

func TestTheWholeFlowMakesABooking(t *testing.T) {
	ui.Reset()
	f := newFake(t)
	a := New()
	login(t, a)

	a.push(route{name: screenSearch})
	a.search.Load.Wait()
	ui.Pump()
	if got := text(a.Build()); !strings.Contains(got, "Europe") {
		t.Fatalf("the search form gave %q", got)
	}
	a.search.Choose("Europe")
	a.search.SetGuests(2)
	a.search.SetDates(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC))
	if !a.search.Valid() {
		t.Fatal("the form reports that a choice is missing")
	}
	a.search.Submit()

	a.results.Search.Wait()
	ui.Pump()
	if got := a.results.Found(); len(got) != 2 {
		t.Fatalf("the search gave %d destinations, want 2", len(got))
	}
	if got := text(a.Build()); strings.Contains(got, "Kyoto") {
		t.Fatalf("the results hold a destination of another continent: %q", got)
	}
	a.results.Choose("alps")

	a.activities.Load.Wait()
	ui.Pump()
	got := text(a.Build())
	if !strings.Contains(got, "Ski") || !strings.Contains(got, "Fondue") {
		t.Fatalf("the activities screen gave %q", got)
	}
	if !strings.Contains(got, "Daytime") || !strings.Contains(got, "Evening") {
		t.Fatalf("the activities screen holds no header: %q", got)
	}
	a.activities.Toggle("ski")
	a.activities.Toggle("fondue")
	if n := len(a.activities.Chosen()); n != 2 {
		t.Fatalf("the model holds %d activities, want 2", n)
	}
	a.activities.Confirm()

	a.booking.Create.Wait()
	ui.Pump()
	if err := a.booking.Create.Err(); err != nil {
		t.Fatalf("the booking gave the error %v", err)
	}
	if len(f.posted) != 1 {
		t.Fatalf("the server holds %d bookings, want 1", len(f.posted))
	}
	p := f.posted[0]
	if p.DestinationRef != "alps" || p.Name != "Alps, Europe" {
		t.Fatalf("the server holds the booking %+v", p)
	}
	if len(p.ActivitiesRef) != 2 {
		t.Fatalf("the booking holds %d activities, want 2", len(p.ActivitiesRef))
	}
	if p.StartDate.Year() != 2026 || p.StartDate.Month() != time.September {
		t.Fatalf("the booking starts %v", p.StartDate)
	}
	if got := text(a.Build()); !strings.Contains(got, "Your chosen activities") {
		t.Fatalf("the booking screen gave %q", got)
	}
	if got := a.Stack(); len(got) != 5 {
		t.Fatalf("the stack holds %v", got)
	}
}

func TestTheCloseOfTheBookingReachesTheHome(t *testing.T) {
	ui.Reset()
	newFake(t)
	a := New()
	login(t, a)
	a.push(route{name: screenBooking, id: 1})
	a.booking.Load.Wait()
	ui.Pump()
	if got := text(a.Build()); !strings.Contains(got, "Ski") {
		t.Fatalf("the booking screen gave %q", got)
	}
	a.popTo(screenHome)
	if got := a.Stack(); len(got) != 1 || got[0] != screenHome {
		t.Fatalf("the stack holds %v", got)
	}
}

func TestTheBackGestureDropsTheTopScreen(t *testing.T) {
	ui.Reset()
	newFake(t)
	a := New()
	login(t, a)
	a.push(route{name: screenSearch})
	a.search.Load.Wait()
	ui.Pump()
	a.onPop(1)
	if got := a.Stack(); len(got) != 1 || got[0] != screenHome {
		t.Fatalf("the stack holds %v", got)
	}
}

func TestTheDeleteRemovesABooking(t *testing.T) {
	ui.Reset()
	f := newFake(t)
	a := New()
	login(t, a)
	a.home.Delete.Run(0)
	a.home.Delete.Wait()
	ui.Pump()
	if err := a.home.Delete.Err(); err != nil {
		t.Fatalf("the delete gave the error %v", err)
	}
	if len(f.gone) != 1 {
		t.Fatalf("the server saw %d deletes, want 1", len(f.gone))
	}
}

func TestAFailedLoadShowsTheRetry(t *testing.T) {
	ui.Reset()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data.LoginResponse{Token: token, UserID: "123"})
			return
		}
		http.Error(w, "no", 503)
	}))
	old := baseURL
	SetBaseURL(s.URL)
	t.Cleanup(func() { SetBaseURL(old); s.Close() })

	a := New()
	a.login.Login.Run()
	a.login.Login.Wait()
	ui.Pump()
	a.home.Load.Wait()
	ui.Pump()
	if a.home.Load.Err() == nil {
		t.Fatal("the load gave no error")
	}
	got := text(a.Build())
	if !strings.Contains(got, "Error while loading home") || !strings.Contains(got, "Try again") {
		t.Fatalf("the home screen gave %q", got)
	}
}
