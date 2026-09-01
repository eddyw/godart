package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type item struct {
	Name  string          `json:"name"`
	Count int             `json:"count"`
	Extra json.RawMessage `json:"extra"`
	Big   json.Number     `json:"big"`
}

func main() {
	decodeStream()
	decodeStrict()
	decodeErrors()
	encode()
	forms()
	rawAndNumber()
}

func decodeStream() {
	d := json.NewDecoder(strings.NewReader(` {"name":"a","count":1}
	{"name":"b","count":2} `))
	for {
		var it item
		err := d.Decode(&it)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("stop", err)
			break
		}
		fmt.Println(it.Name, it.Count, d.InputOffset())
	}

	d = json.NewDecoder(strings.NewReader(`1 2 3`))
	total := 0
	for d.More() {
		var n int
		if err := d.Decode(&n); err != nil {
			break
		}
		total += n
	}
	fmt.Println("total", total)

	d = json.NewDecoder(strings.NewReader(`{"name":"c"} tail`))
	var one item
	fmt.Println(d.Decode(&one) == nil, one.Name)
	rest, _ := io.ReadAll(d.Buffered())
	fmt.Println(strings.TrimSpace(string(rest)))
}

func decodeStrict() {
	d := json.NewDecoder(strings.NewReader(`{"name":"a","other":1}`))
	d.DisallowUnknownFields()
	var it item
	fmt.Println(d.Decode(&it))

	var loose item
	fmt.Println(json.Unmarshal([]byte(`{"name":"a","other":1}`), &loose) == nil)
}

func decodeErrors() {
	var it item
	err := json.Unmarshal([]byte(`{"name":1}`), &it)
	var typed *json.UnmarshalTypeError
	fmt.Println(errors.As(err, &typed), typed.Value, typed.Struct, typed.Field)
	fmt.Println(err)

	err = json.Unmarshal([]byte(`{"name":"a",}`), &it)
	var syntax *json.SyntaxError
	fmt.Println(errors.As(err, &syntax), syntax.Offset)
	fmt.Println(err)

	err = json.Unmarshal([]byte(`{"name":`), &it)
	fmt.Println(errors.As(err, &syntax), syntax.Offset, err)

	err = json.Unmarshal([]byte(`{} {}`), &it)
	fmt.Println(errors.As(err, &syntax), syntax.Offset, err)

	// The target passes through a variable, thus go vet reads no literal.
	var invalid *json.InvalidUnmarshalError
	var none any
	err = json.Unmarshal([]byte(`{}`), none)
	fmt.Println(errors.As(err, &invalid), err)
	var value any = it
	err = json.Unmarshal([]byte(`{}`), value)
	fmt.Println(errors.As(err, &invalid), err)
}

func encode() {
	var out bytes.Buffer
	e := json.NewEncoder(&out)
	if err := e.Encode(item{Name: "a", Count: 1}); err != nil {
		fmt.Println(err)
	}
	if err := e.Encode(item{Name: "b", Count: 2}); err != nil {
		fmt.Println(err)
	}
	fmt.Print(out.String())

	out.Reset()
	e = json.NewEncoder(&out)
	e.SetIndent("| ", "  ")
	if err := e.Encode(map[string]int{"a": 1}); err != nil {
		fmt.Println(err)
	}
	fmt.Print(out.String())
}

func forms() {
	texts := []string{
		`{"a":1}`, ` [1, 2] `, `{"a":1,}`, `01`, `1.5e3`, ``, `"x"`, `nul`,
		`[1,2`, `{"a"}`, `-`, `1.`, `"\q"`, `"a` + "\n" + `b"`,
	}
	for _, text := range texts {
		fmt.Print(json.Valid([]byte(text)), " ")
	}
	fmt.Println()

	var out bytes.Buffer
	src := []byte("{\n  \"a\" : [ 1, {\"b\":null} ],\n  \"c\" : \"x y\"\n}")
	if err := json.Compact(&out, src); err != nil {
		fmt.Println(err)
	}
	fmt.Println(out.String())

	out.Reset()
	if err := json.Indent(&out, src, "> ", "\t"); err != nil {
		fmt.Println(err)
	}
	fmt.Println(out.String())

	out.Reset()
	fmt.Println(json.Indent(&out, []byte(`{"a":}`), "", " "), out.Len())

	b, _ := json.MarshalIndent(map[string]any{"a": []int{1, 2}}, "# ", "  ")
	fmt.Println(string(b))
}

func rawAndNumber() {
	raw := json.RawMessage(`{"k":[1,2]}`)
	b, err := json.Marshal(map[string]json.RawMessage{"x": raw})
	fmt.Println(string(b), err == nil)

	var it item
	err = json.Unmarshal([]byte(`{"extra":{"z":[3]},"big":9007199254740993}`), &it)
	fmt.Println(string(it.Extra), it.Big, err == nil)

	n, err := it.Big.Int64()
	fmt.Println(n, err == nil)
	f, err := it.Big.Float64()
	fmt.Println(f > 9.0e15, err == nil)

	b, err = json.Marshal(it)
	fmt.Println(string(b), err == nil)

	var empty json.RawMessage
	b, err = json.Marshal(empty)
	fmt.Println(string(b), err == nil)
}
