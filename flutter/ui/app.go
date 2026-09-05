package ui

import "slices"

// This file holds the contact between the Go tree and the Dart host. The host
// owns the lifetime of each rebuild boundary, and Go owns the graph.
//
// The order of 1 frame:
//
//  1. A Flutter event runs a Go closure.
//  2. Each Set of a signal marks the boundaries that read it.
//  3. The host calls TakeDirty and rebuilds each boundary that the list names.

// RootID names the whole tree. TakeDirty gives it after a call of Dirty.
const RootID = -1

var hostDirty []int

func addHostDirty(id int) {
	if slices.Contains(hostDirty, id) {
		return
	}
	hostDirty = append(hostDirty, id)
}

// Dirty marks the whole tree. Take this call when the state does not stand in a
// signal.
func Dirty() { addHostDirty(RootID) }

// NewWatch gives the host 1 rebuild boundary. The host calls DisposeWatch when
// the widget leaves the tree.
func NewWatch() int {
	var s *scope
	s = newScope(func() { addHostDirty(s.id) })
	return s.id
}

// RunWatch builds 1 boundary under a tracked scope. Each signal that build
// reads marks this boundary alone on its next change.
func RunWatch(id int, build Builder) Node {
	s := scopeByID(id)
	if s == nil || !s.alive {
		return build()
	}
	var out Node
	s.enter(func() { out = build() })
	return out
}

// DisposeWatch releases 1 boundary.
func DisposeWatch(id int) {
	if s := scopeByID(id); s != nil && s.alive {
		s.drop()
	}
}

// TakeDirty gives the boundaries that must rebuild, and it clears the set. A
// value of RootID means the whole tree.
func TakeDirty() []int {
	out := hostDirty
	hostDirty = nil
	return out
}

// ThemeData holds the part of the Flutter theme that a Go build reads. A Go
// value holds no BuildContext, thus the host reads the theme and writes it
// here before each build.
type ThemeData struct {
	Dark      bool
	Primary   uint32
	Surface   uint32
	OnSurface uint32
	TextSize  float64
}

// MediaData holds the part of the media query that a Go build reads.
type MediaData struct {
	Width     float64
	Height    float64
	TextScale float64
	// TopInset is the part of the screen that the status bar and the notch
	// hold.
	TopInset float64
	// BottomInset is the part of the screen that the keyboard holds. A form
	// reads it and it moves its content above the keyboard.
	BottomInset float64
}

var (
	themeSig Signal[ThemeData]
	mediaSig Signal[MediaData]
)

// Theme gives the current theme, and it records a dependency. A change of the
// theme rebuilds only the boundaries that read it.
func Theme() ThemeData { return themeSig.Get() }

// SetTheme writes the theme. The host calls it.
func SetTheme(t ThemeData) { themeSig.Set(t) }

// Media gives the current media query, and it records a dependency.
func Media() MediaData { return mediaSig.Get() }

// SetMedia writes the media query. The host calls it.
func SetMedia(m MediaData) { mediaSig.Set(m) }

// Root is the top of the tree. A Go application gives 1 to the host.
type Root interface {
	Build() Node
}

// Reset drops the whole graph. A test calls it between 2 cases.
func Reset() {
	scopes = nil
	freeIDs = nil
	current = nil
	depth = 0
	pending = nil
	hostDirty = nil
	themeSig = Signal[ThemeData]{}
	mediaSig = Signal[MediaData]{}
	resetCalls()
}
