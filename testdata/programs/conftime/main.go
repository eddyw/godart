// The program proves the time rules of Go: the arithmetic of a Duration, the
// arithmetic of a Time, each layout of Format, Parse, and each comparison.
//
// The program never reads the clock. Each value comes from time.Date with the
// location UTC, thus 2 runs give the same output.
package main

import (
	"fmt"
	"time"
)

func row(label string, value any) {
	fmt.Printf("%-34s %v\n", label, value)
}

var base = time.Date(2023, time.March, 14, 15, 9, 26, 535897932, time.UTC)

func durations() {
	list := []time.Duration{
		0,
		time.Nanosecond,
		time.Microsecond,
		time.Millisecond,
		time.Second,
		time.Minute,
		time.Hour,
		90 * time.Second,
		3661 * time.Second,
		-90 * time.Second,
		time.Duration(1<<63 - 1),
		time.Duration(-1 << 63),
		1500 * time.Millisecond,
		2*time.Hour + 3*time.Minute + 4*time.Second,
		time.Duration(1),
		time.Duration(999),
		time.Duration(1001),
	}
	fmt.Printf("%-22s %-16s %-14s %-12s %-10s %s\n",
		"String", "Nanoseconds", "Microseconds", "Milliseconds", "Seconds", "Hours")
	for _, d := range list {
		fmt.Printf("%-22s %-16d %-14d %-12d %-10g %g\n",
			d, d.Nanoseconds(), d.Microseconds(), d.Milliseconds(),
			d.Seconds(), d.Hours())
	}
	row("Minutes of 90s", (90 * time.Second).Minutes())
	row("a sum", time.Hour+30*time.Minute)
	row("a difference", time.Hour-30*time.Minute)
	row("a product", 3*time.Minute)
	row("a quotient", time.Hour/time.Minute)
	row("a remainder", (90 * time.Second % time.Minute))
	row("Truncate", (90 * time.Second).Truncate(time.Minute))
	row("Round up", (95 * time.Second).Round(time.Minute))
	row("Round down", (85 * time.Second).Round(time.Minute))
	row("Round at the half", (90 * time.Second).Round(time.Minute))
	row("Abs of a negative", (-time.Hour).Abs())
	row("a comparison", time.Hour > time.Minute)
	// ParseDuration reads the text of String again.
	for _, text := range []string{"1h2m3s", "1.5h", "300ms", "-1.5h", "0",
		"1h", "2h45m", "1ns", "1us", "1µs", "1m30s", "bad", "", "1x"} {
		d, err := time.ParseDuration(text)
		fmt.Printf("ParseDuration %-10q %-16v err=%v\n", text, d, err)
	}
}

func arithmetic() {
	row("base", base)
	row("Add an hour", base.Add(time.Hour))
	row("Add a day", base.Add(24*time.Hour))
	row("Add a negative", base.Add(-time.Hour))
	row("AddDate 1 year", base.AddDate(1, 0, 0))
	row("AddDate 1 month", base.AddDate(0, 1, 0))
	row("AddDate 1 day", base.AddDate(0, 0, 1))
	row("AddDate over the end", time.Date(2023, time.January, 31, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0))
	row("a leap day", time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC))
	row("AddDate of a leap day", time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC).AddDate(1, 0, 0))
	other := base.Add(90 * time.Minute)
	row("Sub", other.Sub(base))
	row("Sub the other way", base.Sub(other))
	row("Sub of itself", base.Sub(base))
	row("Truncate to an hour", base.Truncate(time.Hour))
	row("Truncate to a minute", base.Truncate(time.Minute))
	row("Truncate to a second", base.Truncate(time.Second))
	row("Round to an hour", base.Round(time.Hour))
	row("Round to a minute", base.Round(time.Minute))
	row("the parts", fmt.Sprint(base.Year(), " ", base.Month(), " ", base.Day()))
	row("the clock", fmt.Sprint(base.Hour(), " ", base.Minute(), " ", base.Second()))
	row("Nanosecond", base.Nanosecond())
	row("Weekday", base.Weekday())
	row("YearDay", base.YearDay())
	y, m, d := base.Date()
	row("Date", fmt.Sprint(y, " ", m, " ", d))
	h, mi, s := base.Clock()
	row("Clock", fmt.Sprint(h, " ", mi, " ", s))
	row("ISOWeek", isoWeek(base))
	row("Unix", base.Unix())
	row("UnixMilli", base.UnixMilli())
	row("UnixMicro", base.UnixMicro())
	row("UnixNano", base.UnixNano())
	row("from Unix", time.Unix(base.Unix(), int64(base.Nanosecond())).UTC())
	row("from UnixMilli", time.UnixMilli(base.UnixMilli()).UTC())
	row("IsZero of base", base.IsZero())
	row("the Location", base.Location())
	row("UTC", base.UTC())
	row("the month as a number", int(base.Month()))
	row("each month", months())
	row("each weekday", weekdays())
}

func isoWeek(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprint(y, " ", w)
}

func months() []string {
	out := []string{}
	for m := time.January; m <= time.December; m++ {
		out = append(out, m.String())
	}
	return out
}

func weekdays() []string {
	out := []string{}
	for d := time.Sunday; d <= time.Saturday; d++ {
		out = append(out, d.String())
	}
	return out
}

func layouts() {
	list := []struct {
		name   string
		layout string
	}{
		{"Layout", time.Layout},
		{"ANSIC", time.ANSIC},
		{"UnixDate", time.UnixDate},
		{"RubyDate", time.RubyDate},
		{"RFC822", time.RFC822},
		{"RFC822Z", time.RFC822Z},
		{"RFC850", time.RFC850},
		{"RFC1123", time.RFC1123},
		{"RFC1123Z", time.RFC1123Z},
		{"RFC3339", time.RFC3339},
		{"RFC3339Nano", time.RFC3339Nano},
		{"Kitchen", time.Kitchen},
		{"Stamp", time.Stamp},
		{"StampMilli", time.StampMilli},
		{"StampMicro", time.StampMicro},
		{"StampNano", time.StampNano},
		{"DateTime", time.DateTime},
		{"DateOnly", time.DateOnly},
		{"TimeOnly", time.TimeOnly},
	}
	for _, it := range list {
		fmt.Printf("%-14s %s\n", it.name, base.Format(it.layout))
	}
	// A layout of each chunk.
	chunks := []string{
		"2006", "06", "January", "Jan", "01", "1",
		"Monday", "Mon", "02", "2", "_2", "002",
		"15", "03", "3", "04", "4", "05", "5",
		"PM", "pm", "MST", "Z07:00", "-07:00", "-0700", "-07",
		".000", ".000000", ".000000000", ".0", ".00",
		"2006-01-02T15:04:05.999999999Z07:00",
		"Mon Jan _2 15:04:05 2006",
		"15:04:05.000",
	}
	for _, c := range chunks {
		fmt.Printf("%-36q %s\n", c, base.Format(c))
	}
	// A time with no fraction writes no fraction with a 9 form.
	plain := time.Date(2023, time.March, 14, 15, 9, 26, 0, time.UTC)
	row("no fraction with .999", plain.Format("15:04:05.999"))
	row("no fraction with .000", plain.Format("15:04:05.000"))
	row("String of it", plain.String())
	// A time of a fixed zone.
	zone := time.FixedZone("XYZ", 5*3600+1800)
	inZone := base.In(zone)
	row("in a fixed zone", inZone)
	row("RFC3339 of it", inZone.Format(time.RFC3339))
	row("the zone name", zoneOf(inZone))
	row("the zone of UTC", zoneOf(base))
	// A negative zone.
	west := time.FixedZone("WST", -8*3600)
	row("a west zone", base.In(west).Format(time.RFC1123Z))
}

func zoneOf(t time.Time) string {
	name, offset := t.Zone()
	return fmt.Sprint(name, " ", offset)
}

func parses() {
	inputs := []struct {
		layout string
		text   string
	}{
		{time.RFC3339, "2023-03-14T15:09:26Z"},
		{time.RFC3339, "2023-03-14T15:09:26+02:00"},
		{time.RFC3339Nano, "2023-03-14T15:09:26.535897932Z"},
		{time.DateOnly, "2023-03-14"},
		{time.DateTime, "2023-03-14 15:09:26"},
		{time.ANSIC, "Tue Mar 14 15:09:26 2023"},
		{time.RFC1123, "Tue, 14 Mar 2023 15:09:26 UTC"},
		{"2006-01-02", "2023-13-01"},
		{"2006-01-02", "2023-02-30"},
		{"2006-01-02", "not a date"},
		{time.RFC3339, "2023-03-14T15:09:26"},
	}
	for _, in := range inputs {
		t, err := time.Parse(in.layout, in.text)
		if err != nil {
			fmt.Printf("Parse %-28q err=%v\n", in.text, err)
			continue
		}
		fmt.Printf("Parse %-28q -> %v\n", in.text, t.UTC())
	}
	// A layout that names no year gives the year 0. A Time of the overlay
	// holds 1 count of nanoseconds, thus the range is the year 1678 to the
	// year 2262 and the whole value wraps. Read the section "The clock and
	// the timer" of docs/parity.md, and REPORT-conform.md, item 14. The
	// program gives each such layout a date, thus the result stays inside
	// the range.
	for _, in := range []struct{ layout, text string }{
		{"2006-01-02 15:04:05", "2023-03-14 15:09:26"},
		{"2006-01-02 3:04PM", "2023-03-14 3:09PM"},
		{"2006-01-02 " + time.TimeOnly, "2023-03-14 01:02:03"},
	} {
		t, err := time.Parse(in.layout, in.text)
		fmt.Printf("Parse %-24q -> %s err=%v\n", in.text, t.Format("15:04:05"), err)
	}
	// ParseInLocation reads a text with no zone in a location.
	zone := time.FixedZone("XYZ", 3600)
	t, err := time.ParseInLocation(time.DateTime, "2023-03-14 15:09:26", zone)
	fmt.Printf("%-34s %v err=%v\n", "ParseInLocation", t, err)
	fmt.Printf("%-34s %v\n", "the same instant in UTC", t.UTC())
	// A round trip of Format and Parse.
	text := base.Format(time.RFC3339Nano)
	back, err := time.Parse(time.RFC3339Nano, text)
	fmt.Printf("%-34s %v err=%v\n", "a round trip", back.Equal(base), err)
	// The text of a ParseError.
	_, err = time.Parse(time.RFC3339, "bad")
	row("the error type", fmt.Sprintf("%T", err))
}

func compares() {
	early := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
	row("Before", early.Before(late))
	row("After", early.After(late))
	row("Equal", early.Equal(early))
	row("Equal of 2 zones", early.Equal(early.In(time.FixedZone("X", 3600))))
	row("== of 2 zones", early == early.In(time.FixedZone("X", 3600)))
	row("== of 1 value", early == early)
	row("Compare early late", early.Compare(late))
	row("Compare late early", late.Compare(early))
	row("Compare of itself", early.Compare(early))
	// A Time is a key of a map.
	m := map[time.Time]string{early: "early"}
	row("as a map key", m[early])
	row("as a map gap", fmt.Sprintf("%q", m[late]))
	// A sort of a list of times.
	list := []time.Time{late, early, base}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Before(list[i]) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	texts := []string{}
	for _, t := range list {
		texts = append(texts, t.Format(time.DateOnly))
	}
	row("the sorted order", texts)
	// The text forms.
	row("MarshalText", mustText(early))
	row("MarshalJSON", mustJSON(early))
	var back time.Time
	err := back.UnmarshalText([]byte("2020-01-01T00:00:00Z"))
	row("UnmarshalText", fmt.Sprint(back.Equal(early), " err=", err))
	bin, err := early.MarshalBinary()
	row("MarshalBinary", fmt.Sprint(len(bin), " err=", err))
	var fromBin time.Time
	err = fromBin.UnmarshalBinary(bin)
	row("UnmarshalBinary", fmt.Sprint(fromBin.Equal(early), " err=", err))
}

func mustText(t time.Time) string {
	b, err := t.MarshalText()
	if err != nil {
		return "err " + err.Error()
	}
	return string(b)
}

func mustJSON(t time.Time) string {
	b, err := t.MarshalJSON()
	if err != nil {
		return "err " + err.Error()
	}
	return string(b)
}

func main() {
	fmt.Println("== a duration ==")
	durations()
	fmt.Println("== the arithmetic ==")
	arithmetic()
	fmt.Println("== the layouts ==")
	layouts()
	fmt.Println("== the parse ==")
	parses()
	fmt.Println("== the comparison ==")
	compares()
}
