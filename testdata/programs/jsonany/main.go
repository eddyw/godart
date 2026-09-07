// Command jsonany proves the fast decoder of encoding/json against the Go
// code of the package. Each case writes the value or the error, thus the run
// of go and the run of the Dart output must agree line by line.
package main

import (
	"encoding/json"
	"fmt"
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
	B   bool           `json:"b"`
	S   string         `json:"s"`
	I   int            `json:"i"`
	I8  int8           `json:"i8"`
	U64 uint64         `json:"u64"`
	F   float64        `json:"f"`
	A   any            `json:"a"`
	L   []int          `json:"l"`
	M   map[string]int `json:"m"`
	Arr [3]int         `json:"arr"`
	P   *int           `json:"p"`
	Sub activity       `json:"sub"`
	Off int            `json:"-"`
	Low int            `json:"low,omitempty"`
	Bad int
}

// counter reads the bytes of its value itself.
type counter struct {
	N int
}

func (c *counter) UnmarshalJSON(b []byte) error {
	c.N = len(b)
	return nil
}

func main() {
	anyCases()
	sliceCases()
	mapCases()
	typedCases()
	errorCases()
	depthCases()
	validCases()
}

// show writes the value of a decode into any, or the error of it.
func show(label, in string) {
	var v any
	if err := json.Unmarshal([]byte(in), &v); err != nil {
		fmt.Printf("%s: %T %v\n", label, err, err)
		return
	}
	fmt.Printf("%s: %#v\n", label, v)
}

func anyCases() {
	fmt.Println("== any ==")
	show("null", `null`)
	show("true", `true`)
	show("false", `false`)
	show("zero", `0`)
	show("negzero", `-0`)
	show("int", `42`)
	show("big53", `9007199254740993`)
	show("big64", `18446744073709551615`)
	show("above64", `184467440737095516150`)
	show("fraction", `1.5`)
	show("exponent", `1e3`)
	show("expminus", `1.25e-3`)
	show("expplus", `2E+2`)
	show("long", `0.30000000000000004`)
	show("tiny", `5e-324`)
	show("string", `"text"`)
	show("escapes", `"a\"b\\c\/d\be\ff\ng\rh\ti"`)
	show("unicode", `"éA"`)
	show("surrogate", `"😀"`)
	show("lonehigh", `"\ud800"`)
	show("lonelow", `"\udc00"`)
	show("badpair", `"\ud800A"`)
	show("empty", `""`)
	show("object", `{"a":1,"b":"two"}`)
	show("emptyobject", `{}`)
	show("duplicate", `{"a":1,"a":2}`)
	show("nested", `{"a":{"b":[1,2,{"c":null}]}}`)
	show("array", `[1,"two",true,null,[],{}]`)
	show("emptyarray", `[]`)
	show("spaces", "\t\r\n {\n \"a\" \t: \r [ 1 , 2 ] \n} \n")
	// Go replaces each byte of an invalid UTF-8 sequence with U+FFFD.
	show("badutf8", "\"a\xffb\"")
	show("shortutf8", "\"a\xe2\x82\"")
	show("goodutf8", "\"a\xe2\x82\xacb\"")
}

func sliceCases() {
	fmt.Println("== slice of any ==")
	var v []any
	err := json.Unmarshal([]byte(`[1,{"a":[true]},null]`), &v)
	fmt.Printf("slice: %v %#v\n", err, v)
	var e []any
	err = json.Unmarshal([]byte(`[]`), &e)
	fmt.Printf("empty: %v %#v len=%d\n", err, e, len(e))
	var n []any
	err = json.Unmarshal([]byte(`null`), &n)
	fmt.Printf("null: %v %#v nil=%t\n", err, n, n == nil)
	held := []any{"old", "kept"}
	err = json.Unmarshal([]byte(`[1]`), &held)
	fmt.Printf("held: %v %#v\n", err, held)
}

func mapCases() {
	fmt.Println("== map of any ==")
	var m map[string]any
	err := json.Unmarshal([]byte(`{"b":2,"a":1,"c":{"d":[3]}}`), &m)
	fmt.Printf("map: %v %v\n", err, m)
	var e map[string]any
	err = json.Unmarshal([]byte(`{}`), &e)
	fmt.Printf("empty: %v %v nil=%t\n", err, e, e == nil)
	var n map[string]any
	err = json.Unmarshal([]byte(`null`), &n)
	fmt.Printf("null: %v %v nil=%t\n", err, n, n == nil)
	held := map[string]any{"old": 1}
	err = json.Unmarshal([]byte(`{"new":2}`), &held)
	fmt.Printf("held: %v %v\n", err, held)
}

func typedCases() {
	fmt.Println("== typed ==")
	var list []activity
	in := `[{"name":"a","duration":1.5,"familyFriendly":true,"ref":"r1"},` +
		`{"NAME":"b","Duration":2,"familyfriendly":false,"REF":"r2"}]`
	err := json.Unmarshal([]byte(in), &list)
	fmt.Printf("list: %v %#v\n", err, list)

	var m mixed
	full := `{"b":true,"s":"x","i":-7,"i8":-128,"u64":18446744073709551615,` +
		`"f":0.5,"a":[1,{"k":null}],"l":[1,2,3],"m":{"k":9},"arr":[1,2,3,4],` +
		`"p":null,"sub":{"name":"s"},"-":5,"low":3,"Bad":8}`
	err = json.Unmarshal([]byte(full), &m)
	fmt.Printf("mixed: %v %+v\n", err, m)

	var short mixed
	err = json.Unmarshal([]byte(`{"arr":[9]}`), &short)
	fmt.Printf("shortarray: %v %v\n", err, short.Arr)

	var nulls mixed
	err = json.Unmarshal([]byte(`{"b":null,"s":null,"i":null,"l":null,"m":null,"a":null}`), &nulls)
	fmt.Printf("nulls: %v %+v\n", err, nulls)

	var wrong mixed
	err = json.Unmarshal([]byte(`{"i":"text","s":"kept"}`), &wrong)
	fmt.Printf("wrong: %T %v %+v\n", err, err, wrong)

	var over mixed
	err = json.Unmarshal([]byte(`{"i8":200}`), &over)
	fmt.Printf("over: %T %v %+v\n", err, err, over)

	var frac mixed
	err = json.Unmarshal([]byte(`{"i":1.5}`), &frac)
	fmt.Printf("fraction: %T %v %+v\n", err, err, frac)

	var c counter
	err = json.Unmarshal([]byte(` {"a": 1} `), &c)
	fmt.Printf("method: %v %+v\n", err, c)

	var num json.Number
	err = json.Unmarshal([]byte(`123`), &num)
	fmt.Printf("number: %v %q\n", err, num)

	var raw json.RawMessage
	err = json.Unmarshal([]byte(`{"a": 1}`), &raw)
	fmt.Printf("raw: %v %s\n", err, raw)
}

func errorCases() {
	fmt.Println("== errors ==")
	bad := []string{
		``,
		`   `,
		`1 2`,
		`1,`,
		`[1] x`,
		`{`,
		`}`,
		`[`,
		`]`,
		`[1,]`,
		`{"a":1,}`,
		`{a:1}`,
		`{"a" 1}`,
		`{"a":}`,
		`{"a":1`,
		`"abc`,
		`"a` + "\x01" + `b"`,
		`"a\qb"`,
		`"a\u00"`,
		`"a\u`,
		`tru`,
		`fals`,
		`nul`,
		`treu`,
		`bare`,
		`01`,
		`-`,
		`+1`,
		`.5`,
		`1.`,
		`1e`,
		`1e+`,
		`123e`,
		`-.5`,
		`00`,
		`1.2.3`,
		`[1 2]`,
		`{"a":1 "b":2}`,
		`[}`,
		`{]`,
	}
	for _, in := range bad {
		var v any
		err := json.Unmarshal([]byte(in), &v)
		fmt.Printf("%q -> %T %v %#v\n", in, err, err, v)
	}
	// A trailing space after a whole value is not an error.
	var ok any
	fmt.Println("trailing space:", json.Unmarshal([]byte("1 \t\r\n"), &ok), ok)
}

func depthCases() {
	fmt.Println("== depth ==")
	cases := []struct {
		name  string
		depth int
	}{{"9999", 9999}, {"10000", 10000}}
	for _, c := range cases {
		arr := `{"a":` + strings.Repeat(`[`, c.depth) + strings.Repeat(`]`, c.depth) + `}`
		obj := `{"a":` + strings.Repeat(`{"a":`, c.depth) + `0` + strings.Repeat(`}`, c.depth) + `}`
		var v any
		fmt.Printf("array %s: %v\n", c.name, errText(json.Unmarshal([]byte(arr), &v)))
		var w any
		fmt.Printf("object %s: %v\n", c.name, errText(json.Unmarshal([]byte(obj), &w)))
		fmt.Printf("valid array %s: %t\n", c.name, json.Valid([]byte(arr)))
		fmt.Printf("valid object %s: %t\n", c.name, json.Valid([]byte(obj)))
	}
	// The over stack case of TestUnmarshalMaxDepth holds 3,000,000 brackets.
	deep := `{"a":` + strings.Repeat(`[`, 300000) + strings.Repeat(`]`, 300000) + `}`
	var v any
	fmt.Printf("array 300000: %v\n", errText(json.Unmarshal([]byte(deep), &v)))
	var named struct {
		A any `json:"a"`
	}
	fmt.Printf("named field 300000: %v\n", errText(json.Unmarshal([]byte(deep), &named)))
	var missing struct {
		B any `json:"b"`
	}
	fmt.Printf("missing field 300000: %v\n", errText(json.Unmarshal([]byte(deep), &missing)))
	var c counter
	fmt.Printf("method 300000: %v %d\n", errText(json.Unmarshal([]byte(deep), &c)), c.N)
}

func validCases() {
	fmt.Println("== valid ==")
	inputs := []string{
		``, ` `, `null`, `1`, `-0`, `01`, `1.`, `.5`, `1e`, `"a"`, `"a`,
		`{}`, `[]`, `{"a":1}`, `{"a":1,"a":2}`, `[1,2]`, `[1,]`, `foo`,
		`}{`, `{]`, `{"a"}`, "\"\xff\"", `1 2`, "1\n", `"\ud800"`,
	}
	for _, in := range inputs {
		fmt.Printf("%q -> %t\n", in, json.Valid([]byte(in)))
	}
}

// errText writes the type and the text of an error, or the word none.
func errText(err error) string {
	if err == nil {
		return "none"
	}
	return fmt.Sprintf("%T %v", err, err)
}
