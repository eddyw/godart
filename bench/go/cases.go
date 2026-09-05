// Package cases holds the work of each benchmark case. The Go runner and the
// Dart driver call the same functions, thus the 2 numbers cover the same work.
//
// The document stands in a package variable. A call that takes it as an
// argument would pay the cost of the boundary at each call, and the case of
// the boundary measures that cost on its own.
package cases

import (
	"encoding/json"
	"os"
	"strings"
)

// Activity holds 1 element of activities.json.
type Activity struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	LocationName   string  `json:"locationName"`
	Duration       float64 `json:"duration"`
	TimeOfDay      string  `json:"timeOfDay"`
	FamilyFriendly bool    `json:"familyFriendly"`
	Price          float64 `json:"price"`
	DestinationRef string  `json:"destinationRef"`
	Ref            string  `json:"ref"`
	ImageURL       string  `json:"imageUrl"`
}

var (
	raw   []byte
	text  string
	typed []Activity
)

// Load reads the document of the benchmark and it gives the byte count.
func Load(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	raw = b
	text = string(b)
	return len(raw)
}

// DecodeAny decodes the document into the generic form.
func DecodeAny() int {
	var v []any
	if err := json.Unmarshal(raw, &v); err != nil {
		panic(err)
	}
	return len(v)
}

// DecodeTyped decodes the document into a slice of Activity.
func DecodeTyped() int {
	var v []Activity
	if err := json.Unmarshal(raw, &v); err != nil {
		panic(err)
	}
	typed = v
	return len(v)
}

// EncodeTyped writes the decoded value back. DecodeTyped must run first.
func EncodeTyped() int {
	b, err := json.Marshal(typed)
	if err != nil {
		panic(err)
	}
	return len(b)
}

// Scan reads the document with the scanner of encoding/json alone. It runs no
// decode, thus the pair of Scan and DecodeAny splits the cost of the decode.
func Scan() int {
	if !json.Valid(raw) {
		panic("the document is not valid")
	}
	return len(raw)
}

// CountBytes reads each byte of the document. It gives the cost of a plain
// byte loop over the same input.
func CountBytes() int {
	n := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == ',' {
			n++
		}
	}
	return n
}

const piece = "0123456789"

// Concat builds a string with n additions. Each addition copies the whole
// string that stands, thus the case measures the cost of the naive loop.
func Concat(n int) int {
	s := ""
	for i := 0; i < n; i++ {
		s += piece
	}
	return len(s)
}

// Builder builds the same string with strings.Builder.
func Builder(n int) int {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(piece)
	}
	return b.Len()
}

// Substr takes n substrings of 20 bytes from the document and it joins them.
func Substr(n int) int {
	var b strings.Builder
	span := len(text) - 21
	for i := 0; i < n; i++ {
		off := (i * 977) % span
		b.WriteString(text[off : off+20])
	}
	return b.Len()
}

// Text gives a string of n bytes. The Dart driver reads it for the case of
// the boundary.
func Text(n int) string {
	var b strings.Builder
	for b.Len() < n {
		b.WriteString(piece)
	}
	return b.String()[:n]
}

// Take reads a Dart string that crossed the boundary and it gives the length.
func Take(s string) int { return len(s) }
