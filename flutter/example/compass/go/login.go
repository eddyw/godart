package app

import (
	"math"

	"gart.dev/ui"
	"gart.dev/ui/material"
	"gart.dev/ui/svg"
)

// LoginModel holds the login screen. The 2 fields carry the account of the
// sample, thus the user needs no keyboard to enter.
type LoginModel struct {
	app      *App
	email    ui.Signal[string]
	password ui.Signal[string]

	// Login runs the request of the token.
	Login *ui.Command
}

func newLoginModel(a *App) *LoginModel {
	m := &LoginModel{app: a}
	m.email.Set("email@example.com")
	m.password.Set("password")
	m.Login = ui.NewCommand(m.run)
	return m
}

// run reads the token and it opens the home screen.
func (m *LoginModel) run() error {
	if err := m.app.repos.Auth.Login(m.email.Peek(), m.password.Peek()); err != nil {
		toast(textErrorLogin, textTryAgain, m.Login.Run)
		return err
	}
	ui.Post(func() { m.app.replace(route{name: screenHome}) })
	return nil
}

// Build gives the tree of the login screen.
func (m *LoginModel) Build() ui.Node {
	return material.Scaffold{Body: ui.Watch{Build: m.body}}
}

func (m *LoginModel) body() ui.Node {
	return ui.SingleChildScrollView{Child: ui.Column{
		CrossAxisAlignment: ui.P("center"),
		MainAxisAlignment:  ui.P("spaceEvenly"),
		Children: ui.Kids(
			tiltedCards(),
			ui.Padding{
				Padding: screenSides(),
				Child: ui.Column{
					CrossAxisAlignment: ui.P("stretch"),
					Children: ui.Kids(
						ui.TextInput{
							Value:     m.email.Get(),
							OnChanged: func(s string) { m.email.Set(s) },
						},
						ui.SizedBox{Height: ui.P(paddingVertical)},
						ui.TextInput{
							Value:     m.password.Get(),
							Obscure:   true,
							OnChanged: func(s string) { m.password.Set(s) },
						},
						ui.SizedBox{Height: ui.P(paddingVertical)},
						m.button(),
					),
				},
			},
		),
	}}
}

func (m *LoginModel) button() ui.Node {
	if m.Login.Busy() {
		return ui.Center{Child: material.CircularProgressIndicator{}}
	}
	return material.FilledButton{
		Child:     ui.Text{Data: textLogin},
		OnPressed: m.Login.Run,
	}
}

// The 3 cards of the login screen. The image host of the sample answers each
// of the 3 requests.
const (
	cardLeft  = "https://rstr.in/google/tripedia/g2i0BsYPKW-"
	cardRight = "https://rstr.in/google/tripedia/980sqNgaDRK"
	cardFront = "https://rstr.in/google/tripedia/pHfPmf3o5NU"
)

// tiltedCards gives the 3 pictures of the head of the login screen. The middle
// 1 carries the logo of the sample over a dark filter.
func tiltedCards() ui.Node {
	return ui.ConstrainedBox{
		Constraints: ui.Constraints{MaxWidth: 300},
		Child: ui.AspectRatio{
			AspectRatio: 1,
			Child: ui.Stack{
				Alignment: ui.P("center"),
				Children: ui.Kids(
					ui.Positioned{
						Left:  ui.P(0.0),
						Child: card(cardLeft, 200, 273, -3.83/360, false),
					},
					ui.Positioned{
						Right: ui.P(0.0),
						Child: card(cardRight, 180, 230, 3.46/360, false),
					},
					card(cardFront, 225, 322, 0, true),
				),
			},
		},
	}
}

// card gives 1 picture of the login screen. The turn holds a part of a whole
// circle, exactly as the AlwaysStoppedAnimation of the sample does.
func card(url string, width, height, turn float64, title bool) ui.Node {
	kids := ui.Kids(image(url, title))
	if title {
		kids = append(kids, ui.Center{Child: svg.SvgPictureAsset{
			AssetName: "assets/logo.svg",
		}})
	}
	return ui.TransformRotate{
		Angle: turn * 2 * math.Pi,
		Child: ui.SizedBox{
			Width:  ui.P(width),
			Height: ui.P(height),
			Child: ui.ClipRRect{
				BorderRadius: ui.P(ui.AllCorners(20)),
				Child: ui.Stack{
					Fit:       ui.P("expand"),
					Alignment: ui.P("center"),
					Children:  kids,
				},
			},
		},
	}
}

// image gives 1 picture of a card. The card of the title draws the picture
// through a dark filter, thus the logo stands out.
func image(url string, dark bool) ui.Node {
	out := ui.Picture{Image: ui.NetworkImage(url), Fit: "cover"}
	if dark {
		out.Color = ui.P(ui.Color(0x80000000))
		out.BlendMode = "darken"
	}
	return out
}
