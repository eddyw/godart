package main

import (
	"fmt"
	"time"
)

// layouts covers each chunk of a layout that the scanner names.
var layouts = []string{
	time.Layout,
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC3339,
	time.RFC3339Nano,
	time.Kitchen,
	time.Stamp,
	time.StampMilli,
	time.StampMicro,
	time.StampNano,
	time.DateTime,
	time.DateOnly,
	time.TimeOnly,
	"2006-01-02T15:04:05.000Z07:00",
	"2006-01-02 15:04:05,999",
	"Monday, January 2, 2006 at 3:04:05pm -07:00:00",
	"Jan _2 002 __2 15:04:05 -070000",
	"06-1-2 3:4:5 PM Z07 Z0700 Z070000",
	"-07 -0700 -07:00:00 MST",
}

func main() {
	// The example of the errors package of the Go repository reads this date.
	fmt.Println(time.Date(1989, 3, 15, 22, 30, 0, 0, time.UTC))

	dates := []time.Time{
		time.Date(1989, 3, 15, 22, 30, 0, 0, time.UTC),
		time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC),
		time.Date(1900, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(1969, 12, 31, 23, 59, 59, 999999999, time.UTC),
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 29, 6, 5, 4, 123456789, time.UTC),
		time.Date(2262, 4, 11, 0, 0, 0, 0, time.UTC),
	}
	for _, t := range dates {
		fmt.Println(t.String(), t.UnixNano())
		for _, layout := range layouts {
			fmt.Println(t.Format(layout))
		}
	}

	// A value outside its range moves into the next unit.
	fmt.Println(time.Date(2000, 13, 32, 25, 61, 61, 1500000000, time.UTC))
	fmt.Println(time.Date(2000, 0, 0, -1, -1, -1, -1, time.UTC))
	fmt.Println(time.Date(2001, 25, 1, 0, 0, 0, 0, time.UTC))

	// A round trip through the local zone keeps the instant.
	local := time.Date(2026, 8, 29, 6, 5, 4, 0, time.Local)
	fmt.Println(local.In(time.UTC).Format(time.RFC3339))
	fmt.Println(local.Location(), time.UTC.String())

	// The day of the week and the day of the year read the same count of days.
	for _, day := range []int{1, 59, 60, 61, 365, 366} {
		t := time.Date(2000, 1, day, 0, 0, 0, 0, time.UTC)
		fmt.Println(t.Format("Monday 002 2006-01-02"))
	}

	fmt.Println(time.Month(0), time.Month(13), time.Weekday(7))
	fmt.Println(time.January, time.December, time.Sunday, time.Saturday)
}
