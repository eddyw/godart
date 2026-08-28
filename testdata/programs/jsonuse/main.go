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
}
