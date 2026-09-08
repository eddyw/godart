// Command jsonencode proves the fast marshal of encoding/json against the Go
// code of the package. Each case writes the bytes or the error, thus the run
// of go and the run of the Dart output must agree line by line.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// activity holds the shape of the benchmark document: a struct of plain
// fields with a json tag on each 1.
type activity struct {
	Name     string  `json:"name"`
	Duration float64 `json:"duration"`
	Family   bool    `json:"familyFriendly"`
	Ref      string  `json:"ref"`
}

// mixed holds 1 field of each kind that the fast path serves.
type mixed struct {
	B    bool              `json:"b"`
	S    string            `json:"s"`
	I    int               `json:"i"`
	I8   int8              `json:"i8"`
	U64  uint64            `json:"u64"`
	F    float64           `json:"f"`
	A    any               `json:"a"`
	L    []int             `json:"l"`
	M    map[string]string `json:"m"`
	P    *int              `json:"p"`
	Arr  [3]int            `json:"arr"`
	Byte []byte            `json:"byte"`
	Skip string            `json:"-"`
	Off  string            `json:"off,omitempty"`
	On   string            `json:"on,omitempty"`
	Low  string
}

// withMethod holds a marshal method, thus the fast path hands it back.
type withMethod struct{ N int }

func (w withMethod) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", "n="+fmt.Sprint(w.N))), nil
}

// stamp holds a field of a type with a method.
type stamp struct {
	W withMethod `json:"w"`
	K string     `json:"k"`
}

func show(name string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("%s: error %v\n", name, err)
		return
	}
	fmt.Printf("%s: %s\n", name, b)
}

func main() {
	show("nil", nil)
	show("bool", true)
	show("int", -42)
	show("uint", uint64(18446744073709551615))
	show("string", "plain")
	show("escape", "a\"b\\c\nd\te<f>g&h i j\x01k")
	show("badutf8", string([]byte{0x61, 0xff, 0x62}))
	show("floats", []float64{0, -0.0, 0.5, 3, 1e-6, 1e-7, 1e20, 1e21, 1.5e-9,
		math.MaxFloat64, math.SmallestNonzeroFloat64, 1234567890123456789})
	show("nilslice", []int(nil))
	show("emptyslice", []int{})
	show("nilmap", map[string]int(nil))
	show("emptymap", map[string]int{})
	show("sorted", map[string]int{"b": 2, "a": 1, "C": 3, "": 0, "ab": 4})
	show("bytes", []byte{1, 2, 3, 4, 5})
	show("emptybytes", []byte{})
	show("bytearray", [4]byte{1, 2, 3, 4})
	show("nested", []any{nil, true, "s", 1.5, []any{1.0, 2.0},
		map[string]any{"k": "v"}})

	n := 7
	show("mixed", mixed{
		B: true, S: "s", I: -1, I8: -8, U64: 9, F: 2.5, A: "any",
		L: []int{1, 2}, M: map[string]string{"z": "1", "a": "2"}, P: &n,
		Arr: [3]int{1, 2, 3}, Byte: []byte("hi"), Skip: "gone", On: "here",
		Low: "low",
	})
	show("zeromixed", mixed{})
	show("method", withMethod{N: 3})
	show("methodfield", stamp{W: withMethod{N: 4}, K: "k"})
	show("ptrstruct", &activity{Name: "a", Duration: 1, Family: true, Ref: "r"})
	show("slicestruct", []activity{{Name: "a"}, {Name: "b", Duration: 2}})
	show("nan", math.NaN())
	show("inf", math.Inf(1))
	show("number", json.Number("1.5"))
	show("rawmessage", json.RawMessage(`{"raw": 1}`))

	// A tree of 300 levels stands above the depth that the fast path serves,
	// thus the Go code marshals it.
	var deep any = "leaf"
	for i := 0; i < 300; i++ {
		deep = []any{deep}
	}
	b, err := json.Marshal(deep)
	fmt.Printf("deep: %d %v %v\n", len(b), err, strings.HasPrefix(string(b), "[[["))

	indented, err := json.MarshalIndent(mixed{S: "s", L: []int{1}}, "", "\t")
	fmt.Printf("indent: %v\n%s\n", err, indented)
}
