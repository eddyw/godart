package app

import (
	"strings"

	"example.dev/gocompass/model"
	"gart.dev/ui"
	"gart.dev/ui/material"
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
	return ui.PopScope{
		CanPop:       ui.P(false),
		OnPopInvoked: func(bool) { m.app.replace(route{name: screenSearch}) },
		Child:        material.Scaffold{Body: ui.Watch{Build: m.body}},
	}
}

func (m *ResultsModel) body() ui.Node {
	if m.Search.Busy() {
		return ui.Column{Children: ui.Kids(
			m.bar(),
			ui.Expanded{Child: busy()},
		)}
	}
	if m.Search.Err() != nil {
		return ui.Column{Children: ui.Kids(
			m.bar(),
			ui.Expanded{Child: ui.Center{Child: errorIndicator(
				textErrorDestination, textTryAgain, m.Search.Run,
			)}},
		)}
	}
	list := m.found.Get()
	cards := make([]ui.Node, len(list))
	for i, d := range list {
		cards[i] = m.card(d)
	}
	return ui.Padding{
		Padding: screenX(),
		Child: ui.CustomScrollView{Slivers: ui.Kids(
			ui.SliverToBoxAdapter{Child: m.bar()},
			ui.SliverGrid{
				GridDelegate: ui.GridSpec{
					Kind:         "count",
					Count:        2,
					MainSpacing:  8,
					CrossSpacing: 8,
					AspectRatio:  182.0 / 222.0,
				},
				Delegate: ui.SliverChildren{Children: cards},
			},
		)},
	}
}

// bar gives the search bar over the grid. A tap of it opens the form again.
func (m *ResultsModel) bar() ui.Node {
	d := screen()
	return ui.SafeArea{
		Top:    ui.P(true),
		Bottom: ui.P(false),
		Child: ui.Padding{
			Padding: ui.Insets{Top: d.ScreenVertical, Bottom: paddingVertical},
			Child:   m.app.searchBar(m.query.Get(), m.app.pop),
		},
	}
}

// card gives 1 destination of the grid: the picture, the name, and the tags.
func (m *ResultsModel) card(d model.Destination) ui.Node {
	chips := make([]ui.Node, len(d.Tags))
	for i, t := range d.Tags {
		chips[i] = tagChipSmall(t)
	}
	return ui.Keyed{Key: "result-" + d.Ref, Child: ui.ClipRRect{
		BorderRadius: ui.P(ui.AllCorners(10)),
		Child: ui.Stack{
			Fit: ui.P("expand"),
			Children: ui.Kids(
				picture(d.ImageURL, 0, 0, "fitHeight"),
				ui.Positioned{
					Bottom: ui.P(12.0),
					Left:   ui.P(12.0),
					Right:  ui.P(12.0),
					Child: ui.Column{
						MainAxisSize:       ui.P("min"),
						CrossAxisAlignment: ui.P("start"),
						Children: ui.Kids(
							ui.Text{Data: strings.ToUpper(d.Name), Style: cardTitle()},
							ui.SizedBox{Height: ui.P(6.0)},
							ui.Wrap{
								Spacing:    ui.P(4.0),
								RunSpacing: ui.P(4.0),
								Direction:  ui.P("horizontal"),
								Children:   chips,
							},
						),
					},
				},
				ui.PositionedFill{Child: material.Material{
					Color: ui.P(ui.Color(0)),
					Child: material.InkWell{OnTap: func() { m.Choose(d.Ref) }},
				}},
			),
		},
	}}
}

// cardTitle gives the name of a destination. The shadow of the sample makes
// the text stand out over a light picture.
func cardTitle() *ui.TextStyle {
	return &ui.TextStyle{
		Family:        fontRubik,
		Weight:        "w800",
		Variations:    weight(800),
		Size:          ui.P(15.0),
		Color:         ui.P(colorWhite),
		LetterSpacing: ui.P(1.0),
		Shadows:       []ui.Shadow{{Color: colorBlack, BlurRadius: 3}},
	}
}
