// Command rtmarshaler proves that a marshal of a type that declares
// MarshalJSON or MarshalText writes the text of that method.
//
// encoding/json asks reflect for the pointer to the type first, thus an
// addressable value takes the method of the pointer. The compiler writes no
// description of a pointer whose method set is the method set of the value,
// thus reflect builds that description at run time. The method table of it
// must hold the Dart type of each result.
package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Stamp declares MarshalJSON with a value receiver.
type Stamp struct {
	Sec int
}

func (s Stamp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", fmt.Sprintf("sec-%d", s.Sec))), nil
}

// Tag declares MarshalText with a value receiver.
type Tag struct {
	Name string
}

func (t Tag) MarshalText() ([]byte, error) {
	return []byte("tag:" + t.Name), nil
}

// Count declares MarshalJSON with a pointer receiver.
type Count struct {
	N int
}

func (c *Count) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", c.N*2)), nil
}

// Num is a whole number that declares MarshalJSON with a value receiver. The
// Dart form of a pointer to it is a cell, thus the method of the value cannot
// stand in the table of the pointer with no change.
type Num int

func (n Num) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", int(n)+100)), nil
}

// Word is a text that declares MarshalText with a value receiver.
type Word string

func (w Word) MarshalText() ([]byte, error) {
	return []byte("word:" + string(w)), nil
}

// Pair is a slice that declares MarshalJSON with a value receiver.
type Pair []int

func (p Pair) MarshalJSON() ([]byte, error) {
	if len(p) != 2 {
		return nil, fmt.Errorf("pair of %d", len(p))
	}
	return []byte(fmt.Sprintf("\"%d:%d\"", p[0], p[1])), nil
}

// Wrap holds each shape in 1 struct.
type Wrap struct {
	Stamp Stamp
	Tag   Tag
	Count Count
	List  []Stamp
	Dict  map[string]Tag
	Ptr   *Stamp
	Time  time.Time
}

// Holder holds each named base type. A marshal of a pointer to it makes each
// field addressable, thus each field takes the method of the pointer.
type Holder struct {
	Num  Num
	Word Word
	Pair Pair
}

func show(name string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Println(name, "error", err)
		return
	}
	fmt.Println(name, string(b))
}

func main() {
	show("stamp", Stamp{Sec: 7})
	show("stampptr", &Stamp{Sec: 8})
	show("tag", Tag{Name: "red"})
	show("tagptr", &Tag{Name: "blue"})
	show("count", &Count{N: 21})
	show("slice", []Stamp{{Sec: 1}, {Sec: 2}})
	show("map", map[string]Tag{"a": {Name: "one"}})
	show("wrap", Wrap{
		Stamp: Stamp{Sec: 3},
		Tag:   Tag{Name: "green"},
		Count: Count{N: 5},
		List:  []Stamp{{Sec: 4}},
		Dict:  map[string]Tag{"k": {Name: "v"}},
		Ptr:   &Stamp{Sec: 9},
		Time:  time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC),
	})
	show("wrapptr", &Wrap{
		Stamp: Stamp{Sec: 13},
		Tag:   Tag{Name: "olive"},
		Count: Count{N: 6},
		List:  []Stamp{{Sec: 14}},
		Dict:  map[string]Tag{"k": {Name: "w"}},
		Ptr:   &Stamp{Sec: 19},
		Time:  time.Date(2026, 9, 6, 13, 0, 0, 0, time.UTC),
	})

	// A pointer to the holder makes each field addressable. encoding/json then
	// reads the method of the pointer to the field.
	show("holder", Holder{Num: 1, Word: "a", Pair: Pair{1, 2}})
	show("holderptr", &Holder{Num: 2, Word: "b", Pair: Pair{3, 4}})

	// A value of a named base type, alone and behind a pointer.
	n := Num(5)
	show("num", n)
	show("numptr", &n)
	w := Word("c")
	show("word", w)
	show("wordptr", &w)
	p := Pair{7, 8}
	show("pair", p)
	show("pairptr", &p)

	// A slice and an array of such a value each hold an addressable element.
	show("numslice", []Num{1, 2})
	show("numarray", [2]Num{3, 4})
	show("numsliceptr", &[]Num{5, 6})

	// A key of a map takes MarshalText.
	show("mapkey", map[Tag]int{{Name: "k"}: 1})
	show("wordkey", map[Word]int{"z": 1})

	// An interface value that holds the type also takes the method.
	var any1 any = Stamp{Sec: 11}
	show("iface", any1)

	// A method that gives an error reaches the caller.
	show("pairbad", &Holder{Num: 9, Word: "d", Pair: Pair{1}})

	// Indent reads the same path.
	b, err := json.MarshalIndent([]Tag{{Name: "x"}}, "", " ")
	fmt.Println("indent", string(b), err)
}
