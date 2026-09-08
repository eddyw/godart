// Command sqlzerotime proves that a Time counts from January 1 of the year 1.
//
// The real test suite of database/sql found the gap. convertAssign writes a
// Time into a string with the layout RFC 3339, and the zero value of the
// overlay wrote the start of the Unix epoch.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	var zero time.Time
	fmt.Println("zero", zero.Format(time.RFC3339Nano))
	fmt.Println("zero string", zero.String())
	fmt.Println("zero is zero", zero.IsZero())
	fmt.Println("zero year", zero.Year(), zero.Month(), zero.Day())
	fmt.Println("zero clock", zero.Hour(), zero.Minute(), zero.Second())
	fmt.Println("zero weekday", zero.Weekday(), "yearday", zero.YearDay())
	fmt.Println("zero unix", zero.Unix())
	fmt.Println("zero unix milli", zero.UnixMilli())
	fmt.Println("zero location", zero.Location())
	fmt.Println("epoch is zero", time.Unix(0, 0).IsZero())

	// Each order of the zero value against a real instant.
	at := time.Date(2024, time.March, 1, 12, 30, 45, 123456789, time.UTC)
	fmt.Println("before", zero.Before(at), "after", zero.After(at))
	fmt.Println("compare", zero.Compare(at), at.Compare(zero), zero.Compare(zero))
	fmt.Println("equal", zero.Equal(time.Time{}))

	// A date of the year 1 and each date near it.
	for _, d := range []time.Time{
		time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1, time.January, 1, 0, 0, 0, 1, time.UTC),
		time.Date(1, time.January, 2, 3, 4, 5, 0, time.UTC),
		time.Date(0, time.December, 31, 23, 59, 59, 999999999, time.UTC),
		time.Date(1582, time.October, 15, 0, 0, 0, 0, time.UTC),
		time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC),
	} {
		fmt.Println("date", d.Format("2006-01-02 15:04:05.999999999"),
			d.Weekday(), d.YearDay(), d.Unix())
	}

	// The count of nanoseconds of the range of an int64 still reads.
	old := time.Date(1678, time.January, 1, 0, 0, 0, 0, time.UTC)
	fmt.Println("old unix nano", old.UnixNano())
	fmt.Println("new unix nano", time.Unix(1700000000, 500).UnixNano())

	// Add and Sub against the zero value.
	fmt.Println("add", zero.Add(90*time.Minute).Format(time.RFC3339Nano))
	fmt.Println("add negative", zero.Add(-time.Nanosecond).Format(time.RFC3339Nano))
	fmt.Println("sub", at.Sub(zero))
	fmt.Println("sub reverse", zero.Sub(at))
	fmt.Println("since epoch", time.Unix(0, 0).Sub(zero))

	// Truncate and Round read January 1 of the year 1 as the origin.
	fmt.Println("truncate hour", at.Truncate(time.Hour).Format(time.RFC3339Nano))
	fmt.Println("round hour", at.Round(time.Hour).Format(time.RFC3339Nano))
	fmt.Println("truncate day", at.Truncate(24*time.Hour).Format(time.RFC3339Nano))
	fmt.Println("zero truncate", zero.Truncate(time.Hour).Format(time.RFC3339Nano))
	fmt.Println("year 1 round",
		time.Date(1, time.January, 1, 0, 40, 0, 0, time.UTC).
			Round(time.Hour).Format(time.RFC3339Nano))

	// The text forms.
	text, err := zero.MarshalText()
	fmt.Println("text", string(text), err)
	data, err := json.Marshal(zero)
	fmt.Println("json", string(data), err)
	var back time.Time
	fmt.Println("json back", json.Unmarshal(data, &back), back.Equal(zero))
	bin, err := zero.MarshalBinary()
	fmt.Println("binary", bin, err)
	var round time.Time
	fmt.Println("binary back", round.UnmarshalBinary(bin), round.Equal(zero))

	// A year outside the range 0 to 9999 gives an error, exactly as in Go.
	far := time.Date(12345, time.January, 1, 0, 0, 0, 0, time.UTC)
	_, err = far.MarshalText()
	fmt.Println("far text err", err)
	_, err = far.MarshalJSON()
	fmt.Println("far json err", err)

	// Parse gives the zero value back.
	parsed, err := time.Parse(time.RFC3339, "0001-01-01T00:00:00Z")
	fmt.Println("parse", parsed.Equal(zero), err)

	// A layout that names no year gives the year 0, and that year reads.
	noYear, err := time.Parse("15:04", "10:30")
	fmt.Println("no year", noYear.Format(time.RFC3339Nano), err)
	noYear2, err := time.Parse("Jan 2", "Mar 5")
	fmt.Println("no year 2", noYear2.Format(time.RFC3339Nano), err)

	// Cookie.Valid reads the bound of the year 1601 of RFC 6265.
	c := &http.Cookie{Name: "a", Value: "b",
		Expires: time.Date(1500, time.January, 1, 0, 0, 0, 0, time.UTC)}
	fmt.Println("cookie 1500", c.Valid())
	c.Expires = time.Date(1700, time.January, 1, 0, 0, 0, 0, time.UTC)
	fmt.Println("cookie 1700", c.Valid())
	c.Expires = time.Time{}
	fmt.Println("cookie zero", c.Valid())

	// The zero value of a struct that holds a Time.
	type row struct {
		Name string
		At   time.Time
	}
	var r row
	fmt.Println("struct", r.At.Format(time.RFC3339Nano), r.At.IsZero())
	out, err := json.Marshal(r)
	fmt.Println("struct json", string(out), err)
}
