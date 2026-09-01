package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type address struct {
	City string `json:"city"`
	Zip  string `json:"zip,omitempty"`
}

// point writes its own JSON form through a value receiver, and it reads one
// through a pointer receiver.
type point struct {
	X int
	Y int
}

func (p point) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%d,%d]", p.X, p.Y)), nil
}

func (p *point) UnmarshalJSON(data []byte) error {
	text := strings.Trim(string(data), "[]")
	parts := strings.Split(text, ",")
	if len(parts) != 2 {
		return errors.New("point: want 2 numbers")
	}
	x, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	y, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}
	p.X, p.Y = x, y
	return nil
}

// level is a named string type with a pointer receiver, thus the decoder needs
// a box for it.
type level string

func (l level) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strings.ToUpper(string(l)) + `"`), nil
}

func (l *level) UnmarshalJSON(data []byte) error {
	*l = level(strings.ToLower(strings.Trim(string(data), `"`)))
	return nil
}

// noForm reports a cause from its marshal method.
type noForm struct{}

func (n noForm) MarshalJSON() ([]byte, error) {
	return nil, errors.New("no form")
}

type record struct {
	At    time.Time `json:"at"`
	Blob  []byte    `json:"blob"`
	Where point     `json:"where"`
	Level level     `json:"level"`
	Empty []byte    `json:"empty"`
}

type user struct {
	Name    string            `json:"name"`
	Age     int               `json:"age"`
	Score   float64           `json:"score"`
	Active  bool              `json:"active"`
	Tags    []string          `json:"tags"`
	Home    address           `json:"home"`
	Boss    *user             `json:"boss"`
	Meta    map[string]string `json:"meta"`
	Hidden  string            `json:"-"`
	skipped string
	NoTag   int
	Empty   string `json:"empty,omitempty"`
}

func main() {
	u := user{
		Name:   "ada",
		Age:    36,
		Score:  9.5,
		Active: true,
		Tags:   []string{"x", "y"},
		Home:   address{City: "london"},
		Meta:   map[string]string{"b": "2", "a": "1"},
		Hidden: "secret",
		NoTag:  7,
	}
	b, err := json.Marshal(u)
	fmt.Println(string(b))
	fmt.Println(err == nil)

	boss := user{Name: "boss", Tags: []string{}, Meta: map[string]string{}}
	u.Boss = &boss
	b, _ = json.Marshal(u.Boss)
	fmt.Println(string(b))

	b, _ = json.Marshal([]int{1, 2, 3})
	fmt.Println(string(b))
	b, _ = json.Marshal(map[string]int{"b": 2, "a": 1})
	fmt.Println(string(b))
	b, _ = json.Marshal("plain")
	fmt.Println(string(b))
	b, _ = json.Marshal(42)
	fmt.Println(string(b))
	b, _ = json.Marshal(nil)
	fmt.Println(string(b))
	b, _ = json.Marshal([]string(nil))
	fmt.Println(string(b))

	b, _ = json.MarshalIndent(address{City: "rome", Zip: "1"}, "", "  ")
	fmt.Println(string(b))

	var back user
	err = json.Unmarshal([]byte(`{"name":"ann","tags":["x","y"],"home":{"city":"oslo","zip":"7"},"meta":{"k":"v"},"NoTag":3}`), &back)
	fmt.Println(back.Name, len(back.Tags), back.Tags[1], back.Home.City, back.Meta["k"], back.NoTag, err == nil)

	var pointer *address
	err = json.Unmarshal([]byte(`{"city":"kiel"}`), &pointer)
	fmt.Println(pointer.City, err == nil)

	var numbers []int
	err = json.Unmarshal([]byte(`[3,1,2]`), &numbers)
	fmt.Println(len(numbers), numbers[0], err == nil)

	var table map[string]int
	err = json.Unmarshal([]byte(`{"a":1,"b":2}`), &table)
	fmt.Println(len(table), table["b"], err == nil)

	var loose any
	err = json.Unmarshal([]byte(`{"n":1.5,"s":"t","b":false,"l":[1,null]}`), &loose)
	fmt.Println(loose, err == nil)

	var broken address
	err = json.Unmarshal([]byte(`{`), &broken)
	fmt.Println(err != nil)

	var mismatch int
	err = json.Unmarshal([]byte(`"text"`), &mismatch)
	fmt.Println(err != nil)

	r := record{
		At:    time.Date(2023, time.November, 14, 22, 13, 20, 0, time.UTC),
		Blob:  []byte{1, 2, 250},
		Where: point{X: 3, Y: 4},
		Level: "warn",
	}
	b, err = json.Marshal(r)
	fmt.Println(string(b), err == nil)

	b, _ = json.Marshal(&r.Where)
	fmt.Println(string(b))
	b, _ = json.Marshal([]byte("hi"))
	fmt.Println(string(b))
	b, _ = json.Marshal(map[string]point{"a": {X: 1, Y: 2}})
	fmt.Println(string(b))
	b, _ = json.MarshalIndent(r.Where, "", "  ")
	fmt.Println(string(b))

	_, err = json.Marshal(noForm{})
	fmt.Println(err != nil)

	var read record
	err = json.Unmarshal([]byte(`{"at":"2024-02-29T01:02:03.5Z","blob":"AQL6","where":[7,8],"level":"LOUD","empty":null}`), &read)
	fmt.Println(read.At.UTC().Format(time.RFC3339Nano), err == nil)
	fmt.Println(read.Blob, read.Where.X, read.Where.Y, read.Level, read.Empty == nil)

	var one point
	err = json.Unmarshal([]byte(`[9,10]`), &one)
	fmt.Println(one.X, one.Y, err == nil)

	var stamp time.Time
	err = json.Unmarshal([]byte(`"2020-01-02T03:04:05Z"`), &stamp)
	fmt.Println(stamp.Unix(), err == nil)

	var bad point
	err = json.Unmarshal([]byte(`[1]`), &bad)
	fmt.Println(err != nil)

	var stamps []time.Time
	err = json.Unmarshal([]byte(`["2020-01-02T03:04:05Z","2021-01-02T03:04:05Z"]`), &stamps)
	fmt.Println(len(stamps), stamps[1].Unix(), err == nil)

	text, err := r.At.MarshalText()
	fmt.Println(string(text), err == nil)
	var again time.Time
	fmt.Println(again.UnmarshalText(text) == nil, again.Equal(r.At))

	raw, err := r.At.MarshalBinary()
	fmt.Println(raw, err == nil)
	zoned, _ := r.At.In(time.FixedZone("odd", 3661)).MarshalBinary()
	fmt.Println(zoned)
	var third time.Time
	fmt.Println(third.UnmarshalBinary(raw) == nil, third.Equal(r.At))
	fmt.Println(third.UnmarshalBinary(nil) != nil)
}
