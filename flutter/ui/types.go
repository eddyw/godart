package ui

// This file holds the value types that a field of the generated node set
// reads. Each one is a plain Go struct or a named scalar, and the host of
// flutter/goflutter converts it to the Flutter type of the same meaning.
//
// An optional field is a pointer. A nil pointer means "the default of Flutter",
// and the host then leaves the Dart parameter out. Write P to set one.

// P gives a pointer to a value. An optional field of a node needs one, and Go
// takes no address of a literal.
func P[T any](v T) *T { return &v }

// Color holds a 32-bit ARGB value.
type Color uint32

// RGB gives an opaque color.
func RGB(r, g, b uint8) Color { return ARGB(0xff, r, g, b) }

// ARGB gives a color with an alpha channel.
func ARGB(a, r, g, b uint8) Color {
	return Color(uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

// Insets holds the 4 edges of a padding, a margin, or an inset.
type Insets struct {
	Left   float64
	Top    float64
	Right  float64
	Bottom float64
}

// AllSides gives the same inset on each edge.
func AllSides(v float64) Insets { return Insets{Left: v, Top: v, Right: v, Bottom: v} }

// Sides gives 1 inset on the 2 horizontal edges and 1 on the 2 vertical edges.
func Sides(h, v float64) Insets { return Insets{Left: h, Top: v, Right: h, Bottom: v} }

// Offset holds a shift in logical pixels.
type Offset struct {
	Dx float64
	Dy float64
}

// Constraints holds the size limits of a box. A value of 0 for a maximum means
// no limit.
type Constraints struct {
	MinWidth  float64
	MaxWidth  float64
	MinHeight float64
	MaxHeight float64
}

// TextStyle holds the part of the Flutter TextStyle that a node field reads.
//
// Weight takes "thin", "light", "normal", "medium", "semibold", "bold", or
// "black". Decoration takes "underline", "overline", or "through".
type TextStyle struct {
	Color         *Color
	Size          *float64
	Weight        string
	Italic        bool
	LetterSpacing *float64
	WordSpacing   *float64
	Height        *float64
	Decoration    string
	Family        string
}
