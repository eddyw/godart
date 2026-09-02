package main

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type Address struct {
	City    string `xml:"city"`
	Country string `xml:"country,attr"`
}

type Person struct {
	XMLName xml.Name `xml:"person"`
	ID      int      `xml:"id,attr"`
	Name    string   `xml:"name"`
	Age     uint8    `xml:"age"`
	Score   float32  `xml:"score"`
	Tags    []string `xml:"tags>tag"`
	Raw     []byte   `xml:"raw"`
	Addr    Address  `xml:"address"`
	Note    string   `xml:",comment"`
	Skip    string   `xml:"-"`
}

func main() {
	p := Person{
		ID:    7,
		Name:  "A & B",
		Age:   200,
		Score: 1.5,
		Tags:  []string{"first", "second"},
		Raw:   []byte("bytes here"),
		Addr:  Address{City: "Berlin", Country: "DE"},
		Note:  " a note ",
		Skip:  "never written",
	}
	out, err := xml.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Println("marshal:", err)
		return
	}
	fmt.Println(string(out))

	var back Person
	if err := xml.Unmarshal(out, &back); err != nil {
		fmt.Println("unmarshal:", err)
		return
	}
	fmt.Printf("id=%d name=%q age=%d score=%v tags=%v raw=%q city=%q country=%q skip=%q\n",
		back.ID, back.Name, back.Age, back.Score, back.Tags,
		back.Raw, back.Addr.City, back.Addr.Country, back.Skip)

	dec := xml.NewDecoder(strings.NewReader(`<a><b>1</b><b>2</b></a>`))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			fmt.Println("start:", t.Name.Local)
		case xml.CharData:
			if s := strings.TrimSpace(string(t)); s != "" {
				fmt.Println("chars:", s)
			}
		case xml.EndElement:
			fmt.Println("end:", t.Name.Local)
		}
	}

	if err := xml.Unmarshal([]byte("<a><unclosed>"), &back); err != nil {
		fmt.Println("a broken document gives:", err)
	}
	fmt.Println("escaped:", func() string {
		var sb strings.Builder
		xml.EscapeText(&sb, []byte(`<a> & "b"`))
		return sb.String()
	}())
}
