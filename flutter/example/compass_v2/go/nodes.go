package app

import "gart.dev/ui"

type Label struct {
	Text, Role, Font, Align string
	MaxLines                int
}

type Photo struct {
	URL, Fit, Placeholder string
	Width, Height         float64
	Darken                bool
}

type Tag struct {
	Text  string
	Large bool
}

type Input struct {
	Value     string
	Obscure   bool
	OnChanged func(string)
}

type RouteView struct {
	Route        string
	Home, Screen ui.Node
	OnBack       func()
}

type TripButton struct {
	Share     bool
	OnPressed func()
}

func (Label) IsNode() ui.Mark      { return ui.Mark{} }
func (Photo) IsNode() ui.Mark      { return ui.Mark{} }
func (Tag) IsNode() ui.Mark        { return ui.Mark{} }
func (Input) IsNode() ui.Mark      { return ui.Mark{} }
func (RouteView) IsNode() ui.Mark  { return ui.Mark{} }
func (TripButton) IsNode() ui.Mark { return ui.Mark{} }
