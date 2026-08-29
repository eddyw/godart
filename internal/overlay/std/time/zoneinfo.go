//go:build dart

package time

import "syscall/dart"

// A Location names a time zone. The overlay holds no zone database of its own,
// thus it gives 2 locations: UTC and the local zone of the system.
type Location struct {
	name string
	// local marks the zone of the system. Such a zone reads its offset from
	// the Dart runtime at each instant, thus a change of the offset over the
	// year reaches the output.
	local bool
}

// UTC holds the coordinated universal time.
var UTC = &Location{name: "UTC"}

// Local holds the time zone of the system.
var Local = &Location{name: "Local", local: true}

// String gives the name of the location.
func (l *Location) String() string {
	if l == nil {
		return "UTC"
	}
	return l.name
}

// lookup gives the offset in seconds east of UTC and the name of the zone at an
// instant of the Unix epoch.
func (l *Location) lookup(nanos int64) (int, string) {
	if l != nil && l.local {
		return dart.ZoneOffset(nanos)
	}
	return 0, "UTC"
}

// offsetOf gives the offset of the location at a reading of its wall clock.
//
// The Dart runtime names the offset of an instant, and a wall clock reading is
// not such an instant. The first lookup reads the wall clock as if it were UTC,
// and the second one reads the instant that the first offset gives. A reading
// inside 1 offset of a transition of the zone can take the offset of the other
// side. docs/parity.md holds that gap.
func (l *Location) offsetOf(wall int64) int {
	if l == nil || !l.local {
		return 0
	}
	guess, _ := dart.ZoneOffset(wall)
	offset, _ := dart.ZoneOffset(wall - int64(guess)*1000000000)
	return offset
}
