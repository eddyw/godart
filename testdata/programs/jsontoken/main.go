package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type row struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	// Token walks a whole document.
	d := json.NewDecoder(strings.NewReader(`{"a":[1,"two",true,null],"b":{"c":1.5}}`))
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("token error:", err)
			return
		}
		fmt.Printf("%T %v\n", t, t)
	}

	// Token and Decode together read a stream of a large array.
	d = json.NewDecoder(strings.NewReader(`[{"name":"a","age":1},{"name":"b","age":2}]`))
	if _, err := d.Token(); err != nil {
		fmt.Println("open:", err)
		return
	}
	for d.More() {
		var r row
		if err := d.Decode(&r); err != nil {
			fmt.Println("decode:", err)
			return
		}
		fmt.Println(r.Name, r.Age)
	}
	t, err := d.Token()
	fmt.Println("close:", t, err)

	// UseNumber keeps the text of the literal.
	d = json.NewDecoder(strings.NewReader(`{"big":18446744073709551615,"f":1.50}`))
	d.UseNumber()
	var m map[string]any
	if err := d.Decode(&m); err != nil {
		fmt.Println("usenumber:", err)
		return
	}
	fmt.Println(m["big"], m["f"])
	n := m["big"].(json.Number)
	fmt.Println(n.String())

	// A uint64 above the largest int64 keeps its value.
	var u struct {
		U uint64 `json:"u"`
	}
	if err := json.Unmarshal([]byte(`{"u":18446744073709551615}`), &u); err != nil {
		fmt.Println("uint:", err)
		return
	}
	fmt.Println(u.U)

	// A bad token gives an error. The text of the error differs from Go 1.27,
	// which routes encoding/json through the v2 implementation. Read
	// docs/parity.md.
	d = json.NewDecoder(strings.NewReader(`{]`))
	if _, err := d.Token(); err != nil {
		fmt.Println("open:", err)
	}
	_, err = d.Token()
	fmt.Println("bad token:", err != nil)

	fmt.Println(json.Delim('['), json.Delim('}'))
}
