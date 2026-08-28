package main

import (
	"fmt"
	"time"
)

func main() {
	for _, d := range []time.Duration{
		0,
		1,
		999,
		1500 * time.Nanosecond,
		time.Microsecond,
		1500 * time.Microsecond,
		time.Millisecond,
		1500 * time.Millisecond,
		time.Second,
		90 * time.Second,
		time.Hour + 30*time.Minute,
		25*time.Hour + 61*time.Second + 500*time.Millisecond,
		-1500 * time.Millisecond,
		-2 * time.Hour,
	} {
		fmt.Println(d.String())
	}

	// A date before 1901 reads a historical zone. Go and Dart give 2 different
	// offsets for such an instant, and docs/parity.md holds that gap.
	for _, sec := range []int64{
		0,
		1,
		951782400,
		1451606400,
		1772323200,
		-86400,
		-1,
	} {
		fmt.Println(time.Unix(sec, 0).String())
	}

	fmt.Println(time.Unix(1451606400, 1).String())
	fmt.Println(time.Unix(1451606400, 100000000).String())
	fmt.Println(time.Unix(1451606400, 123456789).String())
	fmt.Println(time.Unix(1451606400, 999999999).String())
}
