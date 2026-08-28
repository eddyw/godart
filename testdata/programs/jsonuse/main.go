package main

import (
	"encoding/json"
	"fmt"
)

type address struct {
	City string `json:"city"`
	Zip  string `json:"zip,omitempty"`
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
}
