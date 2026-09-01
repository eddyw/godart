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
