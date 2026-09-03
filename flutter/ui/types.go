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

// Rect holds the 4 edges of a rectangle, in logical pixels.
//
// A field of the type RelativeRect reads the same struct. Each edge of such a
// field gives the distance from that edge of the parent, and not a coordinate.
type Rect struct {
	Left   float64
	Top    float64
	Right  float64
	Bottom float64
}

// Range holds the 2 ends of a range of numbers, for example the values of a
// RangeSlider.
type Range struct {
	Start float64
	End   float64
}

// Span holds 1 piece of rich text, and it gives a Flutter TextSpan.
//
// A span with children holds the text of each child after its own text, and a
// child that names no style takes the style of the parent.
type Span struct {
	Text     string
	Style    *TextStyle
	Children []Span
}

// SliverChildren gives the children of a sliver.
//
// A Builder makes 1 child at a time, and Flutter then builds a row of a long
// list only when that row reaches the screen. A nil Builder reads Children.
type SliverChildren struct {
	// Children holds the whole list. A nil Builder reads it.
	Children []Node

	// Builder makes 1 child of the list.
	Builder ItemBuilder

	// Count gives the length of the list of the Builder. A Count of 0 gives a
	// list with no end.
	Count int
}

// GridSpec gives the layout of the children of a grid.
//
// Kind takes "count" or "extent". "count" reads Count, and it puts that many
// children on each cross axis line. "extent" reads MaxExtent, and it puts as
// many children as the width allows. An empty Kind gives "count".
type GridSpec struct {
	Kind string

	// Count gives the number of children of 1 cross axis line. A value below 1
	// gives 1.
	Count int

	// MaxExtent gives the largest size of 1 child on the cross axis.
	MaxExtent float64

	// MainSpacing and CrossSpacing give the space between 2 children.
	MainSpacing  float64
	CrossSpacing float64

	// AspectRatio gives the width of a child over its height. A value of 0
	// gives 1.
	AspectRatio float64

	// MainExtent gives the size of a child on the main axis. A nil value reads
	// AspectRatio.
	MainExtent *float64
}

// BarItem holds 1 item of a bottom navigation bar.
type BarItem struct {
	Icon            Node
	Label           string
	ActiveIcon      Node
	Tooltip         string
	BackgroundColor *Color
}

// RailItem holds 1 destination of a navigation rail.
type RailItem struct {
	Icon         Node
	Label        Node
	SelectedIcon Node
	Padding      *Insets
	Disabled     bool
}

// Step holds 1 step of a stepper.
//
// State takes "indexed", "editing", "complete", "disabled", or "error". An
// empty State gives "indexed".
type Step struct {
	Title    Node
	Content  Node
	Subtitle Node
	State    string
	Active   bool
}

// ThemeStyle holds the part of the Flutter ThemeData that a node field reads.
// The name differs from the Flutter name, because app.go holds the type
// ThemeData of the theme that a Go build reads.
//
// Brightness takes "light" or "dark". Seed gives 1 color, and Flutter builds
// the whole color scheme from it. A nil field takes the default of Flutter.
type ThemeStyle struct {
	Brightness         string
	Seed               *Color
	PrimaryColor       *Color
	ScaffoldBackground *Color
	FontFamily         string
	Density            *Density
	IconStyle          *IconStyle
}

// MenuButtonItem holds 1 button of a text selection toolbar.
//
// Kind names the meaning of the button, and Flutter writes the label of the
// platform for it. Kind takes "cut", "copy", "paste", "selectAll", "delete",
// "lookUp", "searchWeb", "share", "liveTextInput", or "custom". An empty Kind
// gives "custom", and such a button reads Label.
type MenuButtonItem struct {
	Label     string
	Kind      string
	OnPressed func()
}

// DataColumn holds 1 column of a data table.
type DataColumn struct {
	Label   Node
	Tooltip string
	Numeric bool

	// OnSort reports a tap of the header. It takes the number of the column
	// and the direction, and true means "from the smallest".
	OnSort func(int, bool)
}

// DataCell holds 1 cell of a row of a data table.
type DataCell struct {
	Child        Node
	Placeholder  bool
	ShowEditIcon bool
	OnTap        func()
}

// DataRow holds 1 row of a data table. The count of the cells must match the
// count of the columns.
type DataRow struct {
	Cells    []DataCell
	Selected bool
	OnSelect func(bool)
	Color    *StateColor
}
