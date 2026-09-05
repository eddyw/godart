package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// The 4 constants of the login. The sample holds the same 4, and the token
// starts with 1 space.
const (
	loginEmail    = "email@example.com"
	loginPassword = "password"
	loginToken    = " e1c37dfd973353b78bb71df050e2c6e72d53034e148920383968ae49b96f1fd2"
	loginUserID   = "123"
)

// theUser is the account that each login gives.
var theUser = User{
	ID:      loginUserID,
	Name:    "Sofie",
	Email:   loginEmail,
	Picture: "assets/user.jpg",
}

// continents holds the first screen of the application.
var continents = []Continent{
	{Name: "Europe", ImageURL: "https://rstr.in/google/tripedia/TmR12QdlVTT"},
	{Name: "Asia", ImageURL: "https://rstr.in/google/tripedia/VJ8BXlQg8O1"},
	{Name: "South America", ImageURL: "https://rstr.in/google/tripedia/flm_-o1aI8e"},
	{Name: "Africa", ImageURL: "https://rstr.in/google/tripedia/-nzi8yFOBpF"},
	{Name: "North America", ImageURL: "https://rstr.in/google/tripedia/jlbgFDrSUVE"},
	{Name: "Oceania", ImageURL: "https://rstr.in/google/tripedia/vxyrDE-fZVL"},
	{Name: "Australia", ImageURL: "https://rstr.in/google/tripedia/z6vy6HeRyvZ"},
}

// server holds the catalog and the bookings. The bookings stand in memory, and
// a restart of the server drops each 1.
type server struct {
	destinations []Destination
	activities   []Activity
	bookings     []Booking
	nextID       int
}

// newServer reads the 2 JSON files of the catalog and it makes the first
// booking, exactly as the sample does.
func newServer(assets string) (*server, error) {
	s := &server{}
	if err := readJSON(filepath.Join(assets, "destinations.json"), &s.destinations); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(assets, "activities.json"), &s.activities); err != nil {
		return nil, err
	}
	d := s.destinations[0]
	var refs []string
	for _, a := range s.activities {
		if a.DestinationRef == d.Ref {
			refs = append(refs, a.Ref)
		}
	}
	id := s.nextID
	s.nextID++
	s.bookings = append(s.bookings, Booking{
		ID:             &id,
		Name:           d.Name + ", " + d.Continent,
		StartDate:      time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 8, 15, 0, 0, 0, 0, time.UTC),
		DestinationRef: d.Ref,
		ActivitiesRef:  refs,
	})
	return s, nil
}

func readJSON(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// handler gives the routes of the server under the authorization rule.
func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /continent", s.getContinents)
	mux.HandleFunc("GET /destination", s.getDestinations)
	mux.HandleFunc("GET /destination/{ref}/activity", s.getActivities)
	mux.HandleFunc("GET /booking", s.getBookings)
	mux.HandleFunc("GET /booking/{id}", s.getBooking)
	mux.HandleFunc("POST /booking", s.postBooking)
	mux.HandleFunc("DELETE /booking/{id}", s.deleteBooking)
	mux.HandleFunc("GET /user", s.getUser)
	mux.HandleFunc("POST /login", s.postLogin)
	return authorized(mux)
}

// authorized rejects each request outside /login that carries no token.
func authorized(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" && r.Header.Get("Authorization") != "Bearer "+loginToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *server) getContinents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, continents)
}

func (s *server) getDestinations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.destinations)
}

func (s *server) getActivities(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	list := []Activity{}
	for _, a := range s.activities {
		if a.DestinationRef == ref {
			list = append(list, a)
		}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *server) getBookings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.bookings)
}

func (s *server) getBooking(w http.ResponseWriter, r *http.Request) {
	b := s.booking(r.PathValue("id"))
	if b == nil {
		http.Error(w, "Invalid id", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, *b)
}

func (s *server) postBooking(w http.ResponseWriter, r *http.Request) {
	var b Booking
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if b.ID != nil {
		http.Error(w, "Booking already has id, use PUT instead.", http.StatusBadRequest)
		return
	}
	id := s.nextID
	s.nextID++
	b.ID = &id
	s.bookings = append(s.bookings, b)
	writeJSON(w, http.StatusCreated, b)
}

func (s *server) deleteBooking(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for i, b := range s.bookings {
		if b.ID != nil && strconv.Itoa(*b.ID) == id {
			s.bookings = append(s.bookings[:i], s.bookings[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	http.Error(w, "Invalid id", http.StatusNotFound)
}

func (s *server) getUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, theUser)
}

func (s *server) postLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if req.Email != loginEmail || req.Password != loginPassword {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{Token: loginToken, UserID: loginUserID})
}

func (s *server) booking(id string) *Booking {
	for i, b := range s.bookings {
		if b.ID != nil && strconv.Itoa(*b.ID) == id {
			return &s.bookings[i]
		}
	}
	return nil
}
