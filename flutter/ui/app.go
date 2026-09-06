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

// HostTheme holds the part of the Flutter theme that the host reads with
// Theme.of(context). A Go value holds no BuildContext, thus the host writes it
// before each build.
//
// A program that calls UseTheme reads its own theme through Theme(), and the
// host value then gives the brightness alone.
type HostTheme struct {
	Dark      bool
	Primary   uint32
	Surface   uint32
	OnSurface uint32
	TextSize  float64
}

// ThemeData holds the theme that a Go build reads through Theme().
//
// A program that calls UseTheme reads the theme that it declared. A program
// that calls no UseTheme reads the 4 values of the host, and Colors then holds
// the 3 colors of the host on each field that derives from them.
type ThemeData struct {
	Dark     bool
	Colors   ColorScheme
	Text     TextTheme
	TextSize float64
	// Input holds the input decoration of the theme. It is nil when the
	// program declared none.
	Input *InputDecorationStyle
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
	themeSig Signal[HostTheme]
	mediaSig Signal[MediaData]

	lightTheme *ThemeStyle
	darkTheme  *ThemeStyle
)

// UseTheme names the theme of the application. Theme() then gives the theme
// of the brightness that the host reports.
//
// Give the same 2 values to the fields Theme and DarkTheme of
// material.MaterialApp, thus Flutter draws with the theme that the Go build
// reads. A nil value for the dark theme makes each brightness read the light
// theme.
func UseTheme(light, dark *ThemeStyle) {
	lightTheme = light
	darkTheme = dark
	Dirty()
}

// Theme gives the current theme, and it records a dependency. A change of the
// theme rebuilds only the boundaries that read it.
func Theme() ThemeData {
	h := themeSig.Get()
	style := lightTheme
	if h.Dark && darkTheme != nil {
		style = darkTheme
	}
	if style == nil {
		return hostThemeData(h)
	}
	out := ThemeData{Dark: h.Dark, TextSize: h.TextSize, Input: style.InputDecoration}
	if style.ColorScheme != nil {
		out.Colors = style.ColorScheme.Resolve()
	} else {
		out.Colors = hostThemeData(h).Colors
	}
	if style.TextTheme != nil {
		out.Text = *style.TextTheme
	}
	if s := out.Text.BodyMedium; s != nil && s.Size != nil {
		out.TextSize = *s.Size
	}
	return out
}

// hostThemeData builds the theme of a program that calls no UseTheme. Each
// color that Flutter derives reads the 3 colors that the host wrote.
func hostThemeData(h HostTheme) ThemeData {
	brightness := "light"
	if h.Dark {
		brightness = "dark"
	}
	c := ColorScheme{
		Brightness:  brightness,
		Primary:     Color(h.Primary),
		OnPrimary:   Color(h.Surface),
		Secondary:   Color(h.Primary),
		OnSecondary: Color(h.Surface),
		Error:       ARGB(0xff, 0xb3, 0x26, 0x1e),
		OnError:     ARGB(0xff, 0xff, 0xff, 0xff),
		Surface:     Color(h.Surface),
		OnSurface:   Color(h.OnSurface),
	}
	return ThemeData{Dark: h.Dark, Colors: c.Resolve(), TextSize: h.TextSize}
}

// SetTheme writes the theme of the host. The host calls it.
func SetTheme(t HostTheme) { themeSig.Set(t) }

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
	themeSig = Signal[HostTheme]{}
	lightTheme = nil
	darkTheme = nil
	mediaSig = Signal[MediaData]{}
	resetCalls()
}
