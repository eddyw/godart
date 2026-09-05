package app

import (
	"errors"

	"example.dev/gocompass/domain"
	"example.dev/gocompass/model"
	"gart.dev/ui"
)

// errNoDestination reports that the itinerary holds no destination.
var errNoDestination = errors.New("the itinerary holds no destination")

// BookingModel holds the booking screen. It makes a booking from the choices,
// or it reads 1 booking of the server.
type BookingModel struct {
	app     *App
	booking ui.Signal[model.Booking]
	made    ui.Signal[bool]

	// Create makes the booking of the choices and it stores it.
	Create *ui.Command
	// Load reads 1 booking of the server.
	Load *ui.Command1[int]
}

func newBookingModel(a *App) *BookingModel {
	m := &BookingModel{app: a}
	m.Create = ui.NewCommand(m.create)
	m.Load = ui.NewCommand1(m.load)
	return m
}

func (m *BookingModel) create() error {
	u := domain.BookingCreate{
		Dest:     m.app.repos.Dest,
		Activity: m.app.repos.Activity,
		Booking:  m.app.repos.Booking,
	}
	b, err := u.CreateFrom(m.app.repos.Itinerary.Get())
	if err != nil {
		return err
	}
	ui.Post(func() {
		m.booking.Set(b)
		m.made.Set(true)
	})
	return nil
}

func (m *BookingModel) load(id int) error {
	b, err := m.app.repos.Booking.Get(id)
	if err != nil {
		return err
	}
	ui.Post(func() { m.booking.Set(b) })
	return nil
}

// Booking gives the trip of the screen. A test reads it.
func (m *BookingModel) Booking() model.Booking { return m.booking.Peek() }

// Build gives the tree of the booking.
func (m *BookingModel) Build() ui.Node {
	return ui.Scaffold{Body: ui.SafeArea{Child: ui.Watch{Build: m.body}}}
}

func (m *BookingModel) body() ui.Node {
	if m.Create.Busy() || m.Load.Busy() {
		return busy()
	}
	if m.Create.Err() != nil {
		return failed("Error while saving itinerary", m.Create.Run)
	}
	if m.Load.Err() != nil {
		return failed("Error while loading booking", func() {})
	}
	b := m.booking.Get()
	rows := []ui.Node{m.header(b), m.chosenHeader()}
	for _, a := range b.Activities {
		rows = append(rows, m.row(a))
	}
	return ui.Column{
		Children: ui.Kids(
			ui.Row{Children: ui.Kids(
				ui.IconButton{
					Icon:      ui.Icon{Icon: "close"},
					Tooltip:   ui.P(textClose),
					OnPressed: func() { m.app.popTo(screenHome) },
				},
				ui.Expanded{Child: ui.Text{Data: b.Destination.Name}},
				ui.IconButton{
					Icon:      ui.Icon{Icon: "share"},
					Tooltip:   ui.P(textShareTrip),
					OnPressed: nil,
				},
			)},
			ui.Expanded{Child: ui.ListView{Children: rows}},
		),
	}
}

func (m *BookingModel) header(b model.Booking) ui.Node {
	return ui.Stack{
		Children: ui.Kids(
			ui.ClipRRect{
				BorderRadius: ui.P(ui.AllCorners(12)),
				Child:        image(b.Destination.ImageURL, 200),
			},
			ui.Positioned{
				Left:   ui.P(16.0),
				Bottom: ui.P(16.0),
				Child: ui.Column{
					CrossAxisAlignment: ui.P("start"),
					Children: ui.Kids(
						ui.Text{
							Data: b.Destination.Name + ", " + b.Destination.Continent,
							Style: &ui.TextStyle{
								Color:  ui.P(ui.Color(0xFFFFFFFF)),
								Size:   ui.P(24.0),
								Weight: "w700",
							},
						},
						ui.Text{
							Data:  dateRange(b.StartDate, b.EndDate),
							Style: &ui.TextStyle{Color: ui.P(ui.Color(0xFFFFFFFF))},
						},
					),
				},
			},
		),
	}
}

func (m *BookingModel) chosenHeader() ui.Node {
	return ui.Padding{
		Padding: ui.Insets{Left: 20, Top: 16, Right: 20, Bottom: 4},
		Child: ui.Text{
			Data:  textChosenActivities,
			Style: &ui.TextStyle{Size: ui.P(18.0), Weight: "w700"},
		},
	}
}

func (m *BookingModel) row(a model.Activity) ui.Node {
	return ui.Keyed{Key: "chosen-" + a.Ref, Child: ui.ListTile{
		Leading:  ui.Icon{Icon: "star"},
		Title:    ui.Text{Data: a.Name},
		Subtitle: ui.Text{Data: a.LocationName},
	}}
}
