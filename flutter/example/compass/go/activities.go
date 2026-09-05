package app

import (
	"slices"

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
	return material.Scaffold{Body: ui.SafeArea{Child: ui.Watch{Build: m.body}}}
}

func (m *ActivitiesModel) body() ui.Node {
	if m.Load.Busy() {
		return busy()
	}
	if m.Load.Err() != nil {
		return failed("Error while loading activities", m.Load.Run)
	}
	rows := []ui.Node{m.header(textDaytime)}
	for _, a := range m.daytime.Get() {
		rows = append(rows, m.row(a))
	}
	rows = append(rows, m.header(textEvening))
	for _, a := range m.evening.Get() {
		rows = append(rows, m.row(a))
	}
	return ui.Column{
		Children: ui.Kids(
			ui.Row{Children: ui.Kids(
				backButton(m.app.pop),
				ui.Text{
					Data:  textActivities,
					Style: &ui.TextStyle{Size: ui.P(22.0), Weight: "w600"},
				},
			)},
			ui.Expanded{Child: ui.ListView{Children: rows}},
			m.confirmRow(),
		),
	}
}

func (m *ActivitiesModel) header(name string) ui.Node {
	return ui.Padding{
		Padding: ui.Insets{Left: 20, Top: 16, Right: 20, Bottom: 4},
		Child: ui.Text{
			Data:  name,
			Style: &ui.TextStyle{Size: ui.P(18.0), Weight: "w700"},
		},
	}
}

func (m *ActivitiesModel) row(a model.Activity) ui.Node {
	on := slices.Contains(m.chosen.Get(), a.Ref)
	return ui.Keyed{Key: "activity-" + a.Ref, Child: material.CheckboxListTile{
		Value:     on,
		Title:     ui.Text{Data: a.Name},
		Subtitle:  ui.Text{Data: a.LocationName + "  " + duration(a.Duration)},
		OnChanged: func(bool) { m.Toggle(a.Ref) },
	}}
}

func (m *ActivitiesModel) confirmRow() ui.Node {
	n := len(m.chosen.Get())
	if n == 0 {
		return ui.Padding{
			Padding: ui.AllSides(20),
			Child: material.FilledButton{
				Child:     ui.Text{Data: textConfirm},
				OnPressed: nil,
			},
		}
	}
	return ui.Padding{
		Padding: ui.AllSides(20),
		Child: ui.Row{
			Children: ui.Kids(
				ui.Expanded{Child: ui.Text{Data: selected(n)}},
				material.FilledButton{
					Child:     ui.Text{Data: textConfirm},
					OnPressed: m.Confirm,
				},
			),
		},
	}
}
