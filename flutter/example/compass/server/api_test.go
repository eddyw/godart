package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	s, err := newServer("../assets")
	if err != nil {
		t.Fatal(err)
	}
	return s.handler()
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if path != "/login" {
		r.Header.Set("Authorization", "Bearer "+loginToken)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestContinents(t *testing.T) {
	w := do(t, newTestServer(t), "GET", "/continent", "")
	if w.Code != 200 {
		t.Fatalf("the code is %d", w.Code)
	}
	var list []Continent
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 7 || list[0].Name != "Europe" {
		t.Fatalf("the list holds %d rows, and the first 1 is %q", len(list), list[0].Name)
	}
}

func TestDestinations(t *testing.T) {
	w := do(t, newTestServer(t), "GET", "/destination", "")
	var list []Destination
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 137 || list[0].Ref != "alaska" {
		t.Fatalf("the list holds %d rows, and the first ref is %q", len(list), list[0].Ref)
	}
}

func TestActivities(t *testing.T) {
	w := do(t, newTestServer(t), "GET", "/destination/alaska/activity", "")
	var list []Activity
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("the list holds no row")
	}
	for _, a := range list {
		if a.DestinationRef != "alaska" {
			t.Fatalf("the row %q holds the destination %q", a.Ref, a.DestinationRef)
		}
	}
}

func TestBookingRound(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "GET", "/booking", "")
	var list []Booking
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || *list[0].ID != 0 {
		t.Fatalf("the list holds %d rows", len(list))
	}

	body := `{"id":null,"startDate":"2024-09-01T00:00:00Z","endDate":"2024-09-08T00:00:00Z",` +
		`"name":"Kyoto, Asia","destinationRef":"kyoto","activitiesRef":["a1"]}`
	w = do(t, h, "POST", "/booking", body)
	if w.Code != 201 {
		t.Fatalf("the code is %d", w.Code)
	}
	var made Booking
	if err := json.Unmarshal(w.Body.Bytes(), &made); err != nil {
		t.Fatal(err)
	}
	if *made.ID != 1 || made.Name != "Kyoto, Asia" {
		t.Fatalf("the server gave the booking %d named %q", *made.ID, made.Name)
	}

	if w := do(t, h, "GET", "/booking/1", ""); w.Code != 200 {
		t.Fatalf("the read of the new booking gave %d", w.Code)
	}
	if w := do(t, h, "DELETE", "/booking/1", ""); w.Code != 204 {
		t.Fatalf("the delete gave %d", w.Code)
	}
	if w := do(t, h, "GET", "/booking/1", ""); w.Code != 404 {
		t.Fatalf("the read after the delete gave %d", w.Code)
	}
}

func TestBookingWithIDFails(t *testing.T) {
	body := `{"id":7,"startDate":"2024-09-01T00:00:00Z","endDate":"2024-09-08T00:00:00Z",` +
		`"name":"x","destinationRef":"kyoto","activitiesRef":[]}`
	if w := do(t, newTestServer(t), "POST", "/booking", body); w.Code != 400 {
		t.Fatalf("the code is %d", w.Code)
	}
}

func TestLogin(t *testing.T) {
	h := newTestServer(t)
	w := do(t, h, "POST", "/login", `{"email":"email@example.com","password":"password"}`)
	if w.Code != 200 {
		t.Fatalf("the code is %d", w.Code)
	}
	var res LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Token != loginToken || res.UserID != "123" {
		t.Fatalf("the answer holds the token %q and the user %q", res.Token, res.UserID)
	}
	if w := do(t, h, "POST", "/login", `{"email":"x","password":"y"}`); w.Code != 401 {
		t.Fatalf("the wrong password gave %d", w.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	h := newTestServer(t)
	r := httptest.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("the code is %d", w.Code)
	}
}

func TestUser(t *testing.T) {
	w := do(t, newTestServer(t), "GET", "/user", "")
	var u User
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
		t.Fatal(err)
	}
	if u.Name != "Sofie" || u.ID != "123" {
		t.Fatalf("the server gave the user %q", u.Name)
	}
}
