package app

import (
	"example.dev/gocore/bindings/brand"
	"example.dev/gocore/bindings/cupertino"
	boundmaterial "example.dev/gocore/bindings/material"
	"example.dev/gocore/bindings/other"
	"gart.dev/ui"
)

func GoBadge(label string) ui.Node {
	return ui.Container{
		Padding: ui.P(ui.AllSides(12)),
		Child:   ui.Text{Data: label},
	}
}

func BrandTree() ui.Node {
	return ui.Column{Children: ui.Kids(
		brand.Badge{Label: "Dart brand", OnPressed: root.Add, Child: GoBadge("Go brand")},
		other.Badge{Label: "same class name"},
		ui.Watch{Build: root.counterRow},
		boundmaterial.ElevatedButton{OnPressed: root.Add, Child: ui.Text{Data: "material binding"}},
		cupertino.CupertinoButton{OnPressed: root.Add, Child: ui.Text{Data: "cupertino binding"}},
	)}
}
