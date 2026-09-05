// Package domain holds the work that stands over more than 1 repository.
package domain

import (
	"errors"
	"slices"

	"example.dev/gocompass/data"
	"example.dev/gocompass/model"
)

// BookingCreate builds 1 booking from the choices of the user, and it stores
// that booking on the server.
type BookingCreate struct {
	Dest     data.DestinationRepo
	Activity data.ActivityRepo
	Booking  data.BookingRepo
}

// CreateFrom gives the booking of the choices. It reports the first choice
// that the user did not make.
func (u BookingCreate) CreateFrom(c model.ItineraryConfig) (model.Booking, error) {
	if c.Destination == "" {
		return model.Booking{}, errors.New("the itinerary holds no destination")
	}
	list, err := u.Dest.Destinations()
	if err != nil {
		return model.Booking{}, err
	}
	i := slices.IndexFunc(list, func(d model.Destination) bool { return d.Ref == c.Destination })
	if i < 0 {
		return model.Booking{}, errors.New("the catalog holds no destination " + c.Destination)
	}
	if len(c.Activities) == 0 {
		return model.Booking{}, errors.New("the itinerary holds no activity")
	}
	all, err := u.Activity.ByDestination(c.Destination)
	if err != nil {
		return model.Booking{}, err
	}
	var chosen []model.Activity
	for _, a := range all {
		if slices.Contains(c.Activities, a.Ref) {
			chosen = append(chosen, a)
		}
	}
	if !c.Set() {
		return model.Booking{}, errors.New("the itinerary holds no date")
	}
	b := model.Booking{
		StartDate:   c.StartDate,
		EndDate:     c.EndDate,
		Destination: list[i],
		Activities:  chosen,
	}
	if err := u.Booking.Create(b); err != nil {
		return model.Booking{}, err
	}
	return b, nil
}
