package main

import (
	"fmt"
	"time"
	_ "time/tzdata"
)

// The import of time/tzdata embeds the IANA database into the program. The
// zones below load on a system with no zone files, and the output stays equal
// to the output of a load from the system.
func main() {
	for _, name := range []string{"America/Chicago", "Europe/Berlin", "Asia/Tokyo"} {
		loc, err := time.LoadLocation(name)
		if err != nil {
			fmt.Println("load:", err)
			continue
		}
		fmt.Println(loc, time.Date(2026, time.February, 1, 12, 0, 0, 0, loc))
		fmt.Println(loc, time.Date(2026, time.August, 1, 12, 0, 0, 0, loc))
	}
	_, err := time.LoadLocation("Nope/Nowhere")
	fmt.Println("unknown:", err)
}
