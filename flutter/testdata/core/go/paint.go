package app

import "gart.dev/ui"

var paintCalls int
var savedCanvas ui.Canvas

func PaintCalls() int { return paintCalls }

func PaintTree() ui.Node {
	return PaintRevision(0)
}

func PaintRevision(revision int) ui.Node {
	return ui.CustomPaint{Size: ui.P(ui.Size{Width: 40, Height: 40}), Painter: ui.P(ui.Painter{
		Revision: revision,
		Paint: func(canvas ui.Canvas, width, height float64) {
			paintCalls++
			savedCanvas = canvas
			canvas.DrawRect(ui.Rect{Right: width, Bottom: height}, ui.Color(0xffff0000))
			canvas.DrawCircle(ui.Offset{Dx: 20, Dy: 20}, 5, ui.Color(0xff00ff00))
			canvas.DrawLine(ui.Offset{Dx: 0, Dy: 35}, ui.Offset{Dx: 40, Dy: 35}, 2, ui.Color(0xff0000ff))
		},
	})}
}

func EscapedCanvas() { savedCanvas.DrawCircle(ui.Offset{}, 1, 0) }
