package main

import "time"

// Destination holds 1 place of the catalog. The field names follow the JSON
// of the sample, thus the Dart client of the sample reads this server.
type Destination struct {
	Ref       string   `json:"ref"`
	Name      string   `json:"name"`
	Country   string   `json:"country"`
	Continent string   `json:"continent"`
	KnownFor  string   `json:"knownFor"`
	Tags      []string `json:"tags"`
	ImageURL  string   `json:"imageUrl"`
}

// Activity holds 1 thing to do at a destination.
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

// Continent holds 1 row of the first screen.
type Continent struct {
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// Booking holds 1 trip. The id stands empty until the server stores it.
type Booking struct {
	ID             *int      `json:"id"`
	StartDate      time.Time `json:"startDate"`
	EndDate        time.Time `json:"endDate"`
	Name           string    `json:"name"`
	DestinationRef string    `json:"destinationRef"`
	ActivitiesRef  []string  `json:"activitiesRef"`
}

// User holds the account that a login gives.
type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

// LoginRequest holds the body of POST /login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse holds the answer of POST /login.
type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"userId"`
}
