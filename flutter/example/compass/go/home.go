package app

import (
	"strconv"

	"example.dev/gocompass/model"
	"gart.dev/ui"
)

// HomeModel holds the home screen. It reads the bookings of the account and
// the account itself.
type HomeModel struct {
	app      *App
	bookings ui.Signal[[]model.BookingSummary]
	user     ui.Signal[model.User]

	// Load reads the bookings and the account.
	Load *ui.Command
	// Delete removes 1 booking, and it reads the list again.
	Delete *ui.Command1[int]
}

func newHomeModel(a *App) *HomeModel {
	m := &HomeModel{app: a}
	m.Load = ui.NewCommand(m.load)
	m.Delete = ui.NewCommand1(m.delete)
	return m
}

func (m *HomeModel) load() error {
	list, err := m.app.repos.Booking.List()
	if err != nil {
		return err
	}
	user, err := m.app.repos.User.User()
	if err != nil {
		return err
	}
	ui.Post(func() {
		m.bookings.Set(list)
		m.user.Set(user)
	})
	return nil
}

func (m *HomeModel) delete(id int) error {
	if err := m.app.repos.Booking.Delete(id); err != nil {
		return err
	}
	list, err := m.app.repos.Booking.List()
	if err != nil {
		return err
	}
	ui.Post(func() { m.bookings.Set(list) })
	return nil
}

// Bookings gives the list that the last read wrote. A test reads it.
func (m *HomeModel) Bookings() []model.BookingSummary { return m.bookings.Peek() }

// Build gives the tree of the home screen.
func (m *HomeModel) Build() ui.Node {
	return ui.Scaffold{
		FloatingActionButton: ui.FloatingActionButtonExtended{
			Label:     ui.Text{Data: textBookNewTrip},
			Icon:      ui.Icon{Icon: "add"},
			OnPressed: func() { m.app.push(route{name: screenSearch}) },
		},
		Body: ui.SafeArea{Child: ui.Watch{Build: m.body}},
	}
}

func (m *HomeModel) body() ui.Node {
	if m.Load.Busy() {
		return busy()
	}
	if m.Load.Err() != nil {
		return failed("Error while loading home", m.Load.Run)
	}
	list := m.bookings.Get()
	user := m.user.Get()
	rows := make([]ui.Node, 0, len(list)+2)
	rows = append(rows, ui.Padding{
		Padding: ui.Insets{Left: 20, Top: 24, Right: 20, Bottom: 8},
		Child: ui.Row{
			Children: ui.Kids(
				ui.Expanded{Child: ui.Text{
					Data:  trips(user.Name),
					Style: &ui.TextStyle{Size: ui.P(28.0), Weight: "w600"},
				}},
				ui.CircleAvatar{
					Radius: ui.P(24.0),
					Child:  ui.Text{Data: initial(user.Name)},
				},
			),
		},
	})
	for _, b := range list {
		rows = append(rows, m.card(b))
	}
	if len(list) == 0 {
		rows = append(rows, ui.Padding{
			Padding: ui.AllSides(40),
			Child:   ui.Center{Child: ui.Text{Data: "no trip yet"}},
		})
	}
	return ui.ListView{Children: rows}
}

// card gives 1 row of the list of the bookings. The delete button stands in
// the place of the swipe of the sample, because the node set holds no
// Dismissible. Read the section "What the port could not hold" of the README.
func (m *HomeModel) card(b model.BookingSummary) ui.Node {
	return ui.Keyed{Key: "booking-" + strconv.Itoa(b.ID), Child: ui.InkWell{
		OnTap: func() { m.app.push(route{name: screenBooking, id: b.ID}) },
		Child: ui.Padding{
			Padding: ui.Sides(20, 8),
			Child: ui.Row{
				Children: ui.Kids(
					ui.Expanded{Child: ui.Column{
						CrossAxisAlignment: ui.P("start"),
						Children: ui.Kids(
							ui.Text{
								Data:  b.Name,
								Style: &ui.TextStyle{Weight: "w600"},
							},
							ui.Text{Data: dateRange(b.StartDate, b.EndDate)},
						),
					}},
					ui.IconButton{
						Icon:      ui.Icon{Icon: "delete"},
						Tooltip:   ui.P("delete the trip"),
						OnPressed: func() { m.Delete.Run(b.ID) },
					},
				),
			},
		},
	}}
}

// initial gives the first letter of the name of the account.
func initial(name string) string {
	if name == "" {
		return "?"
	}
	return string([]rune(name)[:1])
}
