// nodefilter:values
package ui

// PaintFunc runs during Flutter paint. It must not block or retain the Canvas.
//
//gart:sync
type PaintFunc func(canvas Canvas, width, height float64)

// Painter draws a layer of CustomPaint. Paint must be non-nil.
// Change Revision when the callback or its output changes.
type Painter struct {
	Paint    PaintFunc
	Revision int
}

// Canvas is valid only during its PaintFunc call.
type Canvas struct {
	rectangle func(float64, float64, float64, float64, Color)
	circle    func(float64, float64, float64, Color)
	line      func(float64, float64, float64, float64, float64, Color)
}

func (c Canvas) DrawRect(rect Rect, color Color) {
	c.rectangle(rect.Left, rect.Top, rect.Right, rect.Bottom, color)
}

func (c Canvas) DrawCircle(center Offset, radius float64, color Color) {
	c.circle(center.Dx, center.Dy, radius, color)
}

func (c Canvas) DrawLine(start, end Offset, width float64, color Color) {
	c.line(start.Dx, start.Dy, end.Dx, end.Dy, width, color)
}
