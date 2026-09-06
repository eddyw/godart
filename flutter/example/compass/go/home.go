package app

import (
	"strconv"

	"example.dev/gocompass/model"
	"gart.dev/ui"
	"gart.dev/ui/material"
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
		toast(textErrorDelete, "", nil)
		return err
	}
	list, err := m.app.repos.Booking.List()
	if err != nil {
		return err
	}
	toast(textBookingDeleted, "", nil)
	ui.Post(func() { m.bookings.Set(list) })
	return nil
}

// Bookings gives the list that the last read wrote. A test reads it.
func (m *HomeModel) Bookings() []model.BookingSummary { return m.bookings.Peek() }

// Build gives the tree of the home screen.
func (m *HomeModel) Build() ui.Node {
	return material.Scaffold{
		FloatingActionButton: material.FloatingActionButtonExtended{
			Label:     ui.Text{Data: textBookNewTrip},
			Icon:      ui.Icon{Icon: "add_location_outlined"},
			OnPressed: func() { m.app.push(route{name: screenSearch}) },
		},
		Body: ui.SafeArea{
			Top:    ui.P(true),
			Bottom: ui.P(true),
			Child:  ui.Watch{Build: m.body},
		},
	}
}

func (m *HomeModel) body() ui.Node {
	if m.Load.Busy() {
		return busy()
	}
	if m.Load.Err() != nil {
		return errorIndicator(textErrorHome, textTryAgain, m.Load.Run)
	}
	list := m.bookings.Get()
	rows := make([]ui.Node, 0, len(list)+1)
	rows = append(rows, ui.SliverToBoxAdapter{Child: ui.Padding{
		Padding: screenSides(),
		Child:   m.header(),
	}})
	for _, b := range list {
		rows = append(rows, ui.SliverToBoxAdapter{Child: m.row(b)})
	}
	return ui.CustomScrollView{Slivers: rows}
}

// header gives the picture of the account, the logout button, and the title.
func (m *HomeModel) header() ui.Node {
	user := m.user.Get()
	if user.Name == "" {
		return ui.SizedBox{}
	}
	size := screen().ProfilePicture
	return ui.Column{
		CrossAxisAlignment: ui.P("stretch"),
		Children: ui.Kids(
			ui.Row{
				MainAxisAlignment:  ui.P("spaceBetween"),
				CrossAxisAlignment: ui.P("start"),
				Children: ui.Kids(
					ui.ClipOval{Child: ui.Image{
						Image:  ui.AssetImage(user.Picture),
						Width:  ui.P(size),
						Height: ui.P(size),
					}},
					m.app.logoutButton(),
				),
			},
			ui.SizedBox{Height: ui.P(paddingVertical)},
			title(trips(user.Name)),
		),
	}
}

// title gives the head of the home screen. The gradient of the sample paints
// each pixel of the text, thus the mask reads the mode srcIn.
func title(text string) ui.Node {
	style := ui.TextStyle{Family: fontRubik}
	if s := ui.Theme().Text.HeadlineLarge; s != nil {
		style = *s
		style.Family = fontRubik
	}
	return ui.ShaderMask{
		BlendMode: "srcIn",
		Gradient: ui.Gradient{
			Kind:   "radial",
			Center: "bottomLeft",
			Radius: 2,
			Colors: []ui.Color{colorPurple700, colorPurple400},
		},
		Child: ui.Text{Data: text, Style: &style},
	}
}

// row gives 1 booking of the list. A swipe to the left removes it, and the row
// stays when the request of the server fails.
func (m *HomeModel) row(b model.BookingSummary) ui.Node {
	d := screen()
	return ui.Dismissible{
		Key:       "booking-" + strconv.Itoa(b.ID),
		Direction: ui.P("endToStart"),
		// The sample waits for the command and it keeps the row when the
		// request fails. This call runs in a goroutine of the host, thus it
		// reads the server itself and it answers with the result.
		ConfirmDismiss: func(string) bool { return m.delete(b.ID) == nil },
		Background: ui.Container{
			Color: ui.P(colorGrey1),
			Child: ui.Row{
				MainAxisAlignment: ui.P("end"),
				Children: ui.Kids(ui.Padding{
					Padding: ui.Insets{Right: paddingHorizontal},
					Child:   ui.Icon{Icon: "delete"},
				}),
			},
		},
		Child: material.InkWell{
			OnTap: func() { m.app.push(route{name: screenBooking, id: b.ID}) },
			Child: ui.Padding{
				Padding: ui.Sides(d.ScreenHorizontal, paddingVertical),
				Child: ui.Column{
					CrossAxisAlignment: ui.P("start"),
					Children: ui.Kids(
						ui.Text{Data: b.Name, Style: ui.Theme().Text.TitleLarge},
						ui.Text{
							Data:  dateRange(b.StartDate, b.EndDate),
							Style: ui.Theme().Text.BodyLarge,
						},
					),
				},
			},
		},
	}
}
