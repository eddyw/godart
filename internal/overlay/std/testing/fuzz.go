//go:build dart

package testing

import (
	"context"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"syscall/dart"
)

// F holds the state of 1 fuzz target.
//
// gart generates no input. Fuzz runs the body 1 time for each seed of Add and 1
// time for each file of testdata/fuzz/<name>, which is the behavior of go test
// with no -fuzz flag. Read docs/parity.md.
type F struct {
	files []InternalFuzzInput
	t     *T
	// corpus holds 1 input for each seed and for each corpus file, in the order
	// of the run.
	corpus []corpusEntry
	// fuzzed marks a call of Fuzz.
	fuzzed bool
}

// corpusEntry holds 1 input of the seed corpus. The name is the name of the
// subtest that runs the input.
type corpusEntry struct {
	name   string
	values []any
}

// corpusDir holds the corpus files of each fuzz target, 1 directory for each
// target. go test reads the same place.
const corpusDir = "testdata/fuzz"

// encVersion1 is the first line of a corpus file of the version 1.
const encVersion1 = "go test fuzz v1"

// fuzzTypes names each type that a seed and a parameter of a fuzz target can
// hold. Go holds the same list, and byte and rune stand under their own names
// uint8 and int32.
var fuzzTypes = map[string]bool{
	"[]uint8": true,
	"string":  true,
	"bool":    true,
	"uint8":   true,
	"int32":   true,
	"float32": true,
	"float64": true,
	"int":     true,
	"int8":    true,
	"int16":   true,
	"int64":   true,
	"uint":    true,
	"uint16":  true,
	"uint32":  true,
	"uint64":  true,
}

func (f *F) Name() string  { return f.t.Name() }
func (f *F) Failed() bool  { return f.t.Failed() }
func (f *F) Skipped() bool { return f.t.Skipped() }

func (f *F) Helper() { markHelper() }

func (f *F) Cleanup(g func())                  { f.t.Cleanup(g) }
func (f *F) Fail()                             { f.t.Fail() }
func (f *F) FailNow()                          { f.t.FailNow() }
func (f *F) SkipNow()                          { f.t.SkipNow() }
func (f *F) Log(args ...any)                   { f.t.Log(args...) }
func (f *F) Logf(format string, args ...any)   { f.t.Logf(format, args...) }
func (f *F) Error(args ...any)                 { f.t.Error(args...) }
func (f *F) Errorf(format string, args ...any) { f.t.Errorf(format, args...) }
func (f *F) Fatal(args ...any)                 { f.t.Fatal(args...) }
func (f *F) Fatalf(format string, args ...any) { f.t.Fatalf(format, args...) }
func (f *F) Skip(args ...any)                  { f.t.Skip(args...) }
func (f *F) Skipf(format string, args ...any)  { f.t.Skipf(format, args...) }

func (f *F) ArtifactDir() string      { return f.t.ArtifactDir() }
func (f *F) Attr(key, value string)   { f.t.Attr(key, value) }
func (f *F) Chdir(dir string)         { f.t.Chdir(dir) }
func (f *F) Context() context.Context { return f.t.Context() }
func (f *F) Output() io.Writer        { return f.t.Output() }
func (f *F) Setenv(key, value string) { f.t.Setenv(key, value) }
func (f *F) TempDir() string          { return f.t.TempDir() }

// Add gives 1 input to the seed corpus of the target. The values must match the
// parameters of the body that Fuzz takes.
func (f *F) Add(args ...any) {
	values := make([]any, 0, len(args))
	for _, arg := range args {
		if name := typeName(arg); !fuzzTypes[name] {
			panic("testing: unsupported type to Add " + name)
		}
		values = append(values, arg)
	}
	f.corpus = append(f.corpus, corpusEntry{
		name:   "seed#" + strconv.Itoa(len(f.corpus)),
		values: values,
	})
}

// Fuzz runs the body 1 time for each input of the seed corpus, and each run is
// a subtest.
//
// The body takes a *T and 1 parameter for each value of an input. gart
// generates no new input, thus the run holds each seed of Add and each file of
// testdata/fuzz/<name> alone.
func (f *F) Fuzz(body any) {
	if f.fuzzed {
		panic("testing: F.Fuzz called more than once")
	}
	f.fuzzed = true
	if f.t.failed {
		return
	}
	types := f.signature(body)
	for _, entry := range f.corpus {
		if cause := checkCorpus(entry.values, types); cause != "" {
			f.Fatal(cause)
		}
	}
	files, cause := readCorpus(filepath.Join(corpusDir, f.t.name), f.files, types)
	if cause != "" {
		f.Fatal(cause)
	}
	f.corpus = append(f.corpus, files...)
	for _, entry := range f.corpus {
		values := entry.values
		f.t.runSub(entry.name, func(t *T) {
			args := make([]any, 0, len(values)+1)
			args = append(args, t)
			args = append(args, values...)
			dart.CallValue(body, args)
		})
	}
}

// signature reads the type of the body of a fuzz target, and it gives the name
// of each parameter behind the *T.
func (f *F) signature(body any) []string {
	desc := dart.DescOf(body)
	if desc == 0 {
		panic("testing: F.Fuzz must receive a function")
	}
	if _, kind, _ := dart.DescInfo(desc); kind != "func" {
		panic("testing: F.Fuzz must receive a function")
	}
	count := dart.DescNumIn(desc)
	if count < 2 || dart.DescIn(desc, 0) != dart.DescOf(f.t) {
		panic("testing: fuzz target must receive at least two arguments, " +
			"where the first argument is a *T")
	}
	if dart.DescNumOut(desc) != 0 {
		panic("testing: fuzz target must not return a value")
	}
	out := make([]string, 0, count-1)
	for i := 1; i < count; i++ {
		name, _, _ := dart.DescInfo(dart.DescIn(desc, i))
		if !fuzzTypes[name] {
			panic("testing: unsupported type for fuzzing " + name)
		}
		out = append(out, name)
	}
	return out
}

// typeName gives the name of the type of a value, in the words of the package
// reflect. A nil value gives the name that Go writes for it.
func typeName(v any) string {
	handle := dart.DescOf(v)
	if handle == 0 {
		return "<nil>"
	}
	name, _, _ := dart.DescInfo(handle)
	return name
}

// checkCorpus reports whether the values of 1 input match the parameters of the
// body. The result names the cause of a mismatch, and it is empty for a match.
func checkCorpus(values []any, types []string) string {
	if len(values) != len(types) {
		return "wrong number of values in corpus entry: " +
			strconv.Itoa(len(values)) + ", want " + strconv.Itoa(len(types))
	}
	names := make([]string, len(values))
	for i, v := range values {
		names[i] = typeName(v)
	}
	for i := range types {
		if names[i] != types[i] {
			return "mismatched types in corpus entry: [" +
				strings.Join(names, " ") + "], want [" +
				strings.Join(types, " ") + "]"
		}
	}
	return ""
}

// readCorpus validates the original corpus bytes embedded by the compiler.
// The ordered snapshot lets browser targets use the same seeds without file I/O.
func readCorpus(dir string, files []InternalFuzzInput, types []string) ([]corpusEntry, string) {
	var out []corpusEntry
	var causes []string
	for _, file := range files {
		name := filepath.Join(dir, file.Name)
		values, cause := unmarshalCorpusFile(file.Data)
		if cause != "" {
			cause = "unmarshal: " + cause
		} else {
			cause = checkCorpus(values, types)
		}
		if cause != "" {
			causes = append(causes, strconv.Quote(name)+": "+cause)
			continue
		}
		out = append(out, corpusEntry{name: file.Name, values: values})
	}
	if len(causes) > 0 {
		return out, strings.Join(causes, "\n")
	}
	return out, ""
}

// unmarshalCorpusFile reads the versioned text form of 1 corpus file. The first
// line names the version, and each line behind it holds 1 value.
func unmarshalCorpusFile(text string) ([]any, string) {
	if text == "" {
		return nil, "cannot unmarshal empty string"
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return nil, "must include version and at least one value"
	}
	if version := strings.TrimSuffix(lines[0], "\r"); version != encVersion1 {
		return nil, "unknown encoding version: " + version
	}
	var out []any
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		value, cause := parseCorpusValue(line)
		if cause != "" {
			return nil, "malformed line " + strconv.Quote(line) + ": " + cause
		}
		out = append(out, value)
	}
	return out, ""
}

// parseCorpusValue reads 1 line of a corpus file. A line holds 1 value as a Go
// literal with its type in front of it, for example `[]byte("hello")`.
//
// The second result names the cause of a line that the grammar rejects, and it
// is empty for a line that reads.
func parseCorpusValue(line string) (any, string) {
	open := strings.IndexByte(line, '(')
	if open < 0 || !strings.HasSuffix(line, ")") {
		return nil, "expected call expression"
	}
	name := strings.TrimSpace(line[:open])
	arg := strings.TrimSpace(line[open+1 : len(line)-1])
	switch name {
	case "[]byte":
		if !quoted(arg) {
			return nil, "string literal required for type []byte"
		}
		text, err := strconv.Unquote(arg)
		if err != nil {
			return nil, err.Error()
		}
		return []byte(text), ""
	case "string":
		if !quoted(arg) {
			return nil, "string literal value required for type string"
		}
		text, err := strconv.Unquote(arg)
		if err != nil {
			return nil, err.Error()
		}
		return text, ""
	case "bool":
		switch arg {
		case "true":
			return true, ""
		case "false":
			return false, ""
		}
		return nil, "true or false required for type bool"
	case "byte", "rune":
		return parseChar(arg, name)
	case "int", "int8", "int16", "int32", "int64":
		return parseInt(arg, name)
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return parseUint(arg, name)
	case "float32":
		v, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return nil, err.Error()
		}
		return float32(v), ""
	case "float64":
		v, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return nil, err.Error()
		}
		return v, ""
	case "math.Float32frombits":
		bits, err := strconv.ParseUint(arg, 0, 32)
		if err != nil {
			return nil, err.Error()
		}
		return dart.Float32FromBits(uint32(bits)), ""
	case "math.Float64frombits":
		bits, err := strconv.ParseUint(arg, 0, 64)
		if err != nil {
			return nil, err.Error()
		}
		return dart.Float64FromBits(bits), ""
	}
	return nil, "expected []byte or primitive type"
}

// quoted reports whether the text is a string literal of Go. A raw literal
// stands between 2 back quotes, and each other 1 between 2 double quotes.
func quoted(text string) bool {
	if len(text) < 2 {
		return false
	}
	head, tail := text[0], text[len(text)-1]
	return (head == '"' && tail == '"') || (head == '`' && tail == '`')
}

// parseChar reads the value of a byte or of a rune. Go writes a character
// literal for it, and an integer literal also reads.
func parseChar(text, name string) (any, string) {
	if !strings.HasPrefix(text, "'") {
		if name == "rune" {
			return parseInt(text, "rune")
		}
		return parseUint(text, "byte")
	}
	if len(text) < 2 || !strings.HasSuffix(text, "'") {
		return nil, "malformed character literal, missing single quotes"
	}
	code, _, _, err := strconv.UnquoteChar(text[1:len(text)-1], '\'')
	if err != nil {
		return nil, err.Error()
	}
	if name == "rune" {
		return code, ""
	}
	if code >= 256 {
		return nil, "can only encode single byte to a byte type"
	}
	return byte(code), ""
}

// intBits names the count of bits of each signed type of a corpus file.
var intBits = map[string]int{
	"int": 64, "int8": 8, "int16": 16, "int32": 32, "rune": 32, "int64": 64,
}

// uintBits names the count of bits of each unsigned type of a corpus file.
var uintBits = map[string]int{
	"uint": 64, "uint8": 8, "byte": 8, "uint16": 16, "uint32": 32, "uint64": 64,
}

// parseInt reads a whole number of a signed type and gives a value of that
// type.
//
// Go reads an int of the corpus with 64 bits, because a run on another machine
// can write such a value. The int of gart holds 64 bits as well.
func parseInt(text, name string) (any, string) {
	n, err := strconv.ParseInt(text, 0, intBits[name])
	if err != nil {
		return nil, err.Error()
	}
	switch name {
	case "int":
		return int(n), ""
	case "int8":
		return int8(n), ""
	case "int16":
		return int16(n), ""
	case "int32", "rune":
		return int32(n), ""
	}
	return n, ""
}

// parseUint reads a whole number of an unsigned type and gives a value of that
// type.
func parseUint(text, name string) (any, string) {
	n, err := strconv.ParseUint(text, 0, uintBits[name])
	if err != nil {
		return nil, err.Error()
	}
	switch name {
	case "uint":
		return uint(n), ""
	case "uint8", "byte":
		return uint8(n), ""
	case "uint16":
		return uint16(n), ""
	case "uint32":
		return uint32(n), ""
	}
	return n, ""
}

// runFuzzTargets runs the seed corpus of each fuzz target, and it reports
// whether 1 target ran and whether each 1 passed.
//
// go test runs the targets after the tests and in front of the examples, and
// the flag -count runs the whole list again.
func runFuzzTargets(targets []InternalFuzzTarget, flags options) (ran, ok bool) {
	ok = true
	for round := 0; round < flags.rounds; round++ {
		g := &group{}
		for _, target := range targets {
			if !matches(target.Name, flags.run) || skips(target.Name) {
				continue
			}
			ran = true
			t := &T{name: target.Name}
			f := &F{t: t, files: FuzzCorpus[target.Name]}
			fn := target.Fn
			// F holds no Parallel, thus the target never parks and the group
			// needs no second pass.
			g.run(t, func() {
				fn(f)
				if !f.fuzzed && !t.skipped && !t.failed {
					t.Error("returned without calling F.Fuzz, F.Fail, or F.Skip")
				}
			})
			for _, l := range t.report {
				emit(l)
			}
			if t.failed {
				ok = false
			}
		}
	}
	return ran, ok
}
