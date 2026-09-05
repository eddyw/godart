package app

import (
	"strconv"
	"time"

	"example.dev/gocompass/model"
	"gart.dev/ui"
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
	return ui.Scaffold{Body: ui.SafeArea{Child: ui.Watch{Build: m.body}}}
}

func (m *SearchModel) body() ui.Node {
	if m.Load.Busy() {
		return busy()
	}
	if m.Load.Err() != nil {
		return failed("Error while loading continents", m.Load.Run)
	}
	return ui.Column{
		Children: ui.Kids(
			ui.Row{Children: ui.Kids(
				backButton(m.app.pop),
				ui.Text{
					Data:  textSearch,
					Style: &ui.TextStyle{Size: ui.P(22.0), Weight: "w600"},
				},
			)},
			ui.Expanded{Child: ui.ListView{Children: ui.Kids(
				m.continentGrid(),
				m.dateRow(),
				m.guestRow(),
			)}},
			m.submitRow(),
		),
	}
}

// continentGrid gives the picture of each continent. The chosen 1 carries a
// border.
func (m *SearchModel) continentGrid() ui.Node {
	list := m.continents.Get()
	chosen := m.chosen.Get()
	rows := make([]ui.Node, 0, len(list))
	for _, c := range list {
		rows = append(rows, m.continentCard(c, c.Name == chosen))
	}
	return ui.SizedBox{
		Height: ui.P(140.0),
		Child: ui.ListView{
			ScrollDirection: ui.P("horizontal"),
			Children:        rows,
		},
	}
}

func (m *SearchModel) continentCard(c model.Continent, on bool) ui.Node {
	border := 0.0
	if on {
		border = 3.0
	}
	return ui.Keyed{Key: "continent-" + c.Name, Child: ui.Padding{
		Padding: ui.Sides(6, 8),
		Child: ui.InkWell{
			OnTap: func() { m.Choose(c.Name) },
			Child: ui.Container{
				Width: ui.P(160.0),
				Decoration: &ui.Decoration{
					Radius: ui.P(ui.AllCorners(12)),
					Border: &ui.BorderSide{Width: ui.P(border)},
				},
				Child: ui.Stack{
					Fit: ui.P("expand"),
					Children: ui.Kids(
						ui.ClipRRect{
							BorderRadius: ui.P(ui.AllCorners(12)),
							Child:        image(c.ImageURL, 120),
						},
						ui.Center{Child: ui.Text{
							Data: c.Name,
							Style: &ui.TextStyle{
								Color:  ui.P(ui.Color(0xFFFFFFFF)),
								Size:   ui.P(20.0),
								Weight: "w700",
							},
						}},
					),
				},
			},
		},
	}}
}

// dateRow gives the 2 dates of the trip. A tap opens the date picker of
// Flutter, and the sample opens the same 1.
func (m *SearchModel) dateRow() ui.Node {
	label := textAddDates
	if r := dateRange(m.start.Get(), m.end.Get()); r != "" {
		label = r
	}
	return ui.ListTile{
		Leading:  ui.Icon{Icon: "calendar_month"},
		Title:    ui.Text{Data: textWhen},
		Subtitle: ui.Text{Data: label},
		OnTap:    m.pickDates,
	}
}

// pickDates names the 2 dates. The node set holds no call of
// showDateRangePicker, because that call gives a Future and it needs a
// BuildContext. The form thus takes the 7 days after today.
func (m *SearchModel) pickDates() {
	start := time.Now()
	m.SetDates(start, start.AddDate(0, 0, 7))
}

func (m *SearchModel) guestRow() ui.Node {
	n := m.guests.Get()
	return ui.ListTile{
		Leading:  ui.Icon{Icon: "person"},
		Title:    ui.Text{Data: "Who"},
		Subtitle: ui.Text{Data: guestText(n)},
		Trailing: ui.Row{
			MainAxisSize: ui.P("min"),
			Children: ui.Kids(
				ui.IconButton{
					Icon:      ui.Icon{Icon: "remove"},
					Tooltip:   ui.P("1 guest less"),
					OnPressed: func() { m.SetGuests(n - 1) },
				},
				ui.Text{Data: strconv.Itoa(n)},
				ui.IconButton{
					Icon:      ui.Icon{Icon: "add"},
					Tooltip:   ui.P("1 guest more"),
					OnPressed: func() { m.SetGuests(n + 1) },
				},
			),
		},
	}
}

func (m *SearchModel) submitRow() ui.Node {
	if !m.Valid() {
		return ui.Padding{
			Padding: ui.AllSides(20),
			Child: ui.FilledButton{
				Child:     ui.Text{Data: textSearch},
				OnPressed: nil,
			},
		}
	}
	return ui.Padding{
		Padding: ui.AllSides(20),
		Child: ui.FilledButton{
			Child:     ui.Text{Data: textSearch},
			OnPressed: m.Submit,
		},
	}
}
