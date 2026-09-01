package ui

// This file holds the node set of milestone 1. A node is a plain Go struct, and
// the Dart host builds 1 real Flutter widget from each one. A nil Node means no
// child.
//
// A field of the value 0 or "" means "the default of Flutter". Milestone 3
// replaces this hand written set with a generated one, and a pointer field then
// carries the difference between "not set" and "set to the zero value".

// Node is a description of 1 Flutter widget. Only this package declares one.
type Node interface{ isNode() }

// Kids gives a child list. It saves the element type of a slice literal.
func Kids(n ...Node) []Node { return n }

// Column puts each child in a vertical line.
//
// Main and Cross each take "start", "center", "end", "between", "around",
// "evenly", or "stretch". An empty text takes the default of Flutter.
type Column struct {
	Children []Node
	Main     string
	Cross    string
}

// Row puts each child in a horizontal line.
type Row struct {
	Children []Node
	Main     string
	Cross    string
}

// Stack puts each child over the one before it.
type Stack struct {
	Children []Node
}

// Center puts the child in the middle of the free space.
type Center struct {
	Child Node
}

// Padding puts free space around the child. All wins over the 4 sides when it
// is not 0.
type Padding struct {
	All    float64
	Left   float64
	Top    float64
	Right  float64
	Bottom float64
	Child  Node
}

// SizedBox holds a fixed size. A width or a height of 0 takes the size of the
// child.
type SizedBox struct {
	Width  float64
	Height float64
	Child  Node
}

// Expanded gives the child the free space of a Column or of a Row.
type Expanded struct {
	Flex  int
	Child Node
}

// Spacer takes the free space of a Column or of a Row.
type Spacer struct {
	Flex int
}

// Container holds a color, free space, and a size around 1 child. Color is an
// ARGB value, and 0 means no color.
type Container struct {
	Color   uint32
	Padding float64
	Width   float64
	Height  float64
	Radius  float64
	Child   Node
}

// Divider draws a horizontal line.
type Divider struct {
	Height float64
}

// Text writes 1 run of text. A size of 0 and a color of 0 each take the theme.
type Text struct {
	Data  string
	Size  float64
	Bold  bool
	Color uint32
	Align string
}

// Icon draws 1 icon of the Material set. Name holds the name of the icon, for
// example "add" or "delete".
type Icon struct {
	Name  string
	Size  float64
	Color uint32
}

// Image draws 1 picture. URL wins over Asset.
type Image struct {
	URL    string
	Asset  string
	Width  float64
	Height float64
	Fit    string
}

// Progress draws the circular indicator of Material.
type Progress struct {
	Size float64
}

// ElevatedButton is the filled button of Material. A nil OnPressed gives the
// disabled form.
type ElevatedButton struct {
	Child     Node
	OnPressed func()
}

// TextButton is the flat button of Material.
type TextButton struct {
	Child     Node
	OnPressed func()
}

// IconButton holds 1 icon of the Material set.
type IconButton struct {
	Icon      string
	Tooltip   string
	OnPressed func()
}

// TextField reads text from the user. Value holds the text of the Go state, and
// the host writes it into the field only when the 2 differ.
type TextField struct {
	Value     string
	Hint      string
	Label     string
	Obscure   bool
	OnChanged func(string)
	OnSubmit  func(string)
}

// Switch holds a boolean.
type Switch struct {
	Value     bool
	OnChanged func(bool)
}

// Checkbox holds a boolean.
type Checkbox struct {
	Value     bool
	OnChanged func(bool)
}

// InkWell reads a tap over its child, and it draws the ripple of Material.
type InkWell struct {
	Child Node
	OnTap func()
}

// Scaffold holds the frame of 1 screen.
type Scaffold struct {
	AppBar Node
	Body   Node
	Fab    Node
}

// AppBar holds the bar at the top of a Scaffold.
type AppBar struct {
	Title   Node
	Actions []Node
}

// ListView holds a list that Flutter builds in 1 pass.
type ListView struct {
	Children []Node
	Padding  float64
}

// ListViewBuilder holds a lazy list. Flutter calls Item only for a row that the
// screen shows.
type ListViewBuilder struct {
	Count   int
	Item    ItemBuilder
	Padding float64
}

// Card draws the raised surface of Material.
type Card struct {
	Child   Node
	Padding float64
}

// Keyed gives the child a key. Flutter then keeps the state of the child across
// a move in a list.
type Keyed struct {
	Key   string
	Child Node
}

// Watch marks a rebuild boundary. The host runs Build under a tracked scope,
// and a change of a signal that Build read rebuilds this subtree alone.
type Watch struct {
	Build Builder
}

// Builder builds 1 node subtree. The Dart host calls a value of this type and
// reads the result at once. A function that can block returns the zero value
// there, and no report reaches the program, thus the compiler rejects such a
// function. Read the section "Milestone 2" of docs/flutter-design.md.
//
//gart:sync
type Builder func() Node

// ItemBuilder builds 1 row of a lazy list. The rule of Builder applies.
//
//gart:sync
type ItemBuilder func(int) Node

func (Column) isNode()          {}
func (Row) isNode()             {}
func (Stack) isNode()           {}
func (Center) isNode()          {}
func (Padding) isNode()         {}
func (SizedBox) isNode()        {}
func (Expanded) isNode()        {}
func (Spacer) isNode()          {}
func (Container) isNode()       {}
func (Divider) isNode()         {}
func (Text) isNode()            {}
func (Icon) isNode()            {}
func (Image) isNode()           {}
func (Progress) isNode()        {}
func (ElevatedButton) isNode()  {}
func (TextButton) isNode()      {}
func (IconButton) isNode()      {}
func (TextField) isNode()       {}
func (Switch) isNode()          {}
func (Checkbox) isNode()        {}
func (InkWell) isNode()         {}
func (Scaffold) isNode()        {}
func (AppBar) isNode()          {}
func (ListView) isNode()        {}
func (ListViewBuilder) isNode() {}
func (Card) isNode()            {}
func (Keyed) isNode()           {}
func (Watch) isNode()           {}
