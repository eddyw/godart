package app

import (
	"strconv"
	"time"

	"example.dev/gocompass/model"
	"gart.dev/ui"
	"gart.dev/ui/material"
)

// SearchModel holds the search form: the continent, the 2 dates, and the
// count of the guests.
type SearchModel struct {
	app        *App
	continents ui.Signal[[]model.Continent]
	chosen     ui.Signal[string]
	start      ui.Signal[time.Time]
	end        ui.Signal[time.Time]
	guests     ui.Signal[int]

	// Load reads the continents and the choices of the last visit.
	Load *ui.Command
}

func newSearchModel(a *App) *SearchModel {
	m := &SearchModel{app: a}
	m.Load = ui.NewCommand(m.load)
	return m
}

func (m *SearchModel) load() error {
	list, err := m.app.repos.Continent.Continents()
	if err != nil {
		return err
	}
	cfg := m.app.repos.Itinerary.Get()
	ui.Post(func() {
		m.continents.Set(list)
		m.chosen.Set(cfg.Continent)
		m.start.Set(cfg.StartDate)
		m.end.Set(cfg.EndDate)
		m.guests.Set(cfg.Guests)
	})
	return nil
}

// Valid reports whether the user made each choice of the form.
func (m *SearchModel) Valid() bool {
	return m.guests.Get() > 0 && m.chosen.Get() != "" &&
		!m.start.Get().IsZero() && !m.end.Get().IsZero()
}

// Choose names the continent of the search.
func (m *SearchModel) Choose(name string) { m.chosen.Set(name) }

// SetDates names the 2 dates of the trip.
func (m *SearchModel) SetDates(start, end time.Time) {
	m.start.Set(start)
	m.end.Set(end)
}

// SetGuests names the count of the guests. A count below 0 becomes 0.
func (m *SearchModel) SetGuests(n int) {
	if n < 0 {
		n = 0
	}
	m.guests.Set(n)
}

// Submit stores the choices and it opens the results.
func (m *SearchModel) Submit() {
	if !m.Valid() {
		return
	}
	cfg := m.app.repos.Itinerary.Get()
	cfg.Continent = m.chosen.Peek()
	cfg.StartDate = m.start.Peek()
	cfg.EndDate = m.end.Peek()
	cfg.Guests = m.guests.Peek()
	m.app.repos.Itinerary.Set(cfg)
	m.app.push(route{name: screenResults})
}

// Build gives the tree of the search form.
func (m *SearchModel) Build() ui.Node {
	return ui.PopScope{
		CanPop:       ui.P(false),
		OnPopInvoked: func(bool) { m.app.replace(route{name: screenHome}) },
		Child:        material.Scaffold{Body: ui.Watch{Build: m.body}},
	}
}

func (m *SearchModel) body() ui.Node {
	d := screen()
	return ui.Column{
		CrossAxisAlignment: ui.P("stretch"),
		Children: ui.Kids(
			ui.SafeArea{
				Top:    ui.P(true),
				Bottom: ui.P(false),
				Child: ui.Padding{
					Padding: ui.Insets{
						Left:   d.ScreenHorizontal,
						Top:    d.ScreenVertical,
						Right:  d.ScreenHorizontal,
						Bottom: paddingVertical,
					},
					Child: m.app.searchBar("", nil),
				},
			},
			m.continentRow(),
			m.dateRow(),
			m.guestRow(),
			ui.Spacer{},
			m.submitRow(),
		),
	}
}

// continentRow gives the carousel of the continents. The card of a continent
// that the user did not choose carries a cover of the color onPrimary.
func (m *SearchModel) continentRow() ui.Node {
	if m.Load.Busy() {
		return ui.SizedBox{Height: ui.P(140.0), Child: busy()}
	}
	if m.Load.Err() != nil {
		return ui.SizedBox{Height: ui.P(140.0), Child: ui.Center{
			Child: errorIndicator(textErrorContinents, textTryAgain, m.Load.Run),
		}}
	}
	list := m.continents.Get()
	chosen := m.chosen.Get()
	kids := make([]ui.Node, 0, len(list)*2)
	for i, c := range list {
		if i > 0 {
			kids = append(kids, ui.SizedBox{Width: ui.P(8.0)})
		}
		kids = append(kids, m.continentCard(c, chosen == "" || chosen == c.Name))
	}
	return ui.SizedBox{
		Height: ui.P(140.0),
		Child: ui.ListView{
			ScrollDirection: ui.P("horizontal"),
			Padding:         ui.P(screenX()),
			Children:        kids,
		},
	}
}

func (m *SearchModel) continentCard(c model.Continent, on bool) ui.Node {
	cover := 0.7
	if on {
		cover = 0
	}
	return ui.Keyed{Key: "continent-" + c.Name, Child: ui.SizedBox{
		Width:  ui.P(140.0),
		Height: ui.P(140.0),
		Child: ui.ClipRRect{
			BorderRadius: ui.P(ui.AllCorners(10)),
			Child: ui.Stack{Children: ui.Kids(
				picture(c.ImageURL, 140, 140, "cover"),
				ui.Align{
					Alignment: ui.P("center"),
					Child: ui.Padding{
						Padding: ui.AllSides(16),
						Child: ui.Text{
							Data:      c.Name,
							TextAlign: ui.P("center"),
							Style: &ui.TextStyle{
								Family:     fontOpenSans,
								Size:       ui.P(18.0),
								Weight:     "w500",
								Variations: weight(500),
								Color:      ui.P(colorWhite1),
							},
						},
					},
				},
				ui.PositionedFill{Child: ui.AnimatedOpacity{
					Opacity:  cover,
					Duration: 200,
					Child: ui.DecoratedBox{Decoration: ui.Decoration{
						Color: ui.P(ui.Theme().Colors.OnPrimary),
					}},
				}},
				ui.PositionedFill{Child: material.Material{
					Color: ui.P(ui.Color(0)),
					Child: material.InkWell{OnTap: func() { m.Toggle(c.Name) }},
				}},
			)},
		},
	}}
}

// Toggle names the continent, and a second tap of the same 1 drops it.
func (m *SearchModel) Toggle(name string) {
	if m.chosen.Peek() == name {
		m.chosen.Set("")
		return
	}
	m.chosen.Set(name)
}

// formBox gives the box of 64 pixels that the 2 rows of the form carry.
func formBox(child ui.Node) ui.Node {
	d := screen()
	return ui.Padding{
		Padding: ui.Insets{
			Left:  d.ScreenHorizontal,
			Top:   paddingVertical,
			Right: d.ScreenHorizontal,
		},
		Child: ui.Container{
			Height: ui.P(64.0),
			Decoration: &ui.Decoration{
				Border: &ui.BorderSide{Color: ui.P(colorGrey1)},
				Radius: ui.P(ui.AllCorners(16)),
			},
			Child: ui.Padding{
				Padding: ui.Sides(paddingHorizontal, 0),
				Child:   child,
			},
		},
	}
}

// dateRow opens the date range dialog of Material.
func (m *SearchModel) dateRow() ui.Node {
	var value ui.Node
	if r := dateRange(m.start.Get(), m.end.Get()); r != "" {
		value = ui.Text{Data: r, Style: ui.Theme().Text.BodyLarge}
	} else {
		value = ui.Text{Data: textAddDates, Style: hintStyle()}
	}
	return material.InkWell{
		BorderRadius: ui.P(ui.AllCorners(16)),
		OnTap:        m.pickDates,
		Child: formBox(ui.Row{
			MainAxisAlignment: ui.P("spaceBetween"),
			Children: ui.Kids(
				ui.Text{Data: textWhen, Style: ui.Theme().Text.TitleMedium},
				value,
			),
		}),
	}
}

// pickDates asks the host for the 2 days. The call runs in the goroutine of
// the tap, and it parks until the user closes the dialog.
func (m *SearchModel) pickDates() {
	from, to, err := ui.DateRange(0, 0, millis(m.start.Peek()), millis(m.end.Peek()))
	if err != nil || from == 0 || to == 0 {
		return
	}
	ui.Post(func() { m.SetDates(dayOf(from), dayOf(to)) })
}

// millis gives the count of milliseconds of the Unix epoch. A date of the host
// carries that form, and a date that holds no value gives 0.
func millis(t time.Time) int {
	if t.IsZero() {
		return 0
	}
	return int(t.UnixMilli())
}

// dayOf gives the day that the user saw. The dialog names a date of the zone
// of the machine, thus a read in UTC alone moves the day by 1 in a zone east
// of UTC. The read takes the calendar day of the local zone, and it writes
// that day at midnight UTC.
func dayOf(ms int) time.Time {
	d := time.UnixMilli(int64(ms))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

// guestRow gives the count of the guests and the 2 buttons of that count.
func (m *SearchModel) guestRow() ui.Node {
	n := m.guests.Get()
	style := ui.Theme().Text.BodyMedium
	if n == 0 {
		style = hintStyle()
	}
	return formBox(ui.Row{
		MainAxisAlignment: ui.P("spaceBetween"),
		Children: ui.Kids(
			ui.Text{Data: textWho, Style: ui.Theme().Text.TitleMedium},
			ui.SizedBox{Width: ui.P(90.0), Child: ui.Row{
				MainAxisAlignment: ui.P("spaceBetween"),
				Children: ui.Kids(
					ui.Keyed{Key: "remove-guests", Child: material.InkWell{
						OnTap: func() { m.SetGuests(n - 1) },
						Child: ui.Icon{
							Icon:  "remove_circle_outline",
							Color: ui.P(colorGrey3),
						},
					}},
					ui.Text{Data: strconv.Itoa(n), Style: style},
					ui.Keyed{Key: "add-guests", Child: material.InkWell{
						OnTap: func() { m.SetGuests(n + 1) },
						Child: ui.Icon{
							Icon:  "add_circle_outline",
							Color: ui.P(colorGrey3),
						},
					}},
				),
			}},
		),
	})
}

func (m *SearchModel) submitRow() ui.Node {
	d := screen()
	press := m.Submit
	if !m.Valid() {
		press = nil
	}
	return ui.Padding{
		Padding: ui.Insets{
			Left:   d.ScreenHorizontal,
			Top:    paddingVertical,
			Right:  d.ScreenHorizontal,
			Bottom: d.ScreenVertical,
		},
		Child: ui.Keyed{Key: "submit-button", Child: material.FilledButton{
			OnPressed: press,
			Child: ui.SizedBox{
				Height: ui.P(52.0),
				Child:  ui.Center{Child: ui.Text{Data: textSearch}},
			},
		}},
	}
}
