package app

import (
	"gart.dev/ui"
	"gart.dev/ui/material"
)

// busy gives the indicator of a read that runs.
func busy() ui.Node { return ui.Center{Child: material.CircularProgressIndicator{}} }

// failed gives the message of a read that failed, and a button that runs the
// read again.
func failed(msg string, retry func()) ui.Node {
	return ui.Center{Child: ui.Column{
		MainAxisAlignment: ui.P("center"),
		Children: ui.Kids(
			ui.Icon{Icon: "error"},
			ui.Padding{
				Padding: ui.AllSides(12),
				Child:   ui.Text{Data: msg, TextAlign: ui.P("center")},
			},
			material.FilledButton{Child: ui.Text{Data: textTryAgain}, OnPressed: retry},
		),
	}}
}

// tag gives the chip of 1 tag of a destination.
func tag(name string) ui.Node {
	return ui.Padding{
		Padding: ui.Insets{Left: 0, Top: 0, Right: 4, Bottom: 4},
		Child: material.Chip{
			Label:           ui.Text{Data: name},
			LabelPadding:    ui.P(ui.Sides(6, 0)),
			BackgroundColor: ui.P(ui.Color(0xFFEFEFEF)),
			Padding:         ui.P(ui.AllSides(4)),
		},
	}
}

// image gives the picture of a URL over the whole box of the parent.
func image(url string, height float64) ui.Node {
	return ui.ImageNetwork{
		Src:    url,
		Fit:    ui.P("cover"),
		Height: ui.P(height),
		Width:  ui.P(10000.0),
	}
}

// backButton gives the arrow of the top left corner.
func backButton(pop func()) ui.Node {
	return material.IconButton{
		Icon:      ui.Icon{Icon: "arrow_back"},
		OnPressed: pop,
	}
}
