//go:build dart

// Package flag reads the command line flags of a program.
//
// The compiler cannot read the real source. That source reads reflect.TypeOf
// and runtime.Caller, and the subset holds neither. The rules of the parse and
// the format of the report follow the standard library.
package flag

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"syscall/dart"
)

// ErrHelp is the error of the flag -h or -help with no declaration of its own.
var ErrHelp = errors.New("flag: help requested")

// Value holds a flag of a type that the package does not declare.
type Value interface {
	String() string
	Set(string) error
}

// Getter reads the value behind a flag.
type Getter interface {
	Value
	Get() any
}

// ErrorHandling names the action of a parse error.
type ErrorHandling int

const (
	// ContinueOnError gives the error to the caller.
	ContinueOnError ErrorHandling = iota
	// ExitOnError stops the program with the status 2.
	ExitOnError
	// PanicOnError panics.
	PanicOnError
)

// A Flag holds 1 declared flag.
type Flag struct {
	Name     string
	Usage    string
	Value    Value
	DefValue string

	// set marks a flag that the parse read. Go holds the same state in a map
	// of the set.
	set bool
}

// A FlagSet holds each flag of 1 command.
type FlagSet struct {
	// Usage writes the report of a parse error. A nil value writes the default
	// report.
	Usage func()

	name          string
	parsed        bool
	formal        map[string]*Flag
	order         []string
	args          []string
	errorHandling ErrorHandling
	output        io.Writer
}

// CommandLine holds the flags of the program.
var CommandLine = NewFlagSet(programName(), ExitOnError)

func programName() string {
	if len(os.Args) > 0 {
		return os.Args[0]
	}
	return "gart"
}

// NewFlagSet builds a set with a name and an error action.
func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet {
	return &FlagSet{
		name:          name,
		formal:        map[string]*Flag{},
		errorHandling: errorHandling,
	}
}

// Name gives the name of the set.
func (f *FlagSet) Name() string { return f.name }

// ErrorHandling gives the action of a parse error.
func (f *FlagSet) ErrorHandling() ErrorHandling { return f.errorHandling }

// SetOutput sets the writer that the report reaches.
func (f *FlagSet) SetOutput(out io.Writer) { f.output = out }

// Output gives the writer of the report.
func (f *FlagSet) Output() io.Writer {
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}

// The value types of the declared flags.

type boolValue bool

func (v *boolValue) String() string {
	if v == nil {
		return "false"
	}
	return strconv.FormatBool(bool(*v))
}

func (v *boolValue) Set(text string) error {
	n, err := strconv.ParseBool(text)
	if err != nil {
		return errors.New("parse error")
	}
	*v = boolValue(n)
	return nil
}

func (v *boolValue) Get() any { return bool(*v) }

// IsBoolFlag marks a flag that takes no separate argument.
func (v *boolValue) IsBoolFlag() bool { return true }

type intValue int

func (v *intValue) String() string {
	if v == nil {
		return "0"
	}
	return strconv.Itoa(int(*v))
}

func (v *intValue) Set(text string) error {
	n, err := strconv.Atoi(text)
	if err != nil {
		return errors.New("parse error")
	}
	*v = intValue(n)
	return nil
}

func (v *intValue) Get() any { return int(*v) }

type int64Value int64

func (v *int64Value) String() string {
	if v == nil {
		return "0"
	}
	return strconv.FormatInt(int64(*v), 10)
}

func (v *int64Value) Set(text string) error {
	n, err := strconv.ParseInt(text, 0, 64)
	if err != nil {
		return errors.New("parse error")
	}
	*v = int64Value(n)
	return nil
}

func (v *int64Value) Get() any { return int64(*v) }

type uintValue uint

func (v *uintValue) String() string {
	if v == nil {
		return "0"
	}
	return strconv.FormatUint(uint64(*v), 10)
}

func (v *uintValue) Set(text string) error {
	n, err := strconv.ParseUint(text, 0, 64)
	if err != nil {
		return errors.New("parse error")
	}
	*v = uintValue(n)
	return nil
}

func (v *uintValue) Get() any { return uint(*v) }

type uint64Value uint64

func (v *uint64Value) String() string {
	if v == nil {
		return "0"
	}
	return strconv.FormatUint(uint64(*v), 10)
}

func (v *uint64Value) Set(text string) error {
	n, err := strconv.ParseUint(text, 0, 64)
	if err != nil {
		return errors.New("parse error")
	}
	*v = uint64Value(n)
	return nil
}

func (v *uint64Value) Get() any { return uint64(*v) }

type float64Value float64

func (v *float64Value) String() string {
	if v == nil {
		return "0"
	}
	return strconv.FormatFloat(float64(*v), 'g', -1, 64)
}

func (v *float64Value) Set(text string) error {
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return errors.New("parse error")
	}
	*v = float64Value(n)
	return nil
}

func (v *float64Value) Get() any { return float64(*v) }

type stringValue string

func (v *stringValue) String() string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func (v *stringValue) Set(text string) error {
	*v = stringValue(text)
	return nil
}

func (v *stringValue) Get() any { return string(*v) }

// boolFlag marks a value that takes no separate argument.
type boolFlag interface {
	Value
	IsBoolFlag() bool
}

// Var declares a flag of a value of the caller.
func (f *FlagSet) Var(value Value, name string, usage string) {
	if _, ok := f.formal[name]; ok {
		panic(f.name + " flag redefined: " + name)
	}
	f.formal[name] = &Flag{Name: name, Usage: usage, Value: value, DefValue: value.String()}
	f.order = append(f.order, name)
}

func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string) {
	*p = value
	f.Var((*boolValue)(p), name, usage)
}

func (f *FlagSet) Bool(name string, value bool, usage string) *bool {
	p := new(bool)
	f.BoolVar(p, name, value, usage)
	return p
}

func (f *FlagSet) IntVar(p *int, name string, value int, usage string) {
	*p = value
	f.Var((*intValue)(p), name, usage)
}

func (f *FlagSet) Int(name string, value int, usage string) *int {
	p := new(int)
	f.IntVar(p, name, value, usage)
	return p
}

func (f *FlagSet) Int64Var(p *int64, name string, value int64, usage string) {
	*p = value
	f.Var((*int64Value)(p), name, usage)
}

func (f *FlagSet) Int64(name string, value int64, usage string) *int64 {
	p := new(int64)
	f.Int64Var(p, name, value, usage)
	return p
}

func (f *FlagSet) UintVar(p *uint, name string, value uint, usage string) {
	*p = value
	f.Var((*uintValue)(p), name, usage)
}

func (f *FlagSet) Uint(name string, value uint, usage string) *uint {
	p := new(uint)
	f.UintVar(p, name, value, usage)
	return p
}

func (f *FlagSet) Uint64Var(p *uint64, name string, value uint64, usage string) {
	*p = value
	f.Var((*uint64Value)(p), name, usage)
}

func (f *FlagSet) Uint64(name string, value uint64, usage string) *uint64 {
	p := new(uint64)
	f.Uint64Var(p, name, value, usage)
	return p
}

func (f *FlagSet) Float64Var(p *float64, name string, value float64, usage string) {
	*p = value
	f.Var((*float64Value)(p), name, usage)
}

func (f *FlagSet) Float64(name string, value float64, usage string) *float64 {
	p := new(float64)
	f.Float64Var(p, name, value, usage)
	return p
}

func (f *FlagSet) StringVar(p *string, name string, value string, usage string) {
	*p = value
	f.Var((*stringValue)(p), name, usage)
}

func (f *FlagSet) String(name string, value string, usage string) *string {
	p := new(string)
	f.StringVar(p, name, value, usage)
	return p
}

// Lookup gives the flag of the name, or nil.
func (f *FlagSet) Lookup(name string) *Flag { return f.formal[name] }

// Set writes the value of a declared flag.
func (f *FlagSet) Set(name, value string) error {
	flag, ok := f.formal[name]
	if !ok {
		return errors.New("no such flag -" + name)
	}
	if err := flag.Value.Set(value); err != nil {
		return err
	}
	return nil
}

// Args gives each argument after the flags.
func (f *FlagSet) Args() []string { return f.args }

// Arg gives 1 argument after the flags.
func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}

// NArg gives the count of the arguments after the flags.
func (f *FlagSet) NArg() int { return len(f.args) }

// NFlag gives the count of the flags that the parse read.
func (f *FlagSet) NFlag() int {
	n := 0
	for _, name := range f.order {
		if f.formal[name].set {
			n++
		}
	}
	return n
}

// Parsed reports whether Parse ran.
func (f *FlagSet) Parsed() bool { return f.parsed }

// VisitAll calls fn for each declared flag, in the order of the name.
func (f *FlagSet) VisitAll(fn func(*Flag)) {
	names := make([]string, len(f.order))
	copy(names, f.order)
	sort.Strings(names)
	for _, name := range names {
		fn(f.formal[name])
	}
}

// Visit calls fn for each flag that the parse read, in the order of the name.
func (f *FlagSet) Visit(fn func(*Flag)) {
	names := make([]string, 0, len(f.order))
	for _, name := range f.order {
		if f.formal[name].set {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		fn(f.formal[name])
	}
}

// PrintDefaults writes 1 block for each declared flag.
func (f *FlagSet) PrintDefaults() {
	f.VisitAll(func(flag *Flag) {
		line := "  -" + flag.Name
		name, usage := unquoteUsage(flag)
		if name != "" {
			line += " " + name
		}
		if len(line) <= 4 {
			line += "\t"
		} else {
			line += "\n    \t"
		}
		line += usage
		if !isZeroValue(flag, flag.DefValue) {
			if _, ok := flag.Value.(*stringValue); ok {
				line += " (default " + quote(flag.DefValue) + ")"
			} else {
				line += " (default " + flag.DefValue + ")"
			}
		}
		fmt.Fprintln(f.Output(), line)
	})
}

// unquoteUsage gives the name of the argument and the usage text. A name in
// back quotes inside the usage names the argument.
func unquoteUsage(flag *Flag) (string, string) {
	usage := flag.Usage
	for i := 0; i < len(usage); i++ {
		if usage[i] != '`' {
			continue
		}
		for j := i + 1; j < len(usage); j++ {
			if usage[j] == '`' {
				return usage[i+1 : j], usage[:i] + usage[i+1:j] + usage[j+1:]
			}
		}
		break
	}
	switch flag.Value.(type) {
	case *boolValue:
		return "", usage
	case *float64Value:
		return "float", usage
	case *intValue, *int64Value:
		return "int", usage
	case *stringValue:
		return "string", usage
	case *uintValue, *uint64Value:
		return "uint", usage
	}
	return "value", usage
}

// isZeroValue reports whether the text holds the zero value of the type. Go
// builds a zero value through reflect, and the intrinsic reads the type
// descriptor of the runtime.
func isZeroValue(flag *Flag, text string) bool {
	switch flag.Value.(type) {
	case *boolValue:
		return text == "false"
	case *stringValue:
		return text == ""
	case *intValue, *int64Value, *uintValue, *uint64Value, *float64Value:
		return text == "0"
	}
	return text == dart.ZeroText(flag.Value)
}

func quote(text string) string { return "\"" + text + "\"" }

// defaultUsage writes the head of the report and each flag.
func (f *FlagSet) defaultUsage() {
	if f.name == "" {
		fmt.Fprintln(f.Output(), "Usage:")
	} else {
		fmt.Fprintln(f.Output(), "Usage of "+f.name+":")
	}
	f.PrintDefaults()
}

// usage writes the report of a parse error.
func (f *FlagSet) usage() {
	if f.Usage != nil {
		f.Usage()
		return
	}
	f.defaultUsage()
}

// fail writes the cause and the report, and it takes the action of the set.
func (f *FlagSet) fail(cause string) error {
	fmt.Fprintln(f.Output(), cause)
	f.usage()
	switch f.errorHandling {
	case ExitOnError:
		os.Exit(2)
	case PanicOnError:
		panic(cause)
	}
	return errors.New(cause)
}

// Parse reads each flag of the arguments. It stops at the first argument that
// holds no flag, and at the argument --.
func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = arguments
	for {
		done, err := f.parseOne()
		if err != nil {
			return err
		}
		if done {
			break
		}
	}
	return nil
}

// parseOne reads 1 flag. The first result marks the end of the flags.
func (f *FlagSet) parseOne() (bool, error) {
	if len(f.args) == 0 {
		return true, nil
	}
	text := f.args[0]
	if len(text) < 2 || text[0] != '-' {
		return true, nil
	}
	count := 1
	if text[1] == '-' {
		count = 2
		if len(text) == 2 {
			f.args = f.args[1:]
			return true, nil
		}
	}
	name := text[count:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return true, f.fail("bad flag syntax: " + text)
	}
	f.args = f.args[1:]
	value := ""
	hasValue := false
	for i := 0; i < len(name); i++ {
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[:i]
			break
		}
	}
	flag, ok := f.formal[name]
	if !ok {
		if name == "help" || name == "h" {
			f.usage()
			return true, ErrHelp
		}
		return true, f.fail("flag provided but not defined: -" + name)
	}
	if b, ok := flag.Value.(boolFlag); ok && b.IsBoolFlag() {
		if !hasValue {
			value = "true"
		}
		if err := flag.Value.Set(value); err != nil {
			return true, f.fail("invalid boolean value " + quote(value) + " for -" + name + ": " + err.Error())
		}
		flag.set = true
		return false, nil
	}
	if !hasValue && len(f.args) > 0 {
		value, f.args = f.args[0], f.args[1:]
		hasValue = true
	}
	if !hasValue {
		return true, f.fail("flag needs an argument: -" + name)
	}
	if err := flag.Value.Set(value); err != nil {
		return true, f.fail("invalid value " + quote(value) + " for flag -" + name + ": " + err.Error())
	}
	flag.set = true
	return false, nil
}

// The functions of the command line set.

func Var(value Value, name string, usage string) { CommandLine.Var(value, name, usage) }

func BoolVar(p *bool, name string, value bool, usage string) {
	CommandLine.BoolVar(p, name, value, usage)
}

func Bool(name string, value bool, usage string) *bool {
	return CommandLine.Bool(name, value, usage)
}

func IntVar(p *int, name string, value int, usage string) {
	CommandLine.IntVar(p, name, value, usage)
}

func Int(name string, value int, usage string) *int { return CommandLine.Int(name, value, usage) }

func Int64Var(p *int64, name string, value int64, usage string) {
	CommandLine.Int64Var(p, name, value, usage)
}

func Int64(name string, value int64, usage string) *int64 {
	return CommandLine.Int64(name, value, usage)
}

func UintVar(p *uint, name string, value uint, usage string) {
	CommandLine.UintVar(p, name, value, usage)
}

func Uint(name string, value uint, usage string) *uint {
	return CommandLine.Uint(name, value, usage)
}

func Uint64Var(p *uint64, name string, value uint64, usage string) {
	CommandLine.Uint64Var(p, name, value, usage)
}

func Uint64(name string, value uint64, usage string) *uint64 {
	return CommandLine.Uint64(name, value, usage)
}

func Float64Var(p *float64, name string, value float64, usage string) {
	CommandLine.Float64Var(p, name, value, usage)
}

func Float64(name string, value float64, usage string) *float64 {
	return CommandLine.Float64(name, value, usage)
}

func StringVar(p *string, name string, value string, usage string) {
	CommandLine.StringVar(p, name, value, usage)
}

func String(name string, value string, usage string) *string {
	return CommandLine.String(name, value, usage)
}

func Lookup(name string) *Flag { return CommandLine.Lookup(name) }

func Set(name, value string) error { return CommandLine.Set(name, value) }

func Args() []string { return CommandLine.Args() }

func Arg(i int) string { return CommandLine.Arg(i) }

func NArg() int { return CommandLine.NArg() }

func NFlag() int { return CommandLine.NFlag() }

func Parsed() bool { return CommandLine.Parsed() }

func VisitAll(fn func(*Flag)) { CommandLine.VisitAll(fn) }

func Visit(fn func(*Flag)) { CommandLine.Visit(fn) }

func PrintDefaults() { CommandLine.PrintDefaults() }

// Usage writes the report of the command line set.
var Usage = func() { CommandLine.defaultUsage() }

// Parse reads each flag of the arguments of the program.
func Parse() {
	CommandLine.Usage = Usage
	CommandLine.Parse(os.Args[1:])
}
