// Package data reads the server of the application. The client holds the
// token, and each repository stands over it.
package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"example.dev/gocompass/model"
)

// BookingAPI holds the shape of a booking on the wire. The server names the
// destination and each activity with a reference, and the domain type holds
// the whole value.
type BookingAPI struct {
	ID             *int      `json:"id"`
	StartDate      time.Time `json:"startDate"`
	EndDate        time.Time `json:"endDate"`
	Name           string    `json:"name"`
	DestinationRef string    `json:"destinationRef"`
	ActivitiesRef  []string  `json:"activitiesRef"`
}

// Client reads the 9 routes of the server.
type Client struct {
	Base  string
	HTTP  *http.Client
	token string
}

// NewClient gives a client of the server at base, for example
// "http://localhost:8080".
func NewClient(base string) *Client {
	return &Client{Base: base, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// SetToken names the token that each request after it carries.
func (c *Client) SetToken(t string) { c.token = t }

// Token gives the token of the last login.
func (c *Client) Token() string { return c.token }

func (c *Client) do(method, path string, body any, out any, want int) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.Base+path, r)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		return fmt.Errorf("%s %s gave %s", method, path, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
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

// Login gives the token of the account, and it holds that token for each
// request after it.
func (c *Client) Login(email, password string) (LoginResponse, error) {
	var out LoginResponse
	err := c.do("POST", "/login", LoginRequest{Email: email, Password: password}, &out, 200)
	if err != nil {
		return LoginResponse{}, err
	}
	c.token = out.Token
	return out, nil
}

// Continents gives each continent of the search form.
func (c *Client) Continents() ([]model.Continent, error) {
	var out []model.Continent
	return out, c.do("GET", "/continent", nil, &out, 200)
}

// Destinations gives the whole catalog.
func (c *Client) Destinations() ([]model.Destination, error) {
	var out []model.Destination
	return out, c.do("GET", "/destination", nil, &out, 200)
}

// Activities gives each activity of 1 destination.
func (c *Client) Activities(ref string) ([]model.Activity, error) {
	var out []model.Activity
	return out, c.do("GET", "/destination/"+ref+"/activity", nil, &out, 200)
}

// Bookings gives each booking of the account.
func (c *Client) Bookings() ([]BookingAPI, error) {
	var out []BookingAPI
	return out, c.do("GET", "/booking", nil, &out, 200)
}

// Booking gives 1 booking by its number.
func (c *Client) Booking(id int) (BookingAPI, error) {
	var out BookingAPI
	return out, c.do("GET", fmt.Sprintf("/booking/%d", id), nil, &out, 200)
}

// CreateBooking stores a new booking. The server gives the number.
func (c *Client) CreateBooking(b BookingAPI) (BookingAPI, error) {
	b.ID = nil
	var out BookingAPI
	return out, c.do("POST", "/booking", b, &out, 201)
}

// DeleteBooking removes 1 booking.
func (c *Client) DeleteBooking(id int) error {
	return c.do("DELETE", fmt.Sprintf("/booking/%d", id), nil, nil, 204)
}

// User gives the account of the token.
func (c *Client) User() (model.User, error) {
	var out model.User
	return out, c.do("GET", "/user", nil, &out, 200)
}
