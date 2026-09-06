// Package model holds the value types of the application. Each 1 holds no
// behavior, thus a test reads it as a table.
package model

import (
	"fmt"
	"time"
)

// Destination holds 1 place of the catalog.
type Destination struct {
	Ref       string   `json:"ref"`
	Name      string   `json:"name"`
	Country   string   `json:"country"`
	Continent string   `json:"continent"`
	KnownFor  string   `json:"knownFor"`
	Tags      []string `json:"tags"`
	ImageURL  string   `json:"imageUrl"`
}

// Activity holds 1 thing to do at a destination. TimeOfDay takes "any",
// "morning", "afternoon", "evening", or "night".
type Activity struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	LocationName   string  `json:"locationName"`
	Duration       float64 `json:"duration"`
	TimeOfDay      string  `json:"timeOfDay"`
	FamilyFriendly bool    `json:"familyFriendly"`
	Price          int     `json:"price"`
	DestinationRef string  `json:"destinationRef"`
	Ref            string  `json:"ref"`
	ImageURL       string  `json:"imageUrl"`
}

// Daytime reports whether the activity stands in the first list of the screen.
func (a Activity) Daytime() bool {
	return a.TimeOfDay == "any" || a.TimeOfDay == "morning" || a.TimeOfDay == "afternoon"
}

// Continent holds 1 row of the search form.
type Continent struct {
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// User holds the account of the login.
type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

// Booking holds 1 trip with the whole destination and the whole activity list.
type Booking struct {
	ID          int
	StartDate   time.Time
	EndDate     time.Time
	Destination Destination
	Activities  []Activity
}

// BookingSummary holds the row of the home screen. The server gives the name
// and the 2 dates, and the screen needs no more.
type BookingSummary struct {
	ID        int
	Name      string
	StartDate time.Time
	EndDate   time.Time
}

// ItineraryConfig holds the choices of the user between the search form and
// the booking. The application keeps 1 value of it.
type ItineraryConfig struct {
	Continent   string
	Destination string
	StartDate   time.Time
	EndDate     time.Time
	Guests      int
	Activities  []string
}

// Set reports whether the user chose the 2 dates.
func (c ItineraryConfig) Set() bool { return !c.StartDate.IsZero() && !c.EndDate.IsZero() }

// DateRange gives the 2 days of a trip, for example "20 - 25 Jul". The sample
// holds the same rule in ui/core/ui/date_format_start_end.dart.
func DateRange(start, end time.Time) string {
	if start.IsZero() || end.IsZero() {
		return ""
	}
	if start.Month() == end.Month() && start.Year() == end.Year() {
		return fmt.Sprintf("%d - %d %s", start.Day(), end.Day(), start.Format("Jan"))
	}
	return fmt.Sprintf("%s - %s", start.Format("2 Jan"), end.Format("2 Jan"))
}
