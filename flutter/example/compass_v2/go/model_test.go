package app

import (
	"strings"
	"testing"
	"time"

	"gart.dev/ui"
)

func TestTripFlow(t *testing.T) {
	ui.Reset()
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Destinations) != 137 || len(a.Activities) != 2723 {
		t.Fatalf("catalog: %d destinations, %d activities", len(a.Destinations), len(a.Activities))
	}
	a.Login()
	if a.Route != "home" || len(a.Bookings) != 1 || a.Bookings[0].Destination.Name != "Alaska" {
		t.Fatal("default trip")
	}
	a.Navigate("search")
	a.SelectContinent("Europe")
	if a.Valid() {
		t.Fatal("incomplete form")
	}
	a.ChangeGuests(-1)
	if a.Guests != 0 {
		t.Fatal("negative guests")
	}
	a.ChangeGuests(2)
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 3)
	a.SetDates(int(start.UnixMilli()), int(end.UnixMilli()))
	if !a.Valid() {
		t.Fatal("complete form")
	}
	results := a.Results()
	if len(results) == 0 || results[0].Ref != "amalfi-coast" {
		t.Fatal("Europe results")
	}
	for _, d := range results {
		if d.Continent != "Europe" {
			t.Fatal("continent filter")
		}
	}
	a.ChooseDestination(results[0].Ref)
	day, night := a.ActivityList(false), a.ActivityList(true)
	if len(day) == 0 || len(night) == 0 {
		t.Fatal("activity groups")
	}
	a.ToggleActivity(day[0].Ref)
	a.ToggleActivity(night[0].Ref)
	a.ToggleActivity(day[0].Ref)
	if len(a.Selected) != 1 || a.Selected[0] != night[0].Ref {
		t.Fatal("activity selection")
	}
	a.Confirm()
	if a.Route != "booking" || len(a.Bookings) != 2 || len(a.Current.Activities) != 1 {
		t.Fatal("new trip")
	}
	if !strings.Contains(a.ShareText(), "Trip to Amalfi Coast\non 7 - 10 Sep\nActivities:\n - "+night[0].Name+".") {
		t.Fatal(a.ShareText())
	}
	id := a.Current.ID
	a.Navigate("home")
	a.OpenBooking(id)
	if a.Current.ID != id {
		t.Fatal("reopen trip")
	}
	a.DeleteBooking(id)
	if len(a.Bookings) != 1 || a.Bookings[0].ID == id {
		t.Fatal("delete trip")
	}
	a.Logout()
	if a.Route != "login" || a.Valid() || a.Guests != 0 || len(a.Selected) != 0 {
		t.Fatal("logout state")
	}
	a.Login()
	if len(a.Bookings) != 1 {
		t.Fatal("login duplicated the default trip")
	}
}

func TestSearchEdits(t *testing.T) {
	ui.Reset()
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a.SelectContinent("Europe")
	a.SelectContinent("Europe")
	if a.Continent != "" {
		t.Fatal("continent toggle")
	}
	a.SelectContinent("Asia")
	a.ChangeGuests(1)
	a.SetDates(1, 2)
	a.SetDates(0, 0)
	if a.Valid() || !a.Start.IsZero() {
		t.Fatal("cancel dates")
	}
	a.ChooseDestination("amalfi-coast")
	a.ToggleActivity(a.ActivityList(false)[0].Ref)
	a.ChooseDestination("alaska")
	if len(a.Selected) != 0 {
		t.Fatal("destination retained old activities")
	}
}

func TestDateText(t *testing.T) {
	for _, tc := range []struct{ start, end, want string }{
		{"2026-09-07", "2026-09-10", "7 - 10 Sep"},
		{"2026-09-30", "2026-10-02", "30 Sep - 2 Oct"},
		{"2026-12-31", "2027-01-01", "31 Dec - 1 Jan"},
	} {
		start, _ := time.Parse("2006-01-02", tc.start)
		end, _ := time.Parse("2006-01-02", tc.end)
		if got := dateText(start, end); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}
