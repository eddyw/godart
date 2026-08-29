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
	// offset holds the seconds east of UTC of a zone of a fixed offset. Parse
	// builds such a zone when it reads an offset that the local zone does not
	// hold.
	offset int
}

// UTC holds the coordinated universal time.
var UTC = &Location{name: "UTC"}

// Local holds the time zone of the system.
var Local = &Location{name: "Local", local: true}

// FixedZone gives a Location that always reads the offset in seconds east of
// UTC.
func FixedZone(name string, offset int) *Location {
	return &Location{name: name, offset: offset}
}

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
	if l == nil {
		return 0, "UTC"
	}
	if l.local {
		return dart.ZoneOffset(nanos)
	}
	return l.offset, l.name
}

// lookupName gives the offset of a zone of the location that carries the name
// at an instant. It reports whether the location holds such a zone.
func (l *Location) lookupName(name string, nanos int64) (int, bool) {
	if l == nil || !l.local {
		return 0, false
	}
	offset, zone := dart.ZoneOffset(nanos)
	if zone != name {
		return 0, false
	}
	return offset, true
}

// offsetOf gives the offset of the location at a reading of its wall clock.
//
// The Dart runtime names the offset of an instant, and a wall clock reading is
// not such an instant. The first lookup reads the wall clock as if it were UTC,
// and the second one reads the instant that the first offset gives. A reading
// inside 1 offset of a transition of the zone can take the offset of the other
// side. docs/parity.md holds that gap.
func (l *Location) offsetOf(wall int64) int {
	if l == nil {
		return 0
	}
	if !l.local {
		return l.offset
	}
	guess, _ := dart.ZoneOffset(wall)
	offset, _ := dart.ZoneOffset(wall - int64(guess)*1000000000)
	return offset
}
