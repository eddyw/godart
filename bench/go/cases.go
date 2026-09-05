// Package cases holds the work of each benchmark case. The Go runner and the
// Dart driver call the same functions, thus the 2 numbers cover the same work.
//
// The document stands in a package variable. A call that takes it as an
// argument would pay the cost of the boundary at each call, and the case of
// the boundary measures that cost on its own.
package cases

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
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

// numbers and decimals hold the input of IntParse and of FloatParse. A build
// of the text inside the loop would measure the write and not the read.
var (
	numbers  = []string{"0", "7", "42", "1234", "99999", "2147483647", "-58", "300"}
	decimals = []string{"0.5", "3.14159", "-2.5", "1e3", "12345.6789", "0.0001"}
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

// The cases below hold no string and no document. They measure the work that
// each client does between 2 frames: arithmetic, a list, and a map.
//
// A Dart int and a Go int each hold 64 bits. A JavaScript number holds 53
// bits of an integer, thus IntHash gives a value that a JS number cannot
// hold, and the bun form of that case reads a BigInt.

// step gives the next value of a linear congruential generator. Each runtime
// builds the same sequence with it, thus each sort reads the same input.
func step(x int) int { return (x*1103515245 + 12345) & 0x7FFFFFFF }

// IntSum adds the first n integers. The total stays under 2^53, thus a JS
// number holds it with no loss.
func IntSum(n int) int {
	s := 0
	for i := 0; i < n; i++ {
		s += i
	}
	return s
}

// IntHash mixes n values with a 64 bit multiply that wraps. Go and Dart wrap
// at 64 bits, and a JS number cannot hold the value.
func IntHash(n int) int {
	h := 0x1000193
	for i := 0; i < n; i++ {
		h = (h ^ i) * 0x100000001B3
	}
	return h
}

// IntDiv takes the quotient and the remainder n times.
func IntDiv(n int) int {
	s := 0
	for i := 1; i <= n; i++ {
		s += 1000003/i + 1000003%i
	}
	return s
}

// IntParse reads n integers from text.
func IntParse(n int) int {
	s := 0
	for i := 0; i < n; i++ {
		v, err := strconv.Atoi(numbers[i%len(numbers)])
		if err != nil {
			panic(err)
		}
		s += v
	}
	return s
}

// IntFormat writes n integers as text.
func IntFormat(n int) int {
	s := 0
	for i := 0; i < n; i++ {
		s += len(strconv.Itoa(i * 7919))
	}
	return s
}

// FloatSum adds n float64 values.
func FloatSum(n int) float64 {
	s := 0.0
	for i := 0; i < n; i++ {
		s += float64(i) * 0.5
	}
	return s
}

// FloatMath runs the work of a layout pass: a square root and a multiply
// with an addition, n times.
func FloatMath(n int) float64 {
	s := 0.0
	for i := 0; i < n; i++ {
		x := float64(i) * 0.001
		s += math.Sqrt(x)*1.5 + x*x
	}
	return s
}

// FloatParse reads n float64 values from text.
func FloatParse(n int) int {
	s := 0
	for i := 0; i < n; i++ {
		v, err := strconv.ParseFloat(decimals[i%len(decimals)], 64)
		if err != nil {
			panic(err)
		}
		s += int(v)
	}
	return s
}

// FloatFormat writes n float64 values as text.
func FloatFormat(n int) int {
	s := 0
	for i := 0; i < n; i++ {
		s += len(strconv.FormatFloat(float64(i)*1.5, 'f', 3, 64))
	}
	return s
}

// SliceAppend grows a slice to n elements with append.
func SliceAppend(n int) int {
	var s []int
	for i := 0; i < n; i++ {
		s = append(s, i)
	}
	return len(s)
}

// SliceSort sorts n integers of the generator.
func SliceSort(n int) int {
	s := make([]int, n)
	x := 1
	for i := 0; i < n; i++ {
		x = step(x)
		s[i] = x
	}
	sort.Ints(s)
	return s[n/2]
}

// MapString fills a map of n string keys and it reads each key 1 time.
func MapString(n int) int {
	m := make(map[string]int, n)
	for i := 0; i < n; i++ {
		m[strconv.Itoa(i)] = i
	}
	s := 0
	for i := 0; i < n; i++ {
		s += m[strconv.Itoa(i)]
	}
	return s
}

// MapInt fills a map of n integer keys and it reads each key 1 time.
func MapInt(n int) int {
	m := make(map[int]int, n)
	for i := 0; i < n; i++ {
		m[i*7919] = i
	}
	s := 0
	for i := 0; i < n; i++ {
		s += m[i*7919]
	}
	return s
}

// Point holds the 4 numbers of a rectangle. StructAlloc builds n of them.
type Point struct {
	X, Y, W, H float64
}

// StructAlloc builds n rectangles and it adds the area of each 1.
func StructAlloc(n int) float64 {
	s := 0.0
	for i := 0; i < n; i++ {
		p := Point{X: float64(i), Y: float64(i) * 2, W: 10, H: 20}
		s += p.W*p.H + p.X + p.Y
	}
	return s
}

// Labels gives count strings of len bytes each. The Dart driver reads them to
// measure 1 frame of a screen that holds that many text fields.
func Labels(count, size int) []string {
	out := make([]string, count)
	for i := range out {
		var b strings.Builder
		b.WriteString(strconv.Itoa(i))
		for b.Len() < size {
			b.WriteString(piece)
		}
		out[i] = b.String()[:size]
	}
	return out
}

// The cases below hold the text work of a client: a format, a split, a join, a
// replace, and a sort of a struct by a field.

var words = []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}

// Sprintf builds n lines with 3 verbs.
func Sprintf(n int) int {
	total := 0
	for i := 0; i < n; i++ {
		s := fmt.Sprintf("%s-%d-%.2f", words[i%len(words)], i, float64(i)*1.5)
		total += len(s)
	}
	return total
}

// Split cuts a line of 6 words n times.
func Split(n int) int {
	line := strings.Join(words, ",")
	total := 0
	for i := 0; i < n; i++ {
		total += len(strings.Split(line, ","))
	}
	return total
}

// Join builds a line of 6 words n times.
func Join(n int) int {
	total := 0
	for i := 0; i < n; i++ {
		total += len(strings.Join(words, ","))
	}
	return total
}

// Replace changes each comma of a line n times.
func Replace(n int) int {
	line := strings.Join(words, ",")
	total := 0
	for i := 0; i < n; i++ {
		total += len(strings.ReplaceAll(line, ",", " | "))
	}
	return total
}

// Row holds 1 element of the sort case.
type Row struct {
	Key  int
	Name string
}

// SortStruct orders n rows by the key.
func SortStruct(n int) int {
	rows := make([]Row, n)
	x := 1
	for i := range rows {
		x = step(x)
		rows[i] = Row{Key: x, Name: words[i%len(words)]}
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].Key < rows[b].Key })
	return rows[n/2].Key
}
