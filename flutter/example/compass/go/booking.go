package app

import (
	"errors"
	"strings"

	"example.dev/gocompass/domain"
	"example.dev/gocompass/model"
	"gart.dev/ui"
	"gart.dev/ui/material"
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

// Share gives the trip to the share sheet of the platform.
func (m *BookingModel) Share() {
	b := m.booking.Peek()
	if b.ID == 0 && !m.made.Peek() {
		return
	}
	u := domain.BookingShare{}
	if err := ui.Share(u.TextOf(b)); err != nil {
		toast(textErrorShare, textTryAgain, m.Share)
	}
}

// Build gives the tree of the booking.
func (m *BookingModel) Build() ui.Node {
	return ui.PopScope{
		CanPop:       ui.P(false),
		OnPopInvoked: func(bool) { m.app.replace(route{name: screenHome}) },
		Child: material.Scaffold{
			FloatingActionButton: ui.Watch{Build: m.shareButton},
			Body:                 ui.Watch{Build: m.body},
		},
	}
}

func (m *BookingModel) shareButton() ui.Node {
	press := m.Share
	if m.booking.Get().Destination.Ref == "" {
		press = nil
	}
	return ui.Keyed{Key: "share-button", Child: material.FloatingActionButtonExtended{
		Label:     ui.Text{Data: textShareTrip},
		Icon:      ui.Icon{Icon: "share_outlined"},
		OnPressed: press,
	}}
}

func (m *BookingModel) body() ui.Node {
	if m.Create.Busy() || m.Load.Busy() {
		return busy()
	}
	if m.Create.Err() != nil {
		return ui.Center{Child: errorIndicator(
			textErrorBooking, textTryAgain, m.Create.Run,
		)}
	}
	if m.Load.Err() != nil {
		return ui.Center{Child: errorIndicator(textErrorBooking, textClose, func() {
			m.app.replace(route{name: screenHome})
		})}
	}
	b := m.booking.Get()
	if b.Destination.Ref == "" {
		return ui.SizedBox{}
	}
	rows := make([]ui.Node, 0, len(b.Activities)+2)
	rows = append(rows, ui.SliverToBoxAdapter{Child: m.header(b)})
	for _, a := range b.Activities {
		rows = append(rows, ui.SliverToBoxAdapter{Child: activityRow(a)})
	}
	rows = append(rows, ui.SliverToBoxAdapter{
		Child: ui.SizedBox{Height: ui.P(200.0)},
	})
	return ui.CustomScrollView{Slivers: rows}
}

// header gives the picture, the name, the 2 dates, the text of the place, and
// the tags.
func (m *BookingModel) header(b model.Booking) ui.Node {
	return ui.Column{
		CrossAxisAlignment: ui.P("start"),
		Children: ui.Kids(
			bookingTop(b),
			ui.Padding{
				Padding: screenX(),
				Child: ui.Text{
					Data:  b.Destination.KnownFor,
					Style: ui.Theme().Text.BodyLarge,
				},
			},
			ui.SizedBox{Height: ui.P(paddingVertical)},
			bookingTags(b),
			ui.SizedBox{Height: ui.P(paddingVertical)},
			ui.Padding{
				Padding: screenX(),
				Child: ui.Text{
					Data:  textChosenActivities,
					Style: ui.Theme().Text.HeadlineSmall,
				},
			},
		),
	}
}

// bookingTop gives the picture of the destination, the gradient over it, and
// the 2 lines of the head.
func bookingTop(b model.Booking) ui.Node {
	return ui.SizedBox{
		Height: ui.P(260.0),
		Child: ui.Stack{
			Fit: ui.P("expand"),
			Children: ui.Kids(
				picture(b.Destination.ImageURL, 0, 0, "fitWidth"),
				ui.DecoratedBox{Decoration: ui.Decoration{Gradient: &ui.Gradient{
					Kind:   "linear",
					Begin:  "topCenter",
					End:    "bottomCenter",
					Colors: []ui.Color{0, ui.Theme().Colors.Surface},
				}}},
				ui.Align{
					Alignment: ui.P("bottomLeft"),
					Child: ui.Padding{
						Padding: screenSides(),
						Child: ui.Column{
							MainAxisSize:       ui.P("min"),
							CrossAxisAlignment: ui.P("start"),
							Children: ui.Kids(
								ui.Text{
									Data:  b.Destination.Name,
									Style: ui.Theme().Text.HeadlineLarge,
								},
								ui.Text{
									Data:  dateRange(b.StartDate, b.EndDate),
									Style: ui.Theme().Text.HeadlineSmall,
								},
							),
						},
					},
				},
			),
		},
	}
}

// bookingTags gives the tags of the destination. The chip of this screen
// stands over the color of the surface, thus it reads the other 2 colors.
func bookingTags(b model.Booking) ui.Node {
	chip := colorBlackTransparent
	if ui.Theme().Dark {
		chip = colorWhiteTransparent
	}
	onChip := ui.Theme().Colors.OnSurface
	chips := make([]ui.Node, len(b.Destination.Tags))
	for i, t := range b.Destination.Tags {
		chips[i] = tagChip(t, 16, 32, chip, onChip)
	}
	return ui.Padding{
		Padding: screenX(),
		Child: ui.Wrap{
			Spacing:    ui.P(6.0),
			RunSpacing: ui.P(6.0),
			Children:   chips,
		},
	}
}

// activityRow gives 1 activity of the trip.
func activityRow(a model.Activity) ui.Node {
	d := screen()
	return ui.Padding{
		Padding: ui.Insets{
			Left:  d.ScreenHorizontal,
			Top:   paddingVertical,
			Right: d.ScreenHorizontal,
		},
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
						MaxLines: ui.P(1),
						Overflow: ui.P("ellipsis"),
						Style:    ui.Theme().Text.BodyMedium,
					},
					ui.Text{
						Data:     a.Description,
						MaxLines: ui.P(2),
						Overflow: ui.P("ellipsis"),
						Style:    ui.Theme().Text.BodySmall,
					},
				),
			}},
		)},
	}
}
