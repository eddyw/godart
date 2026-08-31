package main

import (
	"fmt"
	"time"
)

func main() {
	loads()
	newYork()
	sydney()
	fixed()
	parse()
}

// loads proves the name rules and the error of an unknown zone.
func loads() {
	for _, name := range []string{"", "UTC", "Local"} {
		loc, err := time.LoadLocation(name)
		fmt.Printf("load %q = %s %v\n", name, loc, err)
	}
	_, err := time.LoadLocation("Nope/Nowhere")
	fmt.Println("unknown:", err)
	_, err = time.LoadLocation("../etc/passwd")
	fmt.Println("dotdot:", err)
	_, err = time.LoadLocation("/etc/localtime")
	fmt.Println("rooted:", err)
}

// newYork proves the northern zone: the 2 halves of the year, the 2
// transitions of 2026, and an instant after the last transition of the file.
func newYork() {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	fmt.Println("name:", nyc)
	fmt.Println(time.Date(2026, time.January, 15, 12, 0, 0, 0, nyc))
	fmt.Println(time.Date(2026, time.July, 4, 12, 0, 0, 0, nyc))
	// The spring transition of 2026 removes the clock hour of 02:00. The
	// reading below stands inside the removed hour.
	fmt.Println(time.Date(2026, time.March, 8, 2, 30, 0, 0, nyc))
	// The fall transition repeats the clock hour of 01:00.
	fmt.Println(time.Date(2026, time.November, 1, 1, 30, 0, 0, nyc))
	// A year far after the last transition of the file reads the extend rule.
	fmt.Println(time.Date(2040, time.July, 1, 12, 0, 0, 0, nyc))
	fmt.Println(time.Date(2040, time.January, 1, 12, 0, 0, 0, nyc))
	// An instant maps into the zone.
	fmt.Println(time.Unix(1767225600, 0).In(nyc))
}

// sydney proves a southern zone. The DST period stands over the turn of the
// year, thus the 2 rules of the extend string swap.
func sydney() {
	syd, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	fmt.Println(time.Date(2026, time.January, 15, 12, 0, 0, 0, syd))
	fmt.Println(time.Date(2026, time.July, 4, 12, 0, 0, 0, syd))
	fmt.Println(time.Date(2041, time.January, 15, 12, 0, 0, 0, syd))
	fmt.Println(time.Unix(1767225600, 0).In(syd))
}

// fixed proves a zone of the Etc area. Such a file holds no transition.
func fixed() {
	gmt5, err := time.LoadLocation("Etc/GMT+5")
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	fmt.Println(gmt5, time.Date(2026, time.June, 1, 12, 0, 0, 0, gmt5))
	utc, err := time.LoadLocation("Etc/UTC")
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	fmt.Println(utc, time.Date(2026, time.June, 1, 12, 0, 0, 0, utc))
}

// parse proves ParseInLocation with a loaded zone.
func parse() {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", "2026-07-04 08:30:00", nyc)
	fmt.Println(t, err)
	u, err := time.ParseInLocation("2006-01-02 15:04:05 MST", "2026-01-15 08:30:00 EST", nyc)
	fmt.Println(u, err)
	fmt.Println(t.UTC())
}
