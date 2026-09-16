//go:build dart

package time

import (
	"strings"
	"testing"
)

func TestMonotonicWallChange(t *testing.T) {
	a := Time{sec: minMonoWall + 100, mono: 1000, hasMono: true}
	b := a
	b.sec -= 60
	b.mono += 7
	if b.Sub(a) != 7 || !b.After(a) || !a.Before(b) || b.Compare(a) != 1 {
		t.Fatal("wall change altered monotonic comparisons")
	}
	b.mono = a.mono
	if !a.Equal(b) || a.Compare(b) != 0 {
		t.Fatal("equal monotonic clocks differ")
	}
	b.stripMono()
	if a.Equal(b) || a.Sub(b) != Minute || a.Compare(b) != 1 {
		t.Fatal("mixed clocks did not use wall time")
	}
}

func TestMonotonicPreservation(t *testing.T) {
	a := Now()
	if !a.hasMono || !a.Add(Second).hasMono || a.Add(Second).Sub(a) != Second {
		t.Fatal("Now or Add lost the monotonic clock")
	}
	for _, b := range []Time{a.UTC(), a.Local(), a.In(FixedZone("x", 3600)), a.AddDate(0, 0, 0), a.Round(0), a.Truncate(0), a.Round(Second), a.Truncate(Second)} {
		if b.hasMono || b.mono != 0 {
			t.Fatal("operation retained the monotonic clock")
		}
	}
	data, err := a.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b := a
	if err := b.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if b.hasMono || !a.Equal(b) {
		t.Fatal("binary round trip retained monotonic state or lost wall time")
	}
	text, err := a.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.UnmarshalText(text); err != nil {
		t.Fatal(err)
	}
	if b.hasMono || !a.Equal(b) {
		t.Fatal("text round trip differs")
	}
}

func TestMonotonicBounds(t *testing.T) {
	a := Time{sec: maxMonoWall, mono: 1, hasMono: true}
	if a.Add(Second).hasMono {
		t.Fatal("wall overflow retained the monotonic clock")
	}
	a.sec = minMonoWall
	if a.Add(-Second).hasMono {
		t.Fatal("wall underflow retained the monotonic clock")
	}
	a.sec++
	a.mono = 1<<63 - 1
	if a.Add(1).hasMono {
		t.Fatal("monotonic overflow retained the clock")
	}
	b := a
	b.mono = -1 << 63
	if a.Sub(b) != maxDuration || b.Sub(a) != minDuration {
		t.Fatal("Sub did not saturate")
	}
	if !strings.HasSuffix(b.String(), " m=-9223372036.854775808") {
		t.Fatal(b.String())
	}
	b.mono = 12
	if !strings.HasSuffix(b.String(), " m=+0.000000012") {
		t.Fatal(b.String())
	}
}
