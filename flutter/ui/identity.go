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

// TextKey names 1 Flutter TextEditingController.
//
//	ui.TextField{Controller: &ui.TextKey{Name: "email"}}
//
// The host owns the text after the first read, thus a later change of Initial
// reaches no controller. A Go program that owns the text reads ui.TextInput.
type TextKey struct {
	Name string

	// Initial gives the text of the first read alone.
	Initial string

	// OnChanged reports the text after each change.
	OnChanged func(string)
}

// AnimationKey names 1 Flutter AnimationController.
//
//	ui.FadeTransition{Opacity: &ui.AnimationKey{Name: "card", Target: 1}}
//
// The Go side names where the animation runs to, and the host runs it. The
// value of the animation stands between 0 and 1. A transition widget reads the
// value on each frame, thus the animation costs no rebuild of the Go tree.
type AnimationKey struct {
	Name string

	// Millis gives the length of 1 whole run, from 0 to 1. A value of 0 gives
	// 300 milliseconds.
	Millis int

	// Target names the value that the animation runs to. The host holds that
	// value after the end of the run, and a new Target starts a new run.
	Target float64

	// Curve names the shape of the run, for example "easeInOut".
	Curve string

	// Repeat runs the animation again and again, and it reads no Target.
	// Reverse then runs it back to the start each second time.
	Repeat  bool
	Reverse bool

	// OnEnd reports the end of a run that reaches Target. A run that a new
	// Target stops gives no report.
	OnEnd func()
}

// LinkKey names 1 Flutter LayerLink. A target widget and a follower widget
// that name 1 key hold the same link, thus the follower stands at the target.
//
//	ui.CompositedTransformTarget{Link: &ui.LinkKey{Name: "menu"}}
type LinkKey struct {
	Name string
}

// OverlapKey names 1 Flutter SliverOverlapAbsorberHandle. An absorber and an
// injector that name 1 key hold the same handle.
type OverlapKey struct {
	Name string
}

// OverlayKey names 1 Flutter OverlayPortalController.
//
//	ui.OverlayPortal{Controller: &ui.OverlayKey{Name: "tip", Show: open}}
//
// The Go side names whether the overlay child stands on the screen, and the
// host shows it or hides it after the frame.
type OverlayKey struct {
	Name string
	Show bool
}

// MenuKey names 1 Flutter MenuController.
//
// The Go side names whether the menu stands open, and the host opens it or
// closes it after the frame.
type MenuKey struct {
	Name string
	Open bool
}

// ScopeKey names 1 Flutter FocusScopeNode. A scope holds the focus of the
// widgets under it, and it gives the focus back when it stands again.
type ScopeKey struct {
	Name string
}

// StorageKey names 1 Flutter PageStorageBucket. Such a bucket keeps the scroll
// offset of each list under it, by the key of the list.
type StorageKey struct {
	Name string
}
