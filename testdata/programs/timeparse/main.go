package main

import (
	"fmt"
	"time"
)

// The program reads each time in UTC, thus the output stays equal on each
// machine.

func show(t time.Time) {
	fmt.Printf("%d-%02d-%02d %02d:%02d:%02d.%09d %s %s yday=%d\n",
		t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second(),
		t.Nanosecond(), t.Weekday(), t.Location(), t.YearDay())
}

func accessors() {
	show(time.Date(2026, time.August, 30, 14, 25, 7, 123456789, time.UTC))
	show(time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC))
	show(time.Date(1969, time.December, 31, 23, 59, 59, 0, time.UTC))
	show(time.Unix(0, 0).In(time.UTC))

	y, m, d := time.Date(2000, time.March, 1, 0, 0, 0, 0, time.UTC).Date()
	h, mi, s := time.Date(2000, time.March, 1, 13, 45, 2, 0, time.UTC).Clock()
	fmt.Println(y, m, d, h, mi, s)
}

func parseGood() {
	layouts := []struct{ layout, value string }{
		{time.RFC3339, "2026-08-30T14:25:07Z"},
		{time.RFC3339, "2026-08-30T14:25:07+02:00"},
		{time.RFC3339Nano, "2026-08-30T14:25:07.123456789Z"},
		{time.DateOnly, "2026-08-30"},
		{time.DateTime, "2026-08-30 14:25:07"},
		{time.ANSIC, "Mon Aug 30 14:25:07 2026"},
		{time.RFC822Z, "30 Aug 26 14:25 +0000"},
		{"2006-01-02 3:04PM", "2026-08-30 3:04PM"},
		{"2006-01-02 3:04PM", "2026-08-30 11:59AM"},
		{"2006-01-02 15:04:05.000", "2026-08-30 14:25:07.250"},
		{"2006-01-02T15:04:05", "2026-08-30T14:25:07.9876"},
		{"Jan _2 2006", "Aug  5 2026"},
		{"2006-002", "2026-242"},
		{"06/1/2", "26/8/30"},
		{"2006-01-02 -07", "2026-08-30 -05"},
		{"2006-01-02 -070000", "2026-08-30 +013000"},
	}
	for _, c := range layouts {
		t, err := time.Parse(c.layout, c.value)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Printf("%-22s %s | %s\n", c.value, t.UTC().Format(time.RFC3339Nano),
			t.Format(c.layout))
	}
}

func parseBad() {
	cases := []struct{ layout, value string }{
		{time.RFC3339, "2026-08-30"},
		{time.DateOnly, "2026-13-30"},
		{time.DateOnly, "2026-02-30"},
		{time.DateOnly, "2026-08-30 extra"},
		{time.TimeOnly, "25:00:00"},
		{time.ANSIC, "Xyz Aug 30 14:25:07 2026"},
		{"2006-002", "2026-400"},
	}
	for _, c := range cases {
		_, err := time.Parse(c.layout, c.value)
		fmt.Println(err)
	}
}

func roundTrip() {
	base := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	for i := 0; i < 5; i++ {
		text := base.Format(time.RFC3339)
		back, err := time.Parse(time.RFC3339, text)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println(text, back.Equal(base), back.Sub(base))
		base = base.Add(97 * 24 * time.Hour)
	}
}

func inLocation() {
	t, err := time.ParseInLocation(time.DateTime, "2026-08-30 14:25:07", time.UTC)
	fmt.Println(t.Format(time.RFC3339), err)
}

// localZone reads the zone of the system. The Go run and the Dart run happen on
// 1 machine, thus the 2 outputs stay equal.
func localZone() {
	t := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.Local)
	text := t.Format(time.RFC3339)
	back, err := time.Parse(time.RFC3339, text)
	fmt.Println(back.Equal(t), back.Location() == time.Local, err)
	fmt.Println(back.Format(time.RFC3339) == text, back.Hour(), back.Minute())

	named, err := time.Parse(time.RFC1123, "Sun, 30 Aug 2026 14:25:07 UTC")
	fmt.Println(named.Format(time.RFC3339), named.Location(), err)
}

func main() {
	accessors()
	parseGood()
	parseBad()
	roundTrip()
	inLocation()
	localZone()
}
