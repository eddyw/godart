//go:build dart

// Package testing gives the subset of the standard testing package that the
// gart compiler supports.
//
// The compiler writes the entry point of a test binary, and that entry point
// calls Main. The output holds the format of `go test`, without the source
// position of a log line. Read docs/parity.md for each gap.
package testing

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/dart"
)

// InternalTest names 1 test function. The compiler writes the list.
type InternalTest struct {
	Name string
	F    func(*T)
}

// TB holds the methods that T, B, and F share.
type TB interface {
	Cleanup(f func())
	Error(args ...any)
	Errorf(format string, args ...any)
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
	Name() string
	Skip(args ...any)
	SkipNow()
	Skipf(format string, args ...any)
	Skipped() bool
}

// verbose holds the state of the -v flag of gart test. The compiler writes the
// value into the call of Main.
var verbose bool

// Verbose reports whether the build asked for 1 line for each test.
func Verbose() bool { return verbose }

// Short reports whether the build asked for the short form of a test. gart has
// no flag for it, thus the result is always false.
func Short() bool { return false }

// T holds the state of 1 test.
type T struct {
	name     string
	level    int
	failed   bool
	skipped  bool
	logs     []string
	cleanups []func()
	// report holds the result line of each subtest, and of the subtests of a
	// subtest. Go writes them after the line of the parent.
	report []string
}

func (t *T) Name() string { return t.name }

func (t *T) Failed() bool { return t.failed }

func (t *T) Skipped() bool { return t.skipped }

// Helper marks a helper function. Go leaves the position of a helper out of a
// log line. A log line of gart holds no position, thus the method does nothing.
func (t *T) Helper() {}

// Parallel does nothing. All Go code runs in 1 isolate, thus gart runs each
// test in the order of the list.
func (t *T) Parallel() {}

func (t *T) Cleanup(f func()) { t.cleanups = append(t.cleanups, f) }

func (t *T) Fail() { t.failed = true }

func (t *T) FailNow() {
	t.failed = true
	dart.Stop()
}

func (t *T) SkipNow() {
	t.skipped = true
	dart.Stop()
}

func (t *T) Log(args ...any) { t.write(fmt.Sprintln(args...)) }

func (t *T) Logf(format string, args ...any) {
	t.write(withNewline(fmt.Sprintf(format, args...)))
}

func (t *T) Error(args ...any) {
	t.write(fmt.Sprintln(args...))
	t.Fail()
}

func (t *T) Errorf(format string, args ...any) {
	t.write(withNewline(fmt.Sprintf(format, args...)))
	t.Fail()
}

func (t *T) Fatal(args ...any) {
	t.write(fmt.Sprintln(args...))
	t.FailNow()
}

func (t *T) Fatalf(format string, args ...any) {
	t.write(withNewline(fmt.Sprintf(format, args...)))
	t.FailNow()
}

func (t *T) Skip(args ...any) {
	t.write(fmt.Sprintln(args...))
	t.SkipNow()
}

func (t *T) Skipf(format string, args ...any) {
	t.write(withNewline(fmt.Sprintf(format, args...)))
	t.SkipNow()
}

// Run runs a subtest and reports whether the subtest passed.
func (t *T) Run(name string, f func(*T)) bool {
	sub := &T{name: t.name + "/" + subName(name), level: t.level + 1}
	runOne(sub, func() { f(sub) })
	t.report = append(t.report, sub.report...)
	if sub.failed {
		t.failed = true
	}
	return !sub.failed
}

// write holds 1 log message. Verbose mode writes it at once, and the plain
// mode writes it only when the test fails.
func (t *T) write(text string) {
	if !verbose {
		t.logs = append(t.logs, text)
		return
	}
	for _, l := range indentLines(1, text) {
		emit(l)
	}
}

// Main runs each test of a test binary. The compiler writes the call into the
// entry point of the binary.
//
// The flags come from the arguments of the program. gart test writes -test.v
// for a line for each test, and -test.run=<pattern> for a filter.
func Main(tests []InternalTest) {
	pattern := ""
	for _, arg := range dart.Args() {
		switch {
		case arg == "-test.v" || arg == "-test.v=true":
			verbose = true
		case strings.HasPrefix(arg, "-test.run="):
			pattern = strings.TrimPrefix(arg, "-test.run=")
		}
	}
	failed := false
	for _, tc := range tests {
		if !matches(tc.Name, pattern) {
			continue
		}
		t := &T{name: tc.Name}
		f := tc.F
		runOne(t, func() { f(t) })
		for _, l := range t.report {
			emit(l)
		}
		if t.failed {
			failed = true
		}
	}
	if failed {
		emit("FAIL")
		dart.Exit(1)
		return
	}
	emit("PASS")
}

// matches reports whether a test name passes the filter of -test.run.
//
// go test reads the pattern as a regular expression. The subset holds no
// regexp package, thus gart reads the pattern as plain text. A subtest of a
// test that matches also runs, because the name of the subtest holds the name
// of the test.
func matches(name, pattern string) bool {
	return pattern == "" || strings.Contains(name, pattern)
}

// runOne runs 1 test body. It puts the result line of the test in front of the
// result lines of the subtests, exactly as go test does.
func runOne(t *T, call func()) {
	if verbose {
		emit("=== RUN   " + t.name)
	}
	start := dart.NowNanos()
	dart.Catch(call)
	for i := len(t.cleanups) - 1; i >= 0; i-- {
		dart.Catch(t.cleanups[i])
	}
	head := fmt.Sprintf("%s--- %s: %s (%ss)", indent(t.level), state(t), t.name,
		seconds(dart.NowNanos()-start))
	if verbose {
		t.report = append([]string{head}, t.report...)
		return
	}
	if !t.failed {
		t.report = nil
		return
	}
	out := []string{head}
	for _, text := range t.logs {
		out = append(out, indentLines(t.level+1, text)...)
	}
	t.report = append(out, t.report...)
}

func state(t *T) string {
	if t.failed {
		return "FAIL"
	}
	if t.skipped {
		return "SKIP"
	}
	return "PASS"
}

// emit writes 1 line to standard output.
func emit(text string) { fmt.Print(text + "\n") }

func indent(level int) string { return strings.Repeat("    ", level) }

// indentLines splits a message into lines and gives each line an indent.
func indentLines(level int, text string) []string {
	parts := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, indent(level)+part)
	}
	return out
}

// withNewline adds a newline when the text has none. Go does the same for Logf.
func withNewline(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

// subName maps a subtest name to the name that go test writes. A space becomes
// an underscore.
func subName(name string) string { return strings.ReplaceAll(name, " ", "_") }

// seconds writes a count of nanoseconds as a count of seconds, with 2 digits
// after the point.
func seconds(nanoseconds int64) string {
	return strconv.FormatFloat(float64(nanoseconds)/1000000000.0, 'f', 2, 64)
}

// B holds the state of 1 benchmark.
//
// gart runs no benchmark, exactly as `go test` with no -bench flag. The type
// exists because a test file declares a benchmark, and that body must type
// check. Each method stops the program, thus no body gives a wrong result.
type B struct {
	N int
}

const noBenchmark = "gart runs no benchmark"

func (b *B) Name() string                            { panic(noBenchmark) }
func (b *B) Failed() bool                            { panic(noBenchmark) }
func (b *B) Skipped() bool                           { panic(noBenchmark) }
func (b *B) Helper()                                 { panic(noBenchmark) }
func (b *B) Parallel()                               { panic(noBenchmark) }
func (b *B) Cleanup(f func())                        { panic(noBenchmark) }
func (b *B) Fail()                                   { panic(noBenchmark) }
func (b *B) FailNow()                                { panic(noBenchmark) }
func (b *B) SkipNow()                                { panic(noBenchmark) }
func (b *B) Log(args ...any)                         { panic(noBenchmark) }
func (b *B) Logf(format string, args ...any)         { panic(noBenchmark) }
func (b *B) Error(args ...any)                       { panic(noBenchmark) }
func (b *B) Errorf(format string, args ...any)       { panic(noBenchmark) }
func (b *B) Fatal(args ...any)                       { panic(noBenchmark) }
func (b *B) Fatalf(format string, args ...any)       { panic(noBenchmark) }
func (b *B) Skip(args ...any)                        { panic(noBenchmark) }
func (b *B) Skipf(format string, args ...any)        { panic(noBenchmark) }
func (b *B) ResetTimer()                             { panic(noBenchmark) }
func (b *B) StartTimer()                             { panic(noBenchmark) }
func (b *B) StopTimer()                              { panic(noBenchmark) }
func (b *B) ReportAllocs()                           { panic(noBenchmark) }
func (b *B) SetBytes(n int64)                        { panic(noBenchmark) }
func (b *B) ReportMetric(value float64, unit string) { panic(noBenchmark) }
func (b *B) Run(name string, f func(*B)) bool        { panic(noBenchmark) }
func (b *B) RunParallel(body func(*PB))              { panic(noBenchmark) }

// PB drives a parallel benchmark. gart runs no benchmark.
type PB struct{}

func (p *PB) Next() bool { panic(noBenchmark) }

// F holds the state of 1 fuzz target. gart runs no fuzz target, and Main
// reports each target as a skipped test.
type F struct{}

const noFuzz = "gart runs no fuzz target"

func (f *F) Name() string                      { panic(noFuzz) }
func (f *F) Failed() bool                      { panic(noFuzz) }
func (f *F) Skipped() bool                     { panic(noFuzz) }
func (f *F) Helper()                           { panic(noFuzz) }
func (f *F) Cleanup(g func())                  { panic(noFuzz) }
func (f *F) Fail()                             { panic(noFuzz) }
func (f *F) FailNow()                          { panic(noFuzz) }
func (f *F) SkipNow()                          { panic(noFuzz) }
func (f *F) Log(args ...any)                   { panic(noFuzz) }
func (f *F) Logf(format string, args ...any)   { panic(noFuzz) }
func (f *F) Error(args ...any)                 { panic(noFuzz) }
func (f *F) Errorf(format string, args ...any) { panic(noFuzz) }
func (f *F) Fatal(args ...any)                 { panic(noFuzz) }
func (f *F) Fatalf(format string, args ...any) { panic(noFuzz) }
func (f *F) Skip(args ...any)                  { panic(noFuzz) }
func (f *F) Skipf(format string, args ...any)  { panic(noFuzz) }
func (f *F) Add(args ...any)                   { panic(noFuzz) }
func (f *F) Fuzz(body any)                     { panic(noFuzz) }

// M holds the state of a test binary. Stage 2 rejects a TestMain function,
// thus no code reaches the method.
type M struct{}

func (m *M) Run() int { panic("gart rejects a TestMain function") }
