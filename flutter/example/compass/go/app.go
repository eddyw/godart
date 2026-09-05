// Package app holds the whole application: the router, the view models, and
// the 6 screens. It reads no Dart name, thus a `go test` runs each part of it
// with no Flutter.
//
// The application is a port of the Flutter sample compass_app. Read
// flutter/example/compass/README.md for the shape of the port and for each
// limit that the port found.
package app

import (
	"slices"
	"strconv"

	"example.dev/gocompass/data"
	"gart.dev/ui"
)

// baseURL names the server of the application.
var baseURL = "http://localhost:8080"

// SetBaseURL names the server that the next call of New reads. A test calls
// it, from Go and from Dart.
func SetBaseURL(u string) { baseURL = u }

// The 6 screens of the application.
const (
	screenLogin      = "login"
	screenHome       = "home"
	screenSearch     = "search"
	screenResults    = "results"
	screenActivities = "activities"
	screenBooking    = "booking"
)

// route names 1 screen of the stack. The number stands for the booking of the
// screen booking, and 0 means the booking that the user just made.
type route struct {
	name string
	id   int
}

// App holds the whole state of the application. 1 value of it stands behind
// the tree, and each screen reads the view model of its own name.
type App struct {
	repos *data.Repos
	stack ui.Signal[[]route]

	login      *LoginModel
	home       *HomeModel
	search     *SearchModel
	results    *ResultsModel
	activities *ActivitiesModel
	booking    *BookingModel
}

// New gives an application that reads the server of SetBaseURL. The first
// screen is the login.
func New() *App {
	a := &App{repos: data.NewRepos(baseURL)}
	a.login = newLoginModel(a)
	a.stack.Set([]route{{name: screenLogin}})
	return a
}

// Repos gives the repositories of the application. A test writes a fake in 1
// field of the result before the first build.
func (a *App) Repos() *data.Repos { return a.repos }

// Stack gives the name of each screen of the stack, from the bottom to the
// top. A test reads it.
func (a *App) Stack() []string {
	s := a.stack.Peek()
	out := make([]string, len(s))
	for i, r := range s {
		out[i] = r.name
	}
	return out
}

// push puts 1 screen over the stack.
func (a *App) push(r route) {
	a.open(r)
	a.stack.Set(append(slices.Clone(a.stack.Peek()), r))
}

// replace drops the whole stack and it puts 1 screen in the place of it.
func (a *App) replace(r route) {
	a.open(r)
	a.stack.Set([]route{r})
}

// pop removes the top screen. The first screen never leaves.
func (a *App) pop() {
	s := a.stack.Peek()
	if len(s) < 2 {
		return
	}
	a.stack.Set(slices.Clone(s[:len(s)-1]))
}

// popTo removes each screen over the named screen.
func (a *App) popTo(name string) {
	s := a.stack.Peek()
	for i, r := range s {
		if r.name == name {
			a.stack.Set(slices.Clone(s[:i+1]))
			return
		}
	}
}

// onPop answers the back gesture of the platform. Flutter removed the page of
// the number, thus the stack drops it.
func (a *App) onPop(i int) {
	s := a.stack.Peek()
	if i <= 0 || i >= len(s) {
		return
	}
	a.stack.Set(slices.Clone(s[:i]))
}

// open makes the view model of 1 screen and it starts the first read. The
// sample makes a fresh view model at each visit, and this call does the same.
func (a *App) open(r route) {
	switch r.name {
	case screenLogin:
		a.login = newLoginModel(a)
	case screenHome:
		a.home = newHomeModel(a)
		a.home.Load.Run()
	case screenSearch:
		a.search = newSearchModel(a)
		a.search.Load.Run()
	case screenResults:
		a.results = newResultsModel(a)
		a.results.Search.Run()
	case screenActivities:
		a.activities = newActivitiesModel(a)
		a.activities.Load.Run()
	case screenBooking:
		a.booking = newBookingModel(a)
		if r.id == 0 {
			a.booking.Create.Run()
		} else {
			a.booking.Load.Run(r.id)
		}
	}
}

// Build gives the tree of the whole application.
func (a *App) Build() ui.Node {
	return ui.Watch{Build: func() ui.Node {
		s := a.stack.Get()
		keys := make([]string, len(s))
		kids := make([]ui.Node, len(s))
		for i, r := range s {
			keys[i] = r.name + "/" + strconv.Itoa(r.id)
			kids[i] = a.screen(r)
		}
		return ui.Pages{Keys: keys, Children: kids, OnPop: a.onPop}
	}}
}

// screen gives the tree of 1 screen.
func (a *App) screen(r route) ui.Node {
	switch r.name {
	case screenLogin:
		return a.login.Build()
	case screenHome:
		return a.home.Build()
	case screenSearch:
		return a.search.Build()
	case screenResults:
		return a.results.Build()
	case screenActivities:
		return a.activities.Build()
	case screenBooking:
		return a.booking.Build()
	}
	return ui.Center{Child: ui.Text{Data: "no screen " + r.name}}
}

// root holds the application that the Dart host reads.
var root = New()

// Tree gives the widget tree of the application. The Dart host calls it.
//
// The name differs from the method Build, because a top level name that a
// method of the package shadows takes a suffix in the Dart output.
func Tree() ui.Node { return root.Build() }

// Root gives the application behind Tree. A widget test reads it.
func Root() *App { return root }

// Reset builds the application again. A widget test calls it after
// SetBaseURL, because the first read of baseURL stands in New.
func Reset() { root = New() }

// Settled reports whether the top screen finished its first read. A widget
// test polls it while the real socket work of the goroutine runs.
func Settled() bool { return root.settled() }

// Top names the screen on top of the stack. A widget test reads it.
func Top() string {
	s := root.stack.Peek()
	if len(s) == 0 {
		return ""
	}
	return s[len(s)-1].name
}

// settled reports whether each command of the top screen finished. The screen
// changes while a command runs, thus a test waits for the name of the screen
// too.
func (a *App) settled() bool {
	s := a.stack.Peek()
	if len(s) == 0 {
		return true
	}
	switch s[len(s)-1].name {
	case screenLogin:
		return !a.login.Login.Busy()
	case screenHome:
		return !a.home.Load.Busy() && !a.home.Delete.Busy()
	case screenSearch:
		return !a.search.Load.Busy()
	case screenResults:
		return !a.results.Search.Busy()
	case screenActivities:
		return !a.activities.Load.Busy()
	case screenBooking:
		return !a.booking.Create.Busy() && !a.booking.Load.Busy()
	}
	return true
}
