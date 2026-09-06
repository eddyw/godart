package app

import (
	"slices"
	"strings"

	"example.dev/gocompass/model"
	"gart.dev/ui"
	"gart.dev/ui/material"
)

// ActivitiesModel holds the 2 lists of the activities of the destination, and
// the set that the user chose.
type ActivitiesModel struct {
	app     *App
	daytime ui.Signal[[]model.Activity]
	evening ui.Signal[[]model.Activity]
	chosen  ui.Signal[[]string]

	// Load reads the activities of the destination of the itinerary.
	Load *ui.Command
}

func newActivitiesModel(a *App) *ActivitiesModel {
	m := &ActivitiesModel{app: a}
	m.Load = ui.NewCommand(m.load)
	return m
}

func (m *ActivitiesModel) load() error {
	cfg := m.app.repos.Itinerary.Get()
	if cfg.Destination == "" {
		return errNoDestination
	}
	list, err := m.app.repos.Activity.ByDestination(cfg.Destination)
	if err != nil {
		return err
	}
	var day, eve []model.Activity
	for _, a := range list {
		if a.Daytime() {
			day = append(day, a)
			continue
		}
		eve = append(eve, a)
	}
	ui.Post(func() {
		m.daytime.Set(day)
		m.evening.Set(eve)
		m.chosen.Set(slices.Clone(cfg.Activities))
	})
	return nil
}

// Chosen gives each activity that the user chose. A test reads it.
func (m *ActivitiesModel) Chosen() []string { return m.chosen.Peek() }

// Toggle adds or removes 1 activity.
func (m *ActivitiesModel) Toggle(ref string) {
	list := slices.Clone(m.chosen.Peek())
	if i := slices.Index(list, ref); i >= 0 {
		m.chosen.Set(slices.Delete(list, i, i+1))
		return
	}
	m.chosen.Set(append(list, ref))
}

// Confirm stores the choice and it opens the booking.
func (m *ActivitiesModel) Confirm() {
	cfg := m.app.repos.Itinerary.Get()
	cfg.Activities = slices.Clone(m.chosen.Peek())
	m.app.repos.Itinerary.Set(cfg)
	m.app.push(route{name: screenBooking})
}

// Build gives the tree of the activities.
func (m *ActivitiesModel) Build() ui.Node {
	return ui.PopScope{
		CanPop:       ui.P(false),
		OnPopInvoked: func(bool) { m.app.replace(route{name: screenResults}) },
		Child:        material.Scaffold{Body: ui.Watch{Build: m.body}},
	}
}

func (m *ActivitiesModel) body() ui.Node {
	if m.Load.Busy() {
		return ui.Column{Children: ui.Kids(
			m.header(),
			ui.Expanded{Child: busy()},
		)}
	}
	if m.Load.Err() != nil {
		return ui.Column{Children: ui.Kids(
			m.header(),
			ui.Expanded{Child: ui.Center{Child: errorIndicator(
				textErrorActivities, textTryAgain, m.Load.Run,
			)}},
		)}
	}
	day := m.daytime.Get()
	eve := m.evening.Get()
	return ui.Column{Children: ui.Kids(
		ui.Expanded{Child: ui.CustomScrollView{Slivers: ui.Kids(
			ui.SliverToBoxAdapter{Child: m.header()},
			listTitle(textDaytime, len(day)),
			m.list(day),
			listTitle(textEvening, len(eve)),
			m.list(eve),
		)}},
		m.bottom(),
	)}
}

// header gives the back button, the title, and the home button.
func (m *ActivitiesModel) header() ui.Node {
	d := screen()
	return ui.SafeArea{
		Top:    ui.P(true),
		Bottom: ui.P(false),
		Child: ui.Padding{
			Padding: ui.Insets{
				Left:   d.ScreenHorizontal,
				Top:    d.ScreenVertical,
				Right:  d.ScreenHorizontal,
				Bottom: paddingVertical,
			},
			Child: ui.Row{
				MainAxisAlignment: ui.P("spaceBetween"),
				Children: ui.Kids(
					backButton(false, func() {
						m.app.replace(route{name: screenResults})
					}),
					ui.Text{Data: textActivities, Style: ui.Theme().Text.TitleLarge},
					m.app.homeButton(false),
				),
			},
		},
	}
}

// listTitle names 1 of the 2 lists. An empty list carries no title.
func listTitle(label string, n int) ui.Node {
	if n == 0 {
		return ui.SliverToBoxAdapter{Child: ui.SizedBox{}}
	}
	return ui.SliverToBoxAdapter{Child: ui.Padding{
		Padding: screenX(),
		Child:   ui.Text{Data: label},
	}}
}

// list gives 1 of the 2 lists of the activities.
func (m *ActivitiesModel) list(items []model.Activity) ui.Node {
	d := screen()
	rows := make([]ui.Node, len(items))
	for i, a := range items {
		gap := 20.0
		if i == len(items)-1 {
			gap = 0
		}
		rows[i] = ui.Padding{
			Padding: ui.Insets{Bottom: gap},
			Child:   m.entry(a),
		}
	}
	return ui.SliverPadding{
		Padding: ui.Insets{
			Left:   d.ScreenHorizontal,
			Top:    paddingVertical,
			Right:  d.ScreenHorizontal,
			Bottom: paddingVertical,
		},
		Sliver: ui.SliverList{Delegate: ui.SliverChildren{Children: rows}},
	}
}

// entry gives 1 activity: the picture, the time of the day, the name, and the
// box of the choice.
func (m *ActivitiesModel) entry(a model.Activity) ui.Node {
	on := slices.Contains(m.chosen.Get(), a.Ref)
	return ui.Keyed{Key: "activity-" + a.Ref, Child: ui.SizedBox{
		Height: ui.P(80.0),
		Child: ui.Row{Children: ui.Kids(
			ui.ClipRRect{
				BorderRadius: ui.P(ui.AllCorners(8)),
				Child:        picture(a.ImageURL, 80, 80, "cover"),
			},
			ui.SizedBox{Width: ui.P(20.0)},
			ui.Expanded{Child: ui.Column{
				CrossAxisAlignment: ui.P("start"),
				MainAxisAlignment:  ui.P("start"),
				Children: ui.Kids(
					ui.Text{
						Data:  strings.ToUpper(a.TimeOfDay),
						Style: ui.Theme().Text.LabelSmall,
					},
					ui.Text{
						Data:     a.Name,
						MaxLines: ui.P(2),
						Overflow: ui.P("ellipsis"),
						Style:    ui.Theme().Text.BodyMedium,
					},
				),
			}},
			ui.SizedBox{Width: ui.P(20.0)},
			ui.Keyed{
				Key:   a.Ref + "-checkbox",
				Child: customCheckbox(on, func(bool) { m.Toggle(a.Ref) }),
			},
		)},
	}}
}

// bottom gives the count of the choices and the button of the booking.
func (m *ActivitiesModel) bottom() ui.Node {
	d := screen()
	n := len(m.chosen.Get())
	press := m.Confirm
	if n == 0 {
		press = nil
	}
	return ui.SafeArea{
		Bottom: ui.P(true),
		Child: material.Material{
			Elevation: ui.P(8.0),
			Child: ui.Padding{
				Padding: ui.Insets{
					Left:   d.ScreenHorizontal,
					Top:    paddingVertical,
					Right:  d.ScreenVertical,
					Bottom: d.ScreenVertical,
				},
				Child: ui.Row{
					MainAxisAlignment: ui.P("spaceBetween"),
					Children: ui.Kids(
						ui.Text{
							Data:  selected(n),
							Style: ui.Theme().Text.LabelLarge,
						},
						ui.Keyed{Key: "confirm-button", Child: material.FilledButton{
							OnPressed: press,
							Child:     ui.Text{Data: textConfirm},
						}},
					),
				},
			},
		},
	}
}
