package app

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gart.dev/ui"
)

//go:embed data/destinations.json
var destinationJSON string

//go:embed data/activities.json
var activityJSON string

type Destination struct {
	Ref, Name, Country, Continent, KnownFor, ImageURL string
	Tags                                              []string
}

type Activity struct {
	Ref, Name, Description, DestinationRef, TimeOfDay, ImageURL string
}

type Booking struct {
	ID          int
	Destination Destination
	Activities  []Activity
	Start, End  time.Time
}

type Continent struct{ Name, ImageURL string }

var continents = []Continent{
	{"Europe", "https://rstr.in/google/tripedia/TmR12QdlVTT"},
	{"Asia", "https://rstr.in/google/tripedia/VJ8BXlQg8O1"},
	{"South America", "https://rstr.in/google/tripedia/flm_-o1aI8e"},
	{"Africa", "https://rstr.in/google/tripedia/-nzi8yFOBpF"},
	{"North America", "https://rstr.in/google/tripedia/jlbgFDrSUVE"},
	{"Oceania", "https://rstr.in/google/tripedia/vxyrDE-fZVL"},
	{"Australia", "https://rstr.in/google/tripedia/z6vy6HeRyvZ"},
}

type App struct {
	Destinations []Destination
	Activities   []Activity
	Bookings     []Booking
	Route        string
	Email        string
	Password     string
	Continent    string
	Start, End   time.Time
	Guests       int
	Destination  string
	Selected     []string
	Current      Booking
	nextID       int
}

var current *App

func New() (*App, error) {
	a := &App{Route: "login", Email: "email@example.com", Password: "password"}
	if err := json.Unmarshal([]byte(destinationJSON), &a.Destinations); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(activityJSON), &a.Activities); err != nil {
		return nil, err
	}
	refs := make(map[string]bool)
	for _, d := range a.Destinations {
		if d.Ref == "" || d.Name == "" || d.Continent == "" || !strings.HasPrefix(d.ImageURL, "https://") || refs[d.Ref] {
			return nil, fmt.Errorf("invalid destination: %s", d.Ref)
		}
		refs[d.Ref] = true
	}
	for _, v := range a.Activities {
		if !refs[v.DestinationRef] || v.Ref == "" || v.Name == "" || !strings.HasPrefix(v.ImageURL, "https://") {
			return nil, fmt.Errorf("invalid activity: %s", v.Ref)
		}
	}
	d := a.Destinations[0]
	var activities []Activity
	for _, v := range a.Activities {
		if v.DestinationRef == d.Ref && len(activities) < 4 {
			activities = append(activities, v)
		}
	}
	a.Bookings = []Booking{{ID: 0, Destination: d, Activities: activities,
		Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)}}
	a.nextID = 1
	return a, nil
}

func Bootstrap() {
	var err error
	current, err = New()
	if err != nil {
		panic(err)
	}
	ui.UseTheme(&lightTheme, &darkTheme)
}

func (a *App) Navigate(route string) { a.Route = route; ui.Dirty() }

func (a *App) Login() { a.Navigate("home") }

func (a *App) Logout() {
	a.Continent, a.Destination = "", ""
	a.Start, a.End = time.Time{}, time.Time{}
	a.Guests, a.Selected = 0, nil
	a.Navigate("login")
}

func (a *App) SelectContinent(name string) {
	if a.Continent == name {
		a.Continent = ""
	} else {
		a.Continent = name
	}
	ui.Dirty()
}

func (a *App) ChangeGuests(delta int) {
	a.Guests = max(0, a.Guests+delta)
	ui.Dirty()
}

func (a *App) SetDates(start, end int) {
	if start == 0 || end == 0 {
		a.Start, a.End = time.Time{}, time.Time{}
	} else {
		a.Start, a.End = time.UnixMilli(int64(start)).In(time.Local), time.UnixMilli(int64(end)).In(time.Local)
	}
	ui.Dirty()
}

func (a *App) ChooseDates() {
	start, end, err := ui.DateRange(0, 0, 0, 0)
	if err == nil {
		ui.Post(func() { a.SetDates(start, end) })
	}
}

func (a *App) Valid() bool {
	return a.Continent != "" && !a.Start.IsZero() && !a.End.IsZero() && a.Guests > 0
}

func (a *App) Results() []Destination {
	var out []Destination
	for _, d := range a.Destinations {
		if d.Continent == a.Continent {
			out = append(out, d)
		}
	}
	return out
}

func (a *App) ChooseDestination(ref string) {
	a.Destination, a.Selected = ref, nil
	a.Navigate("activities")
}

func (a *App) IsSelected(ref string) bool {
	for _, v := range a.Selected {
		if v == ref {
			return true
		}
	}
	return false
}

func (a *App) ToggleActivity(ref string) {
	for i, v := range a.Selected {
		if v == ref {
			a.Selected = append(a.Selected[:i], a.Selected[i+1:]...)
			ui.Dirty()
			return
		}
	}
	a.Selected = append(a.Selected, ref)
	ui.Dirty()
}

func (a *App) ActivityList(evening bool) []Activity {
	var out []Activity
	for _, v := range a.Activities {
		isEvening := v.TimeOfDay == "evening" || v.TimeOfDay == "night"
		if v.DestinationRef == a.Destination && isEvening == evening {
			out = append(out, v)
		}
	}
	return out
}

func (a *App) Confirm() {
	b := Booking{ID: a.nextID, Start: a.Start, End: a.End}
	for _, d := range a.Destinations {
		if d.Ref == a.Destination {
			b.Destination = d
			break
		}
	}
	for _, v := range a.Activities {
		if v.DestinationRef == a.Destination && a.IsSelected(v.Ref) {
			b.Activities = append(b.Activities, v)
		}
	}
	a.nextID++
	a.Bookings = append(a.Bookings, b)
	a.Current = b
	a.Navigate("booking")
}

func (a *App) OpenBooking(id int) {
	for _, b := range a.Bookings {
		if b.ID == id {
			a.Current = b
			a.Navigate("booking")
			return
		}
	}
}

func (a *App) DeleteBooking(id int) {
	for i, b := range a.Bookings {
		if b.ID == id {
			a.Bookings = append(a.Bookings[:i], a.Bookings[i+1:]...)
			ui.Dirty()
			return
		}
	}
}

func (a *App) ShareText() string {
	b := a.Current
	out := "Trip to " + b.Destination.Name + "\non " + dateText(b.Start, b.End) + "\nActivities:\n"
	for i, v := range b.Activities {
		if i > 0 {
			out += "\n"
		}
		out += " - " + v.Name
	}
	return out + "."
}

func (a *App) Share() {
	if err := ui.Share(a.ShareText()); err != nil {
		answer, _ := ui.SnackBar("Error while sharing booking", "Try again")
		if answer == "action" {
			a.Share()
		}
	}
}

func dateText(start, end time.Time) string {
	if start.Month() == end.Month() {
		return start.Format("2") + " - " + end.Format("2 Jan")
	}
	return start.Format("2 Jan") + " - " + end.Format("2 Jan")
}
