package ui

// This file holds the node types that no Flutter widget carries. Each other
// node stands in widgets_gen.go, which nodegen writes from the public API of
// Flutter.
//
// A node is a plain Go struct, and the Dart host builds 1 real Flutter widget
// from each one. A nil Node means no child, and a nil pointer field means "the
// default of Flutter".

// Node is a description of 1 Flutter widget. Only this package declares one.
type Node interface{ isNode() }

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

func (Keyed) isNode() {}
func (Watch) isNode() {}

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

func (TextInput) isNode() {}

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
// axis extent. Read decision 273.
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

func (Pages) isNode() {}
