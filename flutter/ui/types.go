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

// BorderSide holds 1 edge of a border. A nil field takes the default of
// Flutter: a black line of 1 logical pixel.
//
// Style takes "solid" or "none".
type BorderSide struct {
	Color *Color
	Width *float64
	Style string
}

// Radius holds the 4 corners of a rounded rectangle, in logical pixels.
type Radius struct {
	TopLeft     float64
	TopRight    float64
	BottomLeft  float64
	BottomRight float64
}

// AllCorners gives the same radius on each corner.
func AllCorners(v float64) Radius {
	return Radius{TopLeft: v, TopRight: v, BottomLeft: v, BottomRight: v}
}

// Shape holds the outline of a widget.
//
// Kind takes "rounded", "stadium", "circle", "beveled", or "rectangle". An
// empty Kind gives a rectangle. Radius reads for "rounded" and for "beveled"
// alone.
type Shape struct {
	Kind   string
	Radius Radius
	Side   *BorderSide
}

// Density holds the visual density of a Material widget. Each value stands
// between -4 and 4, and 0 gives the density of the theme.
type Density struct {
	Horizontal float64
	Vertical   float64
}

// IconStyle holds the part of the Flutter IconThemeData that a node field
// reads. The name differs from the Flutter name, because the node set holds a
// widget IconTheme.
type IconStyle struct {
	Color   *Color
	Size    *float64
	Opacity *float64
	Fill    *float64
	Weight  *float64
}

// StateColor gives 1 color for each state of a Material widget. A nil field
// takes the color of Default, and a nil Default takes the default of Flutter.
type StateColor struct {
	Default  *Color
	Hovered  *Color
	Focused  *Color
	Pressed  *Color
	Selected *Color
	Disabled *Color
}

// Size holds a width and a height in logical pixels.
type Size struct {
	Width  float64
	Height float64
}

// ImageSource names 1 image and the place that holds it.
//
// Kind takes "network", "asset", or "memory". Src holds the URL of a network
// image or the name of an asset. Bytes holds the encoded image of a memory
// image, and Scale reads for "network" and for "memory" alone. Package names
// the Dart package that owns an asset.
//
// A struct with 1 kind and 1 source beats 1 optional field for each kind,
// because 2 fields set at once then name no image.
//
// No kind names a file of the disk. FileImage reads dart:io, and the host also
// builds for the web. A Go program reads the file and gives the bytes.
type ImageSource struct {
	Kind    string
	Src     string
	Bytes   []byte
	Scale   *float64
	Package string
}

// NetworkImage gives the image of a URL.
func NetworkImage(url string) ImageSource {
	return ImageSource{Kind: "network", Src: url}
}

// AssetImage gives the image of an asset of the application.
func AssetImage(name string) ImageSource {
	return ImageSource{Kind: "asset", Src: name}
}

// MemoryImage gives the image of a block of encoded bytes.
func MemoryImage(b []byte) ImageSource {
	return ImageSource{Kind: "memory", Bytes: b}
}

// Shadow holds 1 shadow under a box. A Color of 0 gives black, which is the
// default of Flutter.
type Shadow struct {
	Color        Color
	Offset       Offset
	BlurRadius   float64
	SpreadRadius float64
}

// Gradient holds the colors that a box fades through.
//
// Kind takes "linear" or "radial". Begin and End name the 2 ends of a linear
// gradient, and Center and Radius place a radial 1. Each name stands in the
// alignment table of the host, for example "topLeft".
//
// Stops holds 1 fraction for each color, and a length that differs spreads the
// colors evenly. Fewer than 2 colors give no gradient.
type Gradient struct {
	Kind   string
	Colors []Color
	Stops  []float64
	Begin  string
	End    string
	Center string
	Radius float64
}

// Decoration holds the paint of a box, and it gives a BoxDecoration.
//
// Shape takes "rectangle" or "circle", and a circle reads no Radius. Border
// holds the same side on each of the 4 edges. Fit names a value of the enum
// BoxFit, for example "cover", and it reads for Image alone.
type Decoration struct {
	Color    *Color
	Image    *ImageSource
	Fit      string
	Border   *BorderSide
	Radius   *Radius
	Shadows  []Shadow
	Gradient *Gradient
	Shape    string
}

// ButtonStyle holds the part of the Flutter ButtonStyle that a node field
// reads.
//
// A color carries 1 value for each state, and each other field carries 1 value
// for the whole button. A button that needs a shape or a padding of its own
// for each state reads a theme.
//
// Alignment and MouseCursor each name a value of a table of the host.
// TapTargetSize takes "padded" or "shrinkWrap".
type ButtonStyle struct {
	BackgroundColor  *StateColor
	ForegroundColor  *StateColor
	OverlayColor     *StateColor
	ShadowColor      *StateColor
	SurfaceTintColor *StateColor
	IconColor        *StateColor
	TextStyle        *TextStyle
	Elevation        *float64
	IconSize         *float64
	Padding          *Insets
	MinimumSize      *Size
	MaximumSize      *Size
	FixedSize        *Size
	Side             *BorderSide
	Shape            *Shape
	Alignment        string
	VisualDensity    *Density
	MouseCursor      string
	TapTargetSize    string
	AnimationMillis  *int
	EnableFeedback   *bool
}
