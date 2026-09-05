package app

import (
	"strings"

	"example.dev/gocompass/model"
	"gart.dev/ui"
)

// ResultsModel holds the list of the destinations of the chosen continent.
type ResultsModel struct {
	app   *App
	found ui.Signal[[]model.Destination]
	query ui.Signal[string]

	// Search reads the catalog and it keeps the destinations of the continent.
	Search *ui.Command
}

func newResultsModel(a *App) *ResultsModel {
	m := &ResultsModel{app: a}
	m.Search = ui.NewCommand(m.search)
	return m
}

func (m *ResultsModel) search() error {
	cfg := m.app.repos.Itinerary.Get()
	list, err := m.app.repos.Dest.Destinations()
	if err != nil {
		return err
	}
	var out []model.Destination
	for _, d := range list {
		if d.Continent == cfg.Continent {
			out = append(out, d)
		}
	}
	ui.Post(func() { m.found.Set(out) })
	return nil
}

// Found gives the destinations of the last search. A test reads it.
func (m *ResultsModel) Found() []model.Destination { return m.found.Peek() }

// Choose names the destination of the trip and it opens the activities.
func (m *ResultsModel) Choose(ref string) {
	cfg := m.app.repos.Itinerary.Get()
	cfg.Destination = ref
	cfg.Activities = nil
	m.app.repos.Itinerary.Set(cfg)
	m.app.push(route{name: screenActivities})
}

// Build gives the tree of the results.
func (m *ResultsModel) Build() ui.Node {
	return ui.Scaffold{Body: ui.SafeArea{Child: ui.Watch{Build: m.body}}}
}

func (m *ResultsModel) body() ui.Node {
	if m.Search.Busy() {
		return busy()
	}
	if m.Search.Err() != nil {
		return failed("Error while loading destinations", m.Search.Run)
	}
	q := strings.ToLower(m.query.Get())
	var rows []ui.Node
	for _, d := range m.found.Get() {
		if q != "" && !strings.Contains(strings.ToLower(d.Name), q) &&
			!strings.Contains(strings.ToLower(d.Country), q) {
			continue
		}
		rows = append(rows, m.card(d))
	}
	if len(rows) == 0 {
		rows = append(rows, ui.Padding{
			Padding: ui.AllSides(40),
			Child:   ui.Center{Child: ui.Text{Data: "no destination"}},
		})
	}
	return ui.Column{
		Children: ui.Kids(
			ui.Row{Children: ui.Kids(
				backButton(m.app.pop),
				ui.Expanded{Child: ui.TextInput{
					Label:     textSearchDestination,
					Value:     m.query.Get(),
					OnChanged: func(s string) { m.query.Set(s) },
				}},
			)},
			ui.Expanded{Child: ui.ListView{Children: rows}},
		),
	}
}

func (m *ResultsModel) card(d model.Destination) ui.Node {
	tags := make([]ui.Node, 0, len(d.Tags))
	for _, t := range d.Tags {
		tags = append(tags, tag(t))
	}
	return ui.Keyed{Key: "dest-" + d.Ref, Child: ui.Padding{
		Padding: ui.Sides(20, 8),
		Child: ui.InkWell{
			OnTap: func() { m.Choose(d.Ref) },
			Child: ui.Column{
				CrossAxisAlignment: ui.P("start"),
				Children: ui.Kids(
					ui.ClipRRect{
						BorderRadius: ui.P(ui.AllCorners(12)),
						Child:        image(d.ImageURL, 160),
					},
					ui.Padding{
						Padding: ui.Insets{Left: 0, Top: 8, Right: 0, Bottom: 4},
						Child: ui.Text{
							Data:  d.Name + ", " + d.Continent,
							Style: &ui.TextStyle{Size: ui.P(18.0), Weight: "w600"},
						},
					},
					ui.Text{Data: d.Country},
					ui.Wrap{Children: tags},
				),
			},
		},
	}}
}
