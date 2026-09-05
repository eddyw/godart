package app

import "gart.dev/ui"

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
		return err
	}
	ui.Post(func() { m.app.replace(route{name: screenHome}) })
	return nil
}

// Build gives the tree of the login screen.
func (m *LoginModel) Build() ui.Node {
	return ui.Scaffold{Body: ui.SafeArea{Child: ui.Watch{Build: m.body}}}
}

func (m *LoginModel) body() ui.Node {
	kids := ui.Kids(
		ui.Text{
			Data:  "compass",
			Style: &ui.TextStyle{Size: ui.P(48.0), Weight: "w300"},
		},
		ui.SizedBox{Height: ui.P(32.0)},
		ui.TextInput{
			Label:     "Email",
			Value:     m.email.Get(),
			OnChanged: func(s string) { m.email.Set(s) },
		},
		ui.SizedBox{Height: ui.P(12.0)},
		ui.TextInput{
			Label:     "Password",
			Value:     m.password.Get(),
			Obscure:   true,
			OnChanged: func(s string) { m.password.Set(s) },
		},
		ui.SizedBox{Height: ui.P(24.0)},
	)
	if m.Login.Busy() {
		kids = append(kids, ui.CircularProgressIndicator{})
	} else {
		kids = append(kids, ui.FilledButton{
			Child:     ui.Text{Data: textLogin},
			OnPressed: m.Login.Run,
		})
	}
	if err := m.Login.Err(); err != nil {
		kids = append(kids, ui.Padding{
			Padding: ui.AllSides(12),
			Child: ui.Text{
				Data:      "Error while trying to login",
				TextAlign: ui.P("center"),
			},
		})
	}
	return ui.Center{Child: ui.Padding{
		Padding: ui.Sides(40, 0),
		Child: ui.Column{
			MainAxisAlignment: ui.P("center"),
			Children:          kids,
		},
	}}
}
