package data

import (
	"errors"
	"slices"

	"example.dev/gocompass/model"
)

// Repos holds each repository of the application. A view model reads 1 field
// of it, and a test writes a fake in that field.
type Repos struct {
	Auth      AuthRepo
	Continent ContinentRepo
	Dest      DestinationRepo
	Activity  ActivityRepo
	Booking   BookingRepo
	User      UserRepo
	Itinerary *ItineraryRepo
}

// AuthRepo holds the login of the account.
type AuthRepo interface {
	Authenticated() bool
	Login(email, password string) error
	Logout() error
}

// ContinentRepo reads the continents of the search form.
type ContinentRepo interface {
	Continents() ([]model.Continent, error)
}

// DestinationRepo reads the catalog.
type DestinationRepo interface {
	Destinations() ([]model.Destination, error)
}

// ActivityRepo reads the activities of 1 destination.
type ActivityRepo interface {
	ByDestination(ref string) ([]model.Activity, error)
}

// BookingRepo reads and writes the bookings of the account.
type BookingRepo interface {
	List() ([]model.BookingSummary, error)
	Get(id int) (model.Booking, error)
	Create(b model.Booking) error
	Delete(id int) error
}

// UserRepo reads the account.
type UserRepo interface {
	User() (model.User, error)
}

// NewRepos gives the repositories that read the server at base.
func NewRepos(base string) *Repos {
	c := NewClient(base)
	return &Repos{
		Auth:      &authRemote{c: c},
		Continent: &continentRemote{c: c},
		Dest:      &destRemote{c: c},
		Activity:  &activityRemote{c: c},
		Booking:   &bookingRemote{c: c},
		User:      &userRemote{c: c},
		Itinerary: &ItineraryRepo{},
	}
}

// ItineraryRepo holds the choices of the user in memory. The sample holds the
// same value in memory, thus a restart of the application drops it.
type ItineraryRepo struct{ cfg model.ItineraryConfig }

// Get gives the current choices.
func (r *ItineraryRepo) Get() model.ItineraryConfig { return r.cfg }

// Set writes the choices.
func (r *ItineraryRepo) Set(c model.ItineraryConfig) { r.cfg = c }

type authRemote struct {
	c  *Client
	in bool
}

func (a *authRemote) Authenticated() bool { return a.in }

func (a *authRemote) Login(email, password string) error {
	if _, err := a.c.Login(email, password); err != nil {
		return err
	}
	a.in = true
	return nil
}

func (a *authRemote) Logout() error {
	a.c.SetToken("")
	a.in = false
	return nil
}

type continentRemote struct{ c *Client }

func (r *continentRemote) Continents() ([]model.Continent, error) { return r.c.Continents() }

type destRemote struct {
	c      *Client
	cached []model.Destination
}

func (r *destRemote) Destinations() ([]model.Destination, error) {
	if r.cached != nil {
		return r.cached, nil
	}
	list, err := r.c.Destinations()
	if err != nil {
		return nil, err
	}
	r.cached = list
	return list, nil
}

type activityRemote struct{ c *Client }

func (r *activityRemote) ByDestination(ref string) ([]model.Activity, error) {
	return r.c.Activities(ref)
}

type bookingRemote struct {
	c      *Client
	cached []model.Destination
}

func (r *bookingRemote) List() ([]model.BookingSummary, error) {
	list, err := r.c.Bookings()
	if err != nil {
		return nil, err
	}
	out := make([]model.BookingSummary, 0, len(list))
	for _, b := range list {
		if b.ID == nil {
			continue
		}
		out = append(out, model.BookingSummary{
			ID: *b.ID, Name: b.Name, StartDate: b.StartDate, EndDate: b.EndDate,
		})
	}
	return out, nil
}

func (r *bookingRemote) Get(id int) (model.Booking, error) {
	api, err := r.c.Booking(id)
	if err != nil {
		return model.Booking{}, err
	}
	if r.cached == nil {
		list, err := r.c.Destinations()
		if err != nil {
			return model.Booking{}, err
		}
		r.cached = list
	}
	i := slices.IndexFunc(r.cached, func(d model.Destination) bool {
		return d.Ref == api.DestinationRef
	})
	if i < 0 {
		return model.Booking{}, errors.New("the catalog holds no destination " + api.DestinationRef)
	}
	dest := r.cached[i]
	all, err := r.c.Activities(dest.Ref)
	if err != nil {
		return model.Booking{}, err
	}
	var chosen []model.Activity
	for _, a := range all {
		if slices.Contains(api.ActivitiesRef, a.Ref) {
			chosen = append(chosen, a)
		}
	}
	out := model.Booking{
		StartDate: api.StartDate, EndDate: api.EndDate,
		Destination: dest, Activities: chosen,
	}
	if api.ID != nil {
		out.ID = *api.ID
	}
	return out, nil
}

func (r *bookingRemote) Create(b model.Booking) error {
	refs := make([]string, 0, len(b.Activities))
	for _, a := range b.Activities {
		refs = append(refs, a.Ref)
	}
	_, err := r.c.CreateBooking(BookingAPI{
		StartDate:      b.StartDate,
		EndDate:        b.EndDate,
		Name:           b.Destination.Name + ", " + b.Destination.Continent,
		DestinationRef: b.Destination.Ref,
		ActivitiesRef:  refs,
	})
	return err
}

func (r *bookingRemote) Delete(id int) error { return r.c.DeleteBooking(id) }

type userRemote struct{ c *Client }

func (r *userRemote) User() (model.User, error) { return r.c.User() }
