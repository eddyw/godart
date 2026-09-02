package ui

// This file holds the value types that name 1 object of Flutter.
//
// A Go tree is data, and it rebuilds on each change. A Go value thus carries
// no FocusNode, no ScrollController, and no WidgetStatesController. A new tree
// would give a new object, and the widget would lose its focus, its scroll
// offset, or its pressed state.
//
// A Go program names the object instead. The host holds 1 object for each
// name, and it makes the object on the first read. The host disposes the
// object when the last widget that reads that name leaves the element tree. 2
// widgets that read 1 name at the same time give a loud report, because 1
// object cannot serve 2 widgets.
//
// Each key also carries an optional callback. The host calls it when the
// object changes, thus the state reaches Go again. Such a callback writes a
// signal, and the graph then rebuilds the boundary that reads that signal.
//
// Give the field a key in each build. A field that holds a key in 1 build and
// no key in the next changes the shape of the widget tree, thus Flutter builds
// the widget again and the widget loses its state.

// FocusKey names 1 Flutter FocusNode.
//
//	ui.TextField{FocusNode: &ui.FocusKey{Name: "email"}}
type FocusKey struct {
	// Name identifies the node. The whole application shares 1 table, thus 2
	// fields that need 2 nodes need 2 names.
	Name string

	// OnFocus reports each change of the focus of this node.
	OnFocus func(bool)
}

// ScrollKey names 1 Flutter ScrollController.
//
//	ui.ListViewBuilder{Controller: &ui.ScrollKey{Name: "rows"}}
type ScrollKey struct {
	Name string

	// Initial gives the offset of the first read alone. The host owns the
	// offset afterwards, thus a later change of this field reaches no
	// controller.
	Initial float64

	// OnScroll reports the offset after each scroll. The host holds the value
	// until the end of the frame, because a rebuild during the layout of
	// Flutter is an error.
	OnScroll func(float64)
}

// StatesKey names 1 Flutter WidgetStatesController.
//
//	ui.ElevatedButton{StatesController: &ui.StatesKey{Name: "send"}}
type StatesKey struct {
	Name string

	// OnStates reports the states that hold, for example "hovered" and
	// "pressed". The order follows the enum WidgetState of Flutter.
	OnStates func([]string)
}
