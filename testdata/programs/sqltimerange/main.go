// Command sqltimerange proves the whole range of a time.Time.
//
// A Time counts seconds from January 1 of the year 1 since this session, thus
// each year of Go reads. The witness walks 8 instants from the year 1 to the
// year 3000, and it reads each method that the new shape touched: the
// calendar, the zone, the round family, the Unix family, and the binary form.
package main

import (
	"fmt"
	"time"
	_ "time/tzdata"
)

func show(tag string, t time.Time) {
	year, week := t.ISOWeek()
	fmt.Printf("%s %s | %d-%02d | %v | %d %d %d\n", tag,
		t.Format("2006-01-02T15:04:05.999999999Z07:00 MST"), year, week,
		t.Weekday(), t.Unix(), t.UnixMilli(), t.UnixMicro())
}

func main() {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Println("load", err)
		return
	}
	dates := []time.Time{
		time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(500, time.July, 4, 13, 45, 30, 250000000, time.UTC),
		time.Date(1600, time.February, 29, 23, 59, 59, 999999999, time.UTC),
		time.Date(1900, time.March, 1, 0, 0, 0, 1, ny),
		time.Date(1969, time.December, 31, 23, 59, 59, 999999999, time.UTC),
		time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, time.November, 3, 1, 30, 0, 0, ny),
		time.Date(3000, time.June, 15, 6, 0, 0, 0, ny),
	}
	for i, d := range dates {
		show(fmt.Sprint("d", i), d)
		show(fmt.Sprint("  trunc-h", i), d.Truncate(time.Hour))
		show(fmt.Sprint("  round-h", i), d.Round(time.Hour))
		show(fmt.Sprint("  trunc-d", i), d.Truncate(24*time.Hour))
		show(fmt.Sprint("  round-m", i), d.Round(time.Minute))
		show(fmt.Sprint("  trunc-ms", i), d.Truncate(time.Millisecond))
		show(fmt.Sprint("  addyear", i), d.AddDate(1, 1, 1))
		from, to := d.ZoneBounds()
		fmt.Println("  bounds", i, from.Format(time.RFC3339),
			to.Format(time.RFC3339), d.IsDST())
		name, offset := d.Zone()
		fmt.Println("  zone", i, name, offset)
		fmt.Println("  milli back", i,
			time.UnixMilli(d.UnixMilli()).UTC().Format(time.RFC3339Nano),
			time.UnixMicro(d.UnixMicro()).UTC().Format(time.RFC3339Nano))
		bin, err := d.MarshalBinary()
		var back time.Time
		err2 := back.UnmarshalBinary(bin)
		fmt.Println("  binary", i, err, err2, back.Equal(d), back.Location())
	}

	// The distance of the 2 ends of the range leaves the range of a Duration,
	// thus Sub gives the bound of Go.
	low := dates[0]
	high := dates[len(dates)-1]
	fmt.Println("sub", high.Sub(low), low.Sub(high))
	fmt.Println("compare", low.Compare(high), high.Compare(low))
	fmt.Println("add max", low.Add(1<<62).Format(time.RFC3339Nano))

	// A negative duration on a date before the epoch, and the round family on a
	// date before the year 1.
	fmt.Println("neg", dates[4].Add(-3*time.Hour-1).Format(time.RFC3339Nano))
	fmt.Println("neg round", dates[4].Round(time.Hour).Format(time.RFC3339Nano))
	fmt.Println("neg trunc",
		dates[4].Truncate(7*time.Minute).Format(time.RFC3339Nano))
	fmt.Println("year1 trunc",
		low.Add(-time.Second).Truncate(time.Hour).Format(time.RFC3339Nano))
	fmt.Println("year1 round",
		low.Add(-31*time.Minute).Round(time.Hour).Format(time.RFC3339Nano))
	fmt.Println("year1 trunc7",
		low.Add(-time.Second).Truncate(7*time.Minute).Format(time.RFC3339Nano))
}
