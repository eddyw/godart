package app

import (
	"fmt"
	"time"

	"example.dev/gocompass/model"
)

// The text of the screens. The sample holds the same strings in
// ui/core/localization/applocalization.dart, and it reads no other language.
const (
	textActivities        = "Activities"
	textAddDates          = "Add Dates"
	textBookNewTrip       = "Book New Trip"
	textClose             = "Close"
	textConfirm           = "Confirm"
	textDaytime           = "Daytime"
	textEvening           = "Evening"
	textLogin             = "Login"
	textSearch            = "Search"
	textSearchDestination = "Search destination"
	textShareTrip         = "Share Trip"
	textTryAgain          = "Try again"
	textChosenActivities  = "Your chosen activities"
	textWhen              = "When"
	textWho               = "Who"

	textBookingDeleted   = "Booking deleted"
	textErrorActivities  = "Error while loading activities"
	textErrorBooking     = "Error while loading booking"
	textErrorContinents  = "Error while loading continents"
	textErrorDelete      = "Error while deleting booking"
	textErrorDestination = "Error while loading destinations"
	textErrorHome        = "Error while loading home"
	textErrorLogin       = "Error while trying to login"
	textErrorLogout      = "Error while trying to logout"
	textErrorSaveActs    = "Error while saving activities"
	textErrorSaveTrip    = "Error while saving itinerary"
	textErrorShare       = "Error while sharing booking"
)

// trips gives the title of the home screen.
func trips(name string) string {
	if name == "" {
		return "Trips"
	}
	return name + "'s Trips"
}

// selected gives the count of the activities that the user chose.
func selected(n int) string { return fmt.Sprintf("%d selected", n) }

// dateRange gives the 2 dates of a trip, for example "20 - 25 Jul".
func dateRange(start, end time.Time) string { return model.DateRange(start, end) }

// guestText gives the line of the guest count.
func guestText(n int) string {
	if n == 1 {
		return "1 guest"
	}
	return fmt.Sprintf("%d guests", n)
}

// duration gives the length of an activity in hours.
func duration(h float64) string {
	if h == float64(int(h)) {
		return fmt.Sprintf("%d hours", int(h))
	}
	return fmt.Sprintf("%g hours", h)
}
