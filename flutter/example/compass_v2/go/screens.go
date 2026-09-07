// Copyright 2024 The Flutter team. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"gart.dev/ui"
	"gart.dev/ui/material"
	"gart.dev/ui/svg"
)

const (
	black = ui.Color(0xFF101010)
	white = ui.Color(0xFFFFF7FA)
	grey1 = ui.Color(0xFFF2F2F2)
	grey3 = ui.Color(0xFFA4A4A4)
)

var lightTheme = ui.ThemeStyle{ColorScheme: &ui.ColorScheme{
	Brightness: "light", Primary: black, OnPrimary: white, Secondary: black,
	OnSecondary: white, Surface: 0xFFFFFFFF, OnSurface: black, Error: 0xFFFFFFFF, OnError: 0xFFF44336,
}}

var darkTheme = ui.ThemeStyle{ColorScheme: &ui.ColorScheme{
	Brightness: "dark", Primary: white, OnPrimary: black, Secondary: white,
	OnSecondary: black, Surface: black, OnSurface: 0xFFFFFFFF, Error: 0xFF000000, OnError: 0xFFE74C3C,
}}

func dimensions() (horizontal, vertical, picture float64) {
	w := ui.Media().Width
	if w > 600 && w < 840 {
		return 100, 64, 128
	}
	return 20, 24, 64
}

func symmetric() ui.Insets                   { h, v, _ := dimensions(); return ui.Sides(h, v) }
func horizontal() ui.Insets                  { h, _, _ := dimensions(); return ui.Sides(h, 0) }
func pad(p ui.Insets, child ui.Node) ui.Node { return ui.Padding{Padding: p, Child: child} }
func column(children ...ui.Node) ui.Node {
	return ui.Column{CrossAxisAlignment: ui.P("start"), Children: children}
}
func gap(height float64) ui.Node              { return ui.SizedBox{Height: ui.P(height)} }
func space(width float64) ui.Node             { return ui.SizedBox{Width: ui.P(width)} }
func text(value, role string) ui.Node         { return Label{Text: value, Role: role} }
func keyed(key string, child ui.Node) ui.Node { return ui.Keyed{Key: key, Child: child} }
func round(radius float64, child ui.Node) ui.Node {
	return ui.ClipRRect{BorderRadius: ui.P(ui.AllCorners(radius)), Child: child}
}
func border(radius float64) *ui.Decoration {
	return &ui.Decoration{Border: &ui.BorderSide{Color: ui.P(grey1), Width: ui.P(1.0)}, Radius: ui.P(ui.AllCorners(radius))}
}
func tap(fn func(), child ui.Node) ui.Node { return material.InkWell{OnTap: fn, Child: child} }
func sliver(child ui.Node) ui.Node         { return ui.SliverToBoxAdapter{Child: child} }

func Tree() ui.Node {
	a := current
	var screen ui.Node
	switch a.Route {
	case "login":
		screen = a.loginScreen()
	case "home":
		screen = a.homeScreen()
	case "search":
		screen = a.searchScreen()
	case "results":
		screen = a.resultsScreen()
	case "activities":
		screen = a.activitiesScreen()
	case "booking":
		screen = a.bookingScreen()
	}
	return RouteView{Route: a.Route, Home: a.homeScreen(), Screen: screen, OnBack: func() {
		switch a.Route {
		case "results":
			a.Navigate("search")
		case "activities":
			a.Navigate("results")
		default:
			a.Navigate("home")
		}
	}}
}

func (a *App) loginScreen() ui.Node {
	return material.Scaffold{Body: ui.SingleChildScrollView{Child: ui.Column{
		CrossAxisAlignment: ui.P("center"), MainAxisAlignment: ui.P("spaceEvenly"), Children: ui.Kids(
			ui.ConstrainedBox{Constraints: ui.Constraints{MaxWidth: 300}, Child: ui.AspectRatio{AspectRatio: 1, Child: ui.Stack{Alignment: ui.P("center"), Children: ui.Kids(
				ui.Positioned{Left: ui.P(0.0), Child: loginCard("https://rstr.in/google/tripedia/g2i0BsYPKW-", 200, 273, -3.83, false)},
				ui.Positioned{Right: ui.P(0.0), Child: loginCard("https://rstr.in/google/tripedia/980sqNgaDRK", 180, 230, 3.46, false)},
				loginCard("https://rstr.in/google/tripedia/pHfPmf3o5NU", 225, 322, 0, true),
			)}}},
			pad(symmetric(), ui.Column{CrossAxisAlignment: ui.P("stretch"), Children: ui.Kids(
				keyed("email", Input{Value: a.Email, OnChanged: func(v string) { a.Email = v }}), gap(24),
				keyed("password", Input{Value: a.Password, Obscure: true, OnChanged: func(v string) { a.Password = v }}), gap(24),
				material.FilledButton{OnPressed: a.Login, Child: text("Login", "")},
			)}),
		),
	}}}
}

func loginCard(url string, width, height, degrees float64, title bool) ui.Node {
	kids := ui.Kids(Photo{URL: url, Fit: "cover", Darken: title})
	if title {
		kids = append(kids, ui.Center{Child: svg.SvgPictureAsset{AssetName: "assets/logo.svg"}})
	}
	return ui.TransformRotate{Angle: 2 * math.Pi * (degrees / 360), Alignment: ui.P("center"), Child: ui.SizedBox{Width: ui.P(width), Height: ui.P(height), Child: round(20,
		ui.Stack{Fit: ui.P("expand"), Alignment: ui.P("center"), Children: kids},
	)}}
}

func (a *App) iconButton(name, key string, onTap func(), blur bool) ui.Node {
	box := ui.DecoratedBox{Decoration: *border(8), Child: material.InkWell{BorderRadius: ui.P(ui.AllCorners(8)), OnTap: onTap,
		Child: ui.Center{Child: ui.Icon{Icon: name, Size: ui.P(24.0), Color: ui.P(ui.Theme().Colors.OnSurface)}}}}
	kids := []ui.Node{}
	if blur {
		kids = append(kids, ui.ClipRect{Child: ui.Blur{SigmaX: 2, SigmaY: 2, Child: ui.SizedBox{Width: ui.P(40.0), Height: ui.P(40.0)}}})
	}
	kids = append(kids, box)
	return keyed(key, ui.SizedBox{Width: ui.P(40.0), Height: ui.P(40.0), Child: ui.Stack{Fit: ui.P("expand"), Children: kids}})
}

func (a *App) homeButton(blur bool) ui.Node {
	return a.iconButton("home_outlined", "home-button", func() { a.Navigate("home") }, blur)
}

func (a *App) homeScreen() ui.Node {
	_, _, picture := dimensions()
	header := pad(symmetric(), ui.Column{CrossAxisAlignment: ui.P("stretch"), Children: ui.Kids(
		ui.Row{MainAxisAlignment: ui.P("spaceBetween"), CrossAxisAlignment: ui.P("start"), Children: ui.Kids(
			ui.ClipOval{Child: ui.ImageAsset{Name: "assets/user.jpg", Width: ui.P(picture), Height: ui.P(picture)}},
			a.iconButton("logout", "logout-button", a.Logout, false),
		)}, gap(24),
		ui.ShaderMask{BlendMode: "srcIn", Gradient: ui.Gradient{Kind: "radial", Center: "bottomLeft", Radius: 2, Colors: []ui.Color{0xFF7B1FA2, 0xFFAB47BC}},
			Child: Label{Text: "Sofie's Trips", Role: "headlineLarge", Font: "Rubik"}},
	)})
	rows := []ui.Node{}
	for _, b := range a.Bookings {
		id := b.ID
		rows = append(rows, ui.Dismissible{Key: "booking-" + strconv.Itoa(id), Direction: ui.P("endToStart"),
			OnDismissed: func(string) {
				a.DeleteBooking(id)
				go func() { _, _ = ui.SnackBar("Booking deleted", "") }()
			},
			Background: ui.Container{Color: ui.P(grey1), Child: ui.Row{MainAxisAlignment: ui.P("end"), Children: ui.Kids(pad(ui.Insets{Right: 20}, ui.Icon{Icon: "delete"}))}},
			Child: tap(func() { a.OpenBooking(id) }, pad(ui.Sides(horizontal().Left, 24), column(
				text(b.Destination.Name+", "+b.Destination.Continent, "titleLarge"), text(dateText(b.Start, b.End), "bodyLarge"),
			))),
		})
	}
	return material.Scaffold{FloatingActionButton: TripButton{OnPressed: func() { a.Navigate("search") }},
		Body: ui.SafeArea{Child: ui.CustomScrollView{Slivers: ui.Kids(sliver(header), ui.SliverListList{Children: rows})}}}
}

func (a *App) searchBar(filled bool) ui.Node {
	var query ui.Node = ui.Row{Children: ui.Kids(ui.Icon{Icon: "search"}, space(12), ui.Expanded{Child: text("Search destination", "hint")})}
	var action func()
	if filled {
		query = Label{Text: fmt.Sprintf("%s - %s - Guests: %d", a.Continent, dateText(a.Start, a.End), a.Guests), Role: "bodyLarge", Align: "center"}
		action = func() { a.Navigate("search") }
	}
	return ui.Row{Children: ui.Kids(ui.Expanded{Child: material.InkWell{OnTap: action, BorderRadius: ui.P(ui.AllCorners(16)),
		Child: ui.Container{Height: ui.P(64.0), Decoration: border(16), Child: pad(ui.Sides(20, 0), ui.Align{Alignment: ui.P("centerLeft"), Child: query})}}},
		space(10), a.homeButton(false))}
}

func (a *App) searchScreen() ui.Node {
	h, v, _ := dimensions()
	var submit func()
	if a.Valid() {
		submit = func() { a.Navigate("results") }
	}
	date := text("Add Dates", "hint")
	if !a.Start.IsZero() {
		date = text(dateText(a.Start, a.End), "bodyLarge")
	}
	guestRole := "bodyMedium"
	if a.Guests == 0 {
		guestRole = "hint"
	}
	return material.Scaffold{Body: ui.Column{CrossAxisAlignment: ui.P("stretch"), Children: ui.Kids(
		ui.SafeArea{Bottom: ui.P(false), Child: pad(ui.Insets{Left: h, Right: h, Top: v, Bottom: 24}, a.searchBar(false))},
		ui.SizedBox{Height: ui.P(140.0), Child: ui.ListViewSeparated{ScrollDirection: ui.P("horizontal"), Padding: ui.P(horizontal()), ItemCount: len(continents),
			ItemBuilder: func(i int) ui.Node { return a.continentCard(continents[i]) }, SeparatorBuilder: func(int) ui.Node { return space(8) }}},
		pad(ui.Insets{Left: h, Right: h, Top: 24}, material.InkWell{OnTap: a.ChooseDates, BorderRadius: ui.P(ui.AllCorners(16)), Child: ui.Container{Height: ui.P(64.0), Decoration: border(16), Child: pad(ui.Sides(20, 0), ui.Row{MainAxisAlignment: ui.P("spaceBetween"), Children: ui.Kids(text("When", "titleMedium"), date)})}}),
		pad(ui.Insets{Left: h, Right: h, Top: 24}, ui.Container{Height: ui.P(64.0), Decoration: border(16), Child: pad(ui.Sides(20, 0), ui.Row{MainAxisAlignment: ui.P("spaceBetween"), Children: ui.Kids(
			text("Who", "titleMedium"), ui.SizedBox{Width: ui.P(90.0), Child: ui.Row{MainAxisAlignment: ui.P("spaceBetween"), Children: ui.Kids(
				keyed("remove-guests", tap(func() { a.ChangeGuests(-1) }, ui.Icon{Icon: "remove_circle_outline", Color: ui.P(grey3)})),
				text(strconv.Itoa(a.Guests), guestRole),
				keyed("add-guests", tap(func() { a.ChangeGuests(1) }, ui.Icon{Icon: "add_circle_outline", Color: ui.P(grey3)})),
			)}},
		)})}),
		ui.Spacer{},
		pad(ui.Insets{Left: h, Right: h, Top: 24, Bottom: v}, keyed("submit-button", material.FilledButton{OnPressed: submit, Child: ui.SizedBox{Height: ui.P(52.0), Child: ui.Center{Child: text("Search", "")}}})),
	)}}
}

func (a *App) continentCard(c Continent) ui.Node {
	opacity := 0.0
	if a.Continent != "" && a.Continent != c.Name {
		opacity = 0.7
	}
	return keyed(c.Name, ui.SizedBox{Width: ui.P(140.0), Height: ui.P(140.0), Child: round(10, ui.Stack{Children: ui.Kids(
		Photo{URL: c.ImageURL, Fit: "cover", Placeholder: "continent"},
		ui.Align{Alignment: ui.P("center"), Child: pad(ui.AllSides(16), Label{Text: c.Name, Role: "continent", Align: "center"})},
		ui.PositionedFill{Child: ui.AnimatedOpacity{Opacity: opacity, Duration: 200, Child: ui.DecoratedBox{Decoration: ui.Decoration{Color: ui.P(ui.Theme().Colors.OnPrimary)}}}},
		ui.PositionedFill{Child: material.Material{Color: ui.P(ui.Color(0)), Child: tap(func() { a.SelectContinent(c.Name) }, nil)}},
	)})})
}

func (a *App) resultsScreen() ui.Node {
	_, v, _ := dimensions()
	cards := []ui.Node{}
	for _, d := range a.Results() {
		destination := d
		tags := []ui.Node{}
		for _, tag := range d.Tags {
			tags = append(tags, Tag{Text: tag})
		}
		cards = append(cards, keyed(d.Ref, round(10, ui.Stack{Fit: ui.P("expand"), Children: ui.Kids(
			Photo{URL: d.ImageURL, Fit: "fitHeight", Placeholder: "result"},
			ui.Positioned{Left: ui.P(12.0), Right: ui.P(12.0), Bottom: ui.P(12.0), Child: ui.Column{MainAxisSize: ui.P("min"), CrossAxisAlignment: ui.P("start"), Children: ui.Kids(
				Label{Text: strings.ToUpper(d.Name), Role: "cardTitle"}, gap(6), ui.Wrap{Spacing: ui.P(4.0), RunSpacing: ui.P(4.0), Children: tags},
			)}},
			ui.PositionedFill{Child: material.Material{Color: ui.P(ui.Color(0)), Child: tap(func() { a.ChooseDestination(destination.Ref) }, nil)}},
		)})))
	}
	return material.Scaffold{Body: pad(horizontal(), ui.CustomScrollView{Slivers: ui.Kids(
		sliver(ui.SafeArea{Bottom: ui.P(false), Child: pad(ui.Insets{Top: v, Bottom: 24}, a.searchBar(true))}),
		ui.SliverGridCount{CrossAxisCount: 2, CrossAxisSpacing: ui.P(8.0), MainAxisSpacing: ui.P(8.0), ChildAspectRatio: ui.P(182.0 / 222.0), Children: cards},
	)})}
}

func (a *App) activitiesScreen() ui.Node {
	h, v, _ := dimensions()
	header := ui.SafeArea{Bottom: ui.P(false), Child: pad(ui.Insets{Left: h, Right: h, Top: v, Bottom: 24}, ui.Row{MainAxisAlignment: ui.P("spaceBetween"), Children: ui.Kids(
		a.iconButton("arrow_back", "back-button", func() { a.Navigate("results") }, false), text("Activities", "titleLarge"), a.homeButton(false),
	)})}
	slivers := ui.Kids(sliver(header))
	for _, evening := range []bool{false, true} {
		list := a.ActivityList(evening)
		label := "Daytime"
		if evening {
			label = "Evening"
		}
		if len(list) > 0 {
			slivers = append(slivers, sliver(pad(horizontal(), text(label, ""))))
		}
		rows := []ui.Node{}
		for i, item := range list {
			bottom := 0.0
			if i < len(list)-1 {
				bottom = 20
			}
			rows = append(rows, pad(ui.Insets{Bottom: bottom}, a.activityEntry(item)))
		}
		slivers = append(slivers, ui.SliverPadding{Padding: ui.Insets{Left: h, Right: h, Top: 24, Bottom: 24}, Sliver: ui.SliverListList{Children: rows}})
	}
	var confirm func()
	if len(a.Selected) > 0 {
		confirm = a.Confirm
	}
	return material.Scaffold{Body: ui.Column{Children: ui.Kids(
		ui.Expanded{Child: ui.CustomScrollView{Slivers: slivers}},
		ui.SafeArea{Child: material.Material{Elevation: ui.P(8.0), Child: pad(ui.Insets{Left: h, Right: v, Top: 24, Bottom: v}, ui.Row{MainAxisAlignment: ui.P("spaceBetween"), Children: ui.Kids(
			text(fmt.Sprintf("%d selected", len(a.Selected)), "labelLarge"), keyed("confirm-button", material.FilledButton{OnPressed: confirm, Child: text("Confirm", "")}),
		)})}},
	)}}
}

func (a *App) activityEntry(item Activity) ui.Node {
	selected := a.IsSelected(item.Ref)
	var mark ui.Node = ui.SizedBox{Width: ui.P(14.0), Height: ui.P(14.0)}
	color := ui.Color(0)
	if selected {
		color = ui.Theme().Colors.Primary
		mark = ui.Icon{Icon: "check", Size: ui.P(14.0), Color: ui.P(ui.Theme().Colors.OnPrimary)}
	}
	checkbox := keyed(item.Ref+"-checkbox", material.InkResponse{Radius: ui.P(24.0), OnTap: func() { a.ToggleActivity(item.Ref) }, Child: ui.DecoratedBox{
		Decoration: ui.Decoration{Radius: ui.P(ui.AllCorners(24)), Border: &ui.BorderSide{Color: ui.P(grey3), Width: ui.P(1.0)}},
		Child:      material.Material{BorderRadius: ui.P(ui.AllCorners(24)), Color: ui.P(color), Child: ui.SizedBox{Width: ui.P(24.0), Height: ui.P(24.0), Child: mark}},
	}})
	return keyed(item.Ref, ui.SizedBox{Height: ui.P(80.0), Child: ui.Row{Children: ui.Kids(
		round(8, Photo{URL: item.ImageURL, Width: 80, Height: 80, Placeholder: "activity"}), space(20),
		ui.Expanded{Child: column(text(strings.ToUpper(item.TimeOfDay), "labelSmall"), Label{Text: item.Name, Role: "bodyMedium", MaxLines: 2})},
		space(20), checkbox,
	)}})
}

func (a *App) bookingScreen() ui.Node {
	b := a.Current
	h, v, _ := dimensions()
	tags := []ui.Node{}
	for _, tag := range b.Destination.Tags {
		tags = append(tags, Tag{Text: tag, Large: true})
	}
	header := column(
		ui.SizedBox{Height: ui.P(260.0), Child: ui.Stack{Fit: ui.P("expand"), Children: ui.Kids(
			Photo{URL: b.Destination.ImageURL, Fit: "fitWidth"},
			ui.DecoratedBox{Decoration: ui.Decoration{Gradient: &ui.Gradient{Begin: "topCenter", End: "bottomCenter", Colors: []ui.Color{0, ui.Theme().Colors.Surface}}}},
			ui.Align{Alignment: ui.P("bottomLeft"), Child: pad(symmetric(), ui.Column{MainAxisSize: ui.P("min"), CrossAxisAlignment: ui.P("start"), Children: ui.Kids(
				text(b.Destination.Name, "headlineLarge"), text(dateText(b.Start, b.End), "headlineSmall"),
			)})},
			ui.Positioned{Right: ui.P(h), Top: ui.P(v), Child: ui.SafeArea{Child: a.homeButton(true)}},
		)}},
		pad(horizontal(), text(b.Destination.KnownFor, "bodyLarge")), gap(24),
		pad(horizontal(), ui.Wrap{Spacing: ui.P(6.0), RunSpacing: ui.P(6.0), Children: tags}), gap(24),
		pad(horizontal(), text("Your chosen activities", "headlineSmall")),
	)
	rows := []ui.Node{}
	for _, item := range b.Activities {
		rows = append(rows, pad(ui.Insets{Top: 24, Left: h, Right: h}, ui.Row{Children: ui.Kids(
			round(8, Photo{URL: item.ImageURL, Width: 80, Height: 80}), space(20),
			ui.Expanded{Child: column(text(strings.ToUpper(item.TimeOfDay), "labelSmall"),
				Label{Text: item.Name, Role: "bodyMedium", MaxLines: 1}, Label{Text: item.Description, Role: "bodySmall", MaxLines: 2})},
		)}))
	}
	return material.Scaffold{FloatingActionButton: TripButton{Share: true, OnPressed: a.Share}, Body: ui.CustomScrollView{Slivers: ui.Kids(
		sliver(header), ui.SliverListList{Children: rows}, sliver(gap(200)),
	)}}
}
