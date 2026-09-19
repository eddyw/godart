package ui

// This file holds the node types that no Flutter widget carries. Each other
// node stands in widgets_gen.go, which nodegen writes from the public API of
// Flutter.
//
// A node is a plain Go struct, and the Dart host builds 1 real Flutter widget
// from each one. A nil Node means no child, and a nil pointer field means "the
// default of Flutter".

// Mark seals the node set. A struct holds a node type when it declares
// IsNode, and the writer of that struct needs the name of this package for
// the answer. A pack of another package thus joins the set on purpose, and no
// value joins it by chance.
type Mark struct{}

// Node is a description of 1 Flutter widget.
type Node interface{ IsNode() Mark }

// Kids gives a child list. It saves the element type of a slice literal.
func Kids(n ...Node) []Node { return n }

// Keyed gives the child a key. Flutter then keeps the state of the child across
// a move in a list. The generated set carries no key, because a Go value holds
// no Flutter Key.
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

func (Keyed) IsNode() Mark      { return Mark{} }
func (Watch) IsNode() Mark      { return Mark{} }
func (Anchor) IsNode() Mark     { return Mark{} }
func (ShaderMask) IsNode() Mark { return Mark{} }
func (Picture) IsNode() Mark    { return Mark{} }
func (Blur) IsNode() Mark       { return Mark{} }

// Blur draws the content behind the child through a blur, and it then draws
// the child.
//
// The generated BackdropFilter node carries no filter, because the parameter
// of Flutter takes an ImageFilter and a Go value builds none. SigmaX and
// SigmaY each hold the width of the blur in logical pixels. BlendMode names a
// value of the Flutter enum BlendMode, and an empty text gives "srcOver".
type Blur struct {
	SigmaX    float64
	SigmaY    float64
	BlendMode string
	Child     Node
}

// AskText runs Go work that the host waits for, and it gives an answer. The
// text names the value of a Flutter enum, for example the direction of a
// dismiss. Read the Ask type for the form with no argument.
type AskText func(string) bool

// Anchor marks the place of the tree where a call of the host runs.
//
// DateRange, SnackBar, and each dialog of Flutter need a BuildContext under
// the application. A Go value holds no such context, thus the host keeps the
// context of this node while the node stands on the screen. Put 1 Anchor under
// the Home of the application.
type Anchor struct {
	Child Node
}

// ShaderMask paints the child, and it then puts the gradient over each pixel
// that the child drew.
//
// BlendMode names a value of the Flutter enum BlendMode, for example "srcIn".
// An empty BlendMode gives "modulate", the default of Flutter. The generated
// node set holds no ShaderMask, because the constructor of Flutter takes a
// function that builds a Shader from a Rect.
type ShaderMask struct {
	Gradient  Gradient
	BlendMode string
	Child     Node
}

// Picture draws an image, and it gives a node of its own while the image loads
// and after a failure.
//
// The generated Image node holds no such node, because Flutter takes a builder
// that reads a BuildContext. Fit and BlendMode each name a value of a Flutter
// enum, for example "cover" and "darken".
type Picture struct {
	Image     ImageSource
	Width     *float64
	Height    *float64
	Fit       string
	Color     *Color
	BlendMode string
	Loading   Node
	Error     Node
}

// TextInput is a text field whose value Go owns. The host keeps the Flutter
// controller and the focus across a rebuild that Go drives, thus the cursor
// stays. The generated TextField holds no controller, because a Go value holds
// no Flutter object.
type TextInput struct {
	Value     string
	Hint      string
	Label     string
	Obscure   bool
	OnChanged func(string)
	OnSubmit  func(string)
}

func (TextInput) IsNode() Mark { return Mark{} }

// OrientBuilder builds 1 subtree from the orientation of the parent. The
// argument takes "portrait" or "landscape". The rule of Builder applies.
//
// The name differs from the Flutter name, because the node set holds the
// widget OrientationBuilder.
//
//gart:sync
type OrientBuilder func(string) Node

// BoxBuilder builds 1 subtree from the limits of the box of the parent. A
// maximum of 0 means no limit, as in ui.Constraints. The rule of Builder
// applies.
//
// The name differs from the Flutter name, because the node set holds the
// widget LayoutBuilder.
//
//gart:sync
type BoxBuilder func(minWidth, maxWidth, minHeight, maxHeight float64) Node

// SliverBuilder builds the slivers that stand over a scroll view. The argument
// reports whether the inner list holds the primary scroll. The rule of Builder
// applies.
//
//gart:sync
type SliverBuilder func(innerBoxIsScrolled bool) []Node

// Task is work that the host waits for. A refresh indicator turns while the
// task runs.
//
// The Go function runs as a goroutine of its own, thus it may block. The host
// reads the handle of that goroutine, thus the function reports nothing
// itself. The indicator stops at each of the 3 ends of a goroutine: the
// function returns, a panic leaves it, and runtime.Goexit stops it.
//
//	ui.RefreshIndicator{OnRefresh: func() {
//		load()
//	}}
type Task func()

// Ask is work that the host waits for, and that gives an answer. A back button
// listener reads 1. The rule of Task applies, and a panic or runtime.Goexit
// gives the answer false.
type Ask func() bool

// ExtentBuilder gives the size of 1 item of a list on the main axis. The first
// argument holds the number of the item, and the 4 numbers after it hold the
// fields of the Flutter SliverLayoutDimensions: the scroll offset, the
// preceding scroll extent, the main axis extent of the viewport, and the cross
// axis extent.
//
// A result below 0 means "no item at that number", and Flutter then holds the
// end of the list. A size of 0 is a valid size. The rule of Builder applies.
//
//gart:sync
type ExtentBuilder func(index int, scrollOffset, precedingScrollExtent, viewportMainAxisExtent, crossAxisExtent float64) float64

// Pages holds the stack of the Flutter navigator. The host builds 1 page for
// each child, and the last child stands on top. A gesture of the platform and
// the back button of the system each pop the top page, and the host then calls
// OnPop with the number of that page.
//
// Keys names 1 key for each page. Flutter keeps the state of a page whose key
// stays in the list across a rebuild.
//
// The generated Navigator carries no page, because a Go value holds no Flutter
// Page.
type Pages struct {
	Keys     []string
	Children []Node
	OnPop    func(int)
}

func (Pages) IsNode() Mark { return Mark{} }
