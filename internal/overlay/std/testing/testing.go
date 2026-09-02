//go:build dart

// Package testing gives the subset of the standard testing package that the
// gart compiler supports.
//
// The compiler writes the entry point of a test binary, and that entry point
// calls Main. The output holds the format of `go test`, without the source
// position of a log line. Read docs/parity.md for each gap.
package testing

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall/dart"
	"time"
	"unicode"
)

// InternalTest names 1 test function. The compiler writes the list.
type InternalTest struct {
	Name string
	F    func(*T)
}

// InternalBenchmark names 1 benchmark function. The compiler writes the list.
type InternalBenchmark struct {
	Name string
	F    func(*B)
}

// TB holds the methods that T, B, and F share.
type TB interface {
	ArtifactDir() string
	Attr(key, value string)
	Chdir(dir string)
	Cleanup(f func())
	Context() context.Context
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
	Output() io.Writer
	Setenv(key, value string)
	Skip(args ...any)
	SkipNow()
	Skipf(format string, args ...any)
	Skipped() bool
	TempDir() string
}

var (
	_ TB = (*T)(nil)
	_ TB = (*B)(nil)
	_ TB = (*F)(nil)
)

// verbose holds the state of the -v flag of gart test. The compiler writes the
// value into the call of Main.
var verbose bool

// Verbose reports whether the build asked for 1 line for each test.
func Verbose() bool { return verbose }

// Short reports whether the build asked for the short form of a test. gart has
// no flag for it, thus the result is always false.
func Short() bool { return false }

// current holds the test that runs now. AllocsPerRun reads it, because the
// function takes no TB.
var current *T

// AllocsPerRun skips the test that runs now.
//
// The Dart VM gives no count of allocations for 1 operation. A result of 0
// would make an allocation test pass, thus the call reports a skip with the
// cause.
func AllocsPerRun(runs int, f func()) float64 {
	if current == nil {
		panic("testing.AllocsPerRun runs outside a test")
	}
	current.Skipf("gart counts no allocation, thus AllocsPerRun skips the test")
	return 0
}

// T holds the state of 1 test.
type T struct {
	name     string
	level    int
	failed   bool
	skipped  bool
	logs     []string
	cleanups []func()
	ctx      context.Context
	cancel   context.CancelFunc
	// artifactDir holds the directory of ArtifactDir. Each call of 1 test
	// gives the same directory, thus the name stands here.
	artifactDir string
	// out holds the writer of Output. It keeps the part of a line that no
	// newline closed yet.
	out *outputWriter
	// report holds the result line of each subtest, and of the subtests of a
	// subtest. Go writes them after the line of the parent.
	report []string
}

func (t *T) Name() string { return t.name }

func (t *T) Failed() bool { return t.failed }

func (t *T) Skipped() bool { return t.skipped }

// Helper marks a helper function. A log line of a helper holds the position of
// the caller of the helper.
func (t *T) Helper() { markHelper() }

// Parallel does nothing. All Go code runs in 1 isolate, thus gart runs each
// test in the order of the list.
func (t *T) Parallel() {}

func (t *T) Cleanup(f func()) { t.cleanups = append(t.cleanups, f) }

// Context gives a context that the harness cancels in front of the cleanups of
// the test.
func (t *T) Context() context.Context {
	if t.ctx == nil {
		t.ctx, t.cancel = context.WithCancel(context.Background())
	}
	return t.ctx
}

// Setenv writes 1 variable of the environment for the test. The cleanup writes
// the value of the start again.
//
// The Dart VM gives no write to the environment of the process. The os overlay
// holds a map of its own, thus the write reaches each reader of the program.
func (t *T) Setenv(name, value string) {
	prev, held := os.LookupEnv(name)
	if cause := os.Setenv(name, value); cause != nil {
		t.Fatal("Setenv: " + cause.Error())
	}
	t.Cleanup(func() {
		if held {
			os.Setenv(name, prev)
			return
		}
		os.Unsetenv(name)
	})
}

// tempSeq numbers each temporary directory of the binary.
var tempSeq int

// newTempDir builds a new directory under the temporary directory of the
// system, and it holds a cleanup that removes the directory.
func (t *T) newTempDir(who string) string {
	tempSeq++
	dir := dart.TempDir() + dart.PathSeparator() +
		"gart" + strconv.Itoa(tempSeq) + "_" + strconv.FormatInt(dart.NowNanos(), 10)
	if cause := dart.Mkdir(dir, true); cause != "" {
		t.Fatal(who + ": " + cause)
	}
	t.Cleanup(func() { dart.RemoveAll(dir) })
	return dir
}

// TempDir gives a new directory for the test. The cleanup of the test removes
// it and each file inside it.
func (t *T) TempDir() string { return t.newTempDir("TempDir") }

// ArtifactDir gives the directory that holds each file that the test writes.
// Each call of 1 test gives the same directory.
//
// Go reads the flag -test.artifacts for the place of the directory, and gart
// holds no such flag. Thus the directory is a temporary one, and the cleanup of
// the test removes it. Go does the same when the flag stands away.
func (t *T) ArtifactDir() string {
	if t.artifactDir == "" {
		t.artifactDir = t.newTempDir("ArtifactDir")
	}
	return t.artifactDir
}

// Chdir moves the process into a directory for the test, and the cleanup of the
// test moves it back. The call also writes PWD, exactly as Go does on a system
// of the POSIX form.
//
// Go holds the old directory as an open descriptor, and gart holds the path,
// because the Dart VM gives no descriptor of a directory.
func (t *T) Chdir(dir string) {
	old, cause := os.Getwd()
	if cause != nil {
		t.Fatal(cause)
	}
	if cause := os.Chdir(dir); cause != nil {
		t.Fatal(cause)
	}
	full := dir
	if !filepath.IsAbs(full) {
		now, cause := os.Getwd()
		if cause != nil {
			t.Fatal(cause)
		}
		full = now
	}
	t.Setenv("PWD", full)
	t.Cleanup(func() {
		if cause := os.Chdir(old); cause != nil {
			panic("testing.Chdir: " + cause.Error())
		}
	})
}

// Deadline gives the time at which the binary passes the timeout of the run.
//
// gart holds no flag for a timeout, thus the flag of the result is always
// false. Go reports the same for a run with the timeout 0.
func (t *T) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }

// Attr writes 1 attribute of the test. A verbose run holds the line, and a
// plain run drops it, exactly as go test does.
//
// The key holds no space, and the value holds no newline. A call that breaks
// the rule reports an error against the test.
func (t *T) Attr(key, value string) {
	if strings.ContainsFunc(key, unicode.IsSpace) {
		t.Errorf("disallowed whitespace in attribute key %q", key)
		return
	}
	if strings.ContainsAny(value, "\r\n") {
		t.Errorf("disallowed newline in attribute value %q", value)
		return
	}
	if !verbose {
		return
	}
	emit("=== ATTR  " + t.name + " " + key + " " + value)
}

// Output gives a writer that holds each line in the log of the test. The writer
// adds no source position and no newline.
//
// The writer holds a part of a line until a newline closes it. A call of Log,
// or the end of the test, writes such a part with a newline behind it.
func (t *T) Output() io.Writer {
	if t.out == nil {
		t.out = &outputWriter{t: t}
	}
	return t.out
}

// outputWriter holds the bytes of Output until a newline closes a line.
type outputWriter struct {
	t *T
	// partial holds the bytes behind the last newline.
	partial string
}

func (o *outputWriter) Write(p []byte) (int, error) {
	o.partial += string(p)
	for {
		i := strings.IndexByte(o.partial, '\n')
		if i < 0 {
			break
		}
		line := o.partial[:i]
		o.partial = o.partial[i+1:]
		o.t.writeRaw(line + "\n")
	}
	return len(p), nil
}

// flush writes the part of a line that no newline closed. A writer that holds
// nothing writes nothing.
func (o *outputWriter) flush() {
	if o == nil || o.partial == "" {
		return
	}
	text := o.partial
	o.partial = ""
	o.t.writeRaw(text + "\n")
}

func (t *T) Fail() { t.failed = true }

func (t *T) FailNow() {
	t.failed = true
	stopping = true
	dart.Stop()
}

func (t *T) SkipNow() {
	t.skipped = true
	stopping = true
	dart.Stop()
}

// stopping marks the unwind of a test that FailNow or SkipNow stopped. A
// deferred log line of that unwind holds no position, because Go holds the
// position of its own runtime there.
var stopping bool

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
	if !matches(sub.name, runFilter) {
		return true
	}
	runOne(sub, func() { f(sub) })
	t.report = append(t.report, sub.report...)
	if sub.failed {
		t.failed = true
	}
	return !sub.failed
}

// helpers holds the name of each function that called Helper.
var helpers = map[string]bool{}

// markHelper marks the caller of Helper. The walk drops the frames of this
// package, thus no count of frames stands in the code.
func markHelper() {
	for _, frame := range dart.Stack(0) {
		name, _, ok := strings.Cut(frame, "|")
		if !ok || strings.HasPrefix(name, "testing.") {
			continue
		}
		helpers[name] = true
		return
	}
}

// logPos gives the source position of the call that writes a log line. The walk
// drops the frames of this package and the frame of each helper.
func logPos() string {
	if stopping {
		return ""
	}
	for _, frame := range dart.Stack(0) {
		name, rest, ok := strings.Cut(frame, "|")
		if !ok || strings.HasPrefix(name, "testing.") || helpers[name] {
			continue
		}
		file, line, _ := strings.Cut(rest, "|")
		// go test names the base name of the file, and the map holds the
		// package path in front of it.
		if i := strings.LastIndexByte(file, '/'); i >= 0 {
			file = file[i+1:]
		}
		return file + ":" + line
	}
	return ""
}

// write holds 1 log message with the position of the caller in front of it. A
// part of a line that Output holds goes out first, exactly as Go does.
func (t *T) write(text string) {
	t.out.flush()
	if pos := logPos(); pos != "" {
		text = pos + ": " + text
	}
	t.writeRaw(text)
}

// writeRaw holds 1 log message as it stands. Verbose mode writes it at once,
// and the plain mode writes it only when the test fails.
func (t *T) writeRaw(text string) {
	if !verbose {
		t.logs = append(t.logs, text)
		return
	}
	for _, l := range indentLines(1, text) {
		emit(l)
	}
}

// M holds the state of a test binary. A TestMain function reads it.
type M struct {
	tests      []InternalTest
	benchmarks []InternalBenchmark
	// code holds the exit status after Run.
	code int
	// ran marks a call of Run.
	ran bool
}

// NewM builds the state of a test binary. The compiler writes the call into the
// entry point.
func NewM(tests []InternalTest, benchmarks []InternalBenchmark) *M {
	testBinary = true
	return &M{tests: tests, benchmarks: benchmarks}
}

// Run runs each test and each benchmark, and it gives the exit status. A status
// of 0 means that each one passed.
func (m *M) Run() int {
	m.ran = true
	flags := readFlags()
	failed := false
	ran := false
	for _, tc := range m.tests {
		if !matches(tc.Name, flags.run) {
			continue
		}
		ran = true
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
	if flags.benchmarks {
		for _, bc := range m.benchmarks {
			if !matches(bc.Name, flags.bench) {
				continue
			}
			if !runBenchmark(bc.Name, bc.F, flags) {
				failed = true
			}
		}
	}
	if !ran && !flags.benchmarks {
		emit(NoTestsToRun)
	}
	if failed {
		emit("FAIL")
		m.code = 1
		return 1
	}
	emit("PASS")
	m.code = 0
	return 0
}

// Main runs each test and each benchmark of a test binary. The compiler writes
// the call into the entry point of a binary with no TestMain function.
//
// The flags come from the arguments of the program. gart test writes -test.v
// for a line for each test, -test.run=<pattern> for a filter of the tests,
// -test.bench=<pattern> for a filter of the benchmarks, and one of
// -test.benchtime=<nanoseconds> or -test.benchcount=<count> for the length of
// a benchmark.
func Main(tests []InternalTest, benchmarks []InternalBenchmark) {
	m := NewM(tests, benchmarks)
	Finish(m, m.Run())
}

// Finish stops the binary with the status of a run. The compiler writes the call
// after a TestMain function.
//
// Go passes the result of m.Run to os.Exit when TestMain returns. A TestMain
// function that never calls m.Run gives the status 0.
func Finish(m *M, code int) {
	if code != 0 {
		dart.Exit(code)
	}
}

// MainWith runs a TestMain function and then stops the binary. The compiler
// writes the call into the entry point of a binary with such a function.
func MainWith(tests []InternalTest, benchmarks []InternalBenchmark, f func(*M)) {
	m := NewM(tests, benchmarks)
	f(m)
	if !m.ran {
		emit("testing: warning: no tests to run")
		return
	}
	Finish(m, m.code)
}

// runFilter holds the pattern of -test.run. Run of T reads it for a subtest.
var runFilter string

// options holds the flags of a run.
type options struct {
	run string
	// benchmarks reports whether the arguments held -test.bench.
	benchmarks bool
	bench      string
	// nanoseconds is the length that 1 benchmark targets.
	nanoseconds int64
	// count fixes the count of iterations. It replaces the search when it is
	// above 0.
	count int
}

// defaultBenchNanos is the length that 1 benchmark targets. Go uses 1 second.
const defaultBenchNanos = 1000000000

func readFlags() options {
	out := options{nanoseconds: defaultBenchNanos}
	for _, arg := range dart.Args() {
		switch {
		case arg == "-test.v" || arg == "-test.v=true":
			verbose = true
		case strings.HasPrefix(arg, "-test.run="):
			out.run = strings.TrimPrefix(arg, "-test.run=")
			runFilter = out.run
		case strings.HasPrefix(arg, "-test.bench="):
			out.benchmarks = true
			out.bench = strings.TrimPrefix(arg, "-test.bench=")
		case strings.HasPrefix(arg, "-test.benchtime="):
			if v, err := strconv.Atoi(strings.TrimPrefix(arg, "-test.benchtime=")); err == nil {
				out.nanoseconds = int64(v)
			}
		case strings.HasPrefix(arg, "-test.benchcount="):
			if v, err := strconv.Atoi(strings.TrimPrefix(arg, "-test.benchcount=")); err == nil {
				out.count = v
			}
		}
	}
	return out
}

// MatchString matches 1 element of a test name against 1 part of the pattern of
// -test.run. The entry point of a test binary gives it, because this package
// imports no regexp: regexp/syntax imports this package in its own test, and
// the 2 imports would give a cycle.
var MatchString func(pattern, name string) (bool, error)

// matches reports whether a test name passes a filter. match.go holds the rule,
// and the driver of the compiler reads the same file.
func matches(name, pattern string) bool {
	return MatchName(name, pattern, MatchString)
}

// runOne runs 1 test body. It puts the result line of the test in front of the
// result lines of the subtests, exactly as go test does.
func runOne(t *T, call func()) {
	if verbose {
		emit("=== RUN   " + t.name)
	}
	start := dart.MonoNanos()
	parent := current
	current = t
	dart.Catch(call)
	stopping = false
	t.out.flush()
	if t.cancel != nil {
		t.cancel()
	}
	for i := len(t.cleanups) - 1; i >= 0; i-- {
		dart.Catch(t.cleanups[i])
	}
	current = parent
	head := fmt.Sprintf("%s--- %s: %s (%ss)", indent(t.level), state(t), t.name,
		seconds(dart.MonoNanos()-start))
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
// The loop of a benchmark reads N. The harness runs the body again with a
// larger N until the run time reaches the target.
type B struct {
	// N is the count of iterations that the body must run.
	N int

	common *T
	// start holds the time of the last start of the timer.
	start int64
	// elapsed holds the measured time of the run, in nanoseconds.
	elapsed int64
	running bool
	// sub marks a benchmark that called Run. Such a benchmark reports no line
	// of its own, exactly as go test does.
	sub bool
	// bytes is the count of bytes that 1 iteration reads or writes. SetBytes
	// gives it, and the report adds a rate.
	bytes int64
	// metrics holds each value that ReportMetric gave, with its unit.
	metrics []string
	// extra holds the same values by unit. BenchmarkResult reads them.
	extra map[string]float64
	// parallelism holds the count that SetParallelism gave. gart runs no
	// parallel benchmark, thus no run reads it.
	parallelism int
	// loopN counts the iterations that Loop gave. A value above 0 marks a body
	// that reads Loop, and Loop runs the search itself.
	loopN int
}

func (b *B) Name() string  { return b.common.Name() }
func (b *B) Failed() bool  { return b.common.Failed() }
func (b *B) Skipped() bool { return b.common.Skipped() }

func (b *B) Helper() { markHelper() }

func (b *B) Cleanup(f func())                  { b.common.Cleanup(f) }
func (b *B) Fail()                             { b.common.Fail() }
func (b *B) FailNow()                          { b.common.FailNow() }
func (b *B) SkipNow()                          { b.common.SkipNow() }
func (b *B) Log(args ...any)                   { b.common.Log(args...) }
func (b *B) Logf(format string, args ...any)   { b.common.Logf(format, args...) }
func (b *B) Error(args ...any)                 { b.common.Error(args...) }
func (b *B) Errorf(format string, args ...any) { b.common.Errorf(format, args...) }
func (b *B) Fatal(args ...any)                 { b.common.Fatal(args...) }
func (b *B) Fatalf(format string, args ...any) { b.common.Fatalf(format, args...) }
func (b *B) Skip(args ...any)                  { b.common.Skip(args...) }
func (b *B) Skipf(format string, args ...any)  { b.common.Skipf(format, args...) }
func (b *B) SetBytes(n int64)                  { b.bytes = n }

func (b *B) ArtifactDir() string      { return b.common.ArtifactDir() }
func (b *B) Attr(key, value string)   { b.common.Attr(key, value) }
func (b *B) Chdir(dir string)         { b.common.Chdir(dir) }
func (b *B) Context() context.Context { return b.common.Context() }
func (b *B) Output() io.Writer        { return b.common.Output() }
func (b *B) Setenv(key, value string) { b.common.Setenv(key, value) }
func (b *B) TempDir() string          { return b.common.TempDir() }

// Elapsed gives the measured time of the run up to this point. A timer that
// runs now holds the time of the current stretch as well.
func (b *B) Elapsed() time.Duration { return time.Duration(b.measured()) }

// SetParallelism holds the count of workers that RunParallel would use.
//
// gart runs no parallel benchmark, because all Go code runs in 1 isolate. Thus
// the count reaches no run. Go states the same for a benchmark that calls no
// RunParallel: the call has no effect.
func (b *B) SetParallelism(p int) {
	if p >= 1 {
		b.parallelism = p
	}
}

func (b *B) ReportMetric(value float64, unit string) {
	b.metrics = append(b.metrics, strconv.FormatFloat(value, 'f', 4, 64)+" "+unit)
	if b.extra == nil {
		b.extra = map[string]float64{}
	}
	b.extra[unit] = value
}

// ReportAllocs does nothing. The Dart VM gives no count of allocations for 1
// operation, thus gart writes no allocation column.
func (b *B) ReportAllocs() {}

// StartTimer starts the measurement. The setup of a benchmark stops the timer
// first, thus the setup stays out of the result.
func (b *B) StartTimer() {
	if b.running {
		return
	}
	b.start = dart.MonoNanos()
	b.running = true
}

// StopTimer stops the measurement.
func (b *B) StopTimer() {
	if !b.running {
		return
	}
	b.elapsed += dart.MonoNanos() - b.start
	b.running = false
}

// ResetTimer drops the measured time. The timer keeps its state.
func (b *B) ResetTimer() {
	if b.running {
		b.start = dart.MonoNanos()
	}
	b.elapsed = 0
}

// Loop reports whether the body must run 1 more iteration.
//
// The harness calls a body that reads Loop only 1 time. Loop runs the search
// itself: it raises N until the run time reaches the target, and it stops the
// timer at the end. Thus N holds the count of iterations of the whole run.
func (b *B) Loop() bool {
	if b.loopN > 0 && b.loopN < b.N {
		b.loopN++
		return true
	}
	return b.loopNext()
}

// loopNext raises the count of iterations, or it ends the run.
func (b *B) loopNext() bool {
	if b.loopN == 0 {
		b.loopN = 1
		b.N = 1
		b.ResetTimer()
		return true
	}
	if benchFlags.count > 0 {
		if b.N < benchFlags.count {
			b.N = benchFlags.count
			b.loopN++
			return true
		}
		b.StopTimer()
		return false
	}
	elapsed := b.measured()
	if elapsed >= benchFlags.nanoseconds || b.N >= 1000000000 {
		b.StopTimer()
		return false
	}
	b.N = nextCount(b.N, elapsed, benchFlags.nanoseconds)
	b.loopN++
	return true
}

// measured gives the time of the run up to this point, in nanoseconds. The
// timer of a run of Loop still runs, thus Loop cannot read elapsed alone.
func (b *B) measured() int64 {
	if b.running {
		return b.elapsed + dart.MonoNanos() - b.start
	}
	return b.elapsed
}

// RunParallel stops the benchmark with an error.
//
// The method measures the throughput of a parallel run, and all Go code runs in
// 1 isolate. A serial run would give a number that says something else, thus
// gart rejects the call.
func (b *B) RunParallel(body func(*PB)) {
	b.Fatalf("gart runs no parallel benchmark, because all Go code runs in 1 isolate")
}

// PB drives a parallel benchmark. gart runs no parallel benchmark.
type PB struct{}

func (p *PB) Next() bool { panic("gart runs no parallel benchmark") }

// benchFlags holds the flags of the run. Run reads them for a subbenchmark.
var benchFlags = options{nanoseconds: defaultBenchNanos}

// Run runs a subbenchmark and reports whether it passed.
func (b *B) Run(name string, f func(*B)) bool {
	b.sub = true
	full := b.common.name + "/" + subName(name)
	if !matches(full, benchFlags.bench) {
		return true
	}
	return runBenchmark(full, f, benchFlags)
}

// runBenchmark measures 1 benchmark and writes its line. It reports whether the
// benchmark passed.
//
// The search follows go test: the first run takes 1 iteration, and each next
// count comes from the time of the last run. The count grows by a factor of 100
// at most, and it stops at 1000 million.
func runBenchmark(name string, f func(*B), flags options) bool {
	benchFlags = flags
	if verbose {
		emit("=== RUN   " + name)
	}
	n := 1
	if flags.count > 0 {
		n = flags.count
	}
	last := runIterations(name, f, n)
	for flags.count == 0 {
		if last.loopN > 0 {
			break
		}
		if last.common.failed || last.common.skipped || last.sub {
			break
		}
		if last.elapsed >= flags.nanoseconds || n >= 1000000000 {
			break
		}
		n = nextCount(n, last.elapsed, flags.nanoseconds)
		last = runIterations(name, f, n)
	}
	for _, l := range last.common.report {
		emit(l)
	}
	if last.common.failed {
		return false
	}
	if last.sub || last.common.skipped {
		return true
	}
	emit(benchLine(name, last))
	return true
}

// BenchmarkResult holds the outcome of 1 run of Benchmark.
type BenchmarkResult struct {
	// N is the count of iterations.
	N int
	// T is the measured time of the run, in nanoseconds. Go holds a
	// time.Duration, and docs/parity.md names the difference.
	T int64
	// Bytes is the count of bytes that 1 iteration reads or writes.
	Bytes int64
	// MemAllocs is the count of allocations of the run. The Dart VM gives no
	// count of allocations, thus a run of the harness holds 0.
	MemAllocs uint64
	// MemBytes is the count of allocated bytes of the run. The Dart VM gives
	// no count of allocations, thus a run of the harness holds 0.
	MemBytes uint64
	// Extra holds each value that ReportMetric gave, by unit. A value here
	// replaces the built-in value of the same unit.
	Extra map[string]float64
}

// NsPerOp gives the time of 1 iteration, in nanoseconds.
func (r BenchmarkResult) NsPerOp() int64 {
	if v, held := r.Extra["ns/op"]; held {
		return int64(v)
	}
	if r.N <= 0 {
		return 0
	}
	return r.T / int64(r.N)
}

// AllocsPerOp gives the count of allocations of 1 iteration. The Dart VM gives
// no count of allocations, thus a run of the harness gives 0.
func (r BenchmarkResult) AllocsPerOp() int64 {
	if v, held := r.Extra["allocs/op"]; held {
		return int64(v)
	}
	if r.N <= 0 {
		return 0
	}
	return int64(r.MemAllocs) / int64(r.N)
}

// AllocedBytesPerOp gives the count of allocated bytes of 1 iteration. The Dart
// VM gives no count of allocations, thus a run of the harness gives 0.
func (r BenchmarkResult) AllocedBytesPerOp() int64 {
	if v, held := r.Extra["B/op"]; held {
		return int64(v)
	}
	if r.N <= 0 {
		return 0
	}
	return int64(r.MemBytes) / int64(r.N)
}

// MemString gives the memory numbers in the format of go test.
func (r BenchmarkResult) MemString() string {
	return fmt.Sprintf("%8d B/op\t%8d allocs/op",
		r.AllocedBytesPerOp(), r.AllocsPerOp())
}

// String gives the line of the result, in the format of go test.
func (r BenchmarkResult) String() string {
	return strconv.Itoa(r.N) + "\t" + strconv.FormatInt(r.NsPerOp(), 10) + " ns/op"
}

// Benchmark runs 1 benchmark function and gives its result. It writes no line,
// thus a caller reads the numbers itself.
func Benchmark(f func(*B)) BenchmarkResult {
	n := 1
	last := runIterations("", f, n)
	for {
		if last.loopN > 0 || last.common.failed || last.common.skipped || last.sub {
			break
		}
		if last.elapsed >= defaultBenchNanos || n >= 1000000000 {
			break
		}
		n = nextCount(n, last.elapsed, defaultBenchNanos)
		last = runIterations("", f, n)
	}
	if last.common.failed {
		return BenchmarkResult{}
	}
	return BenchmarkResult{N: last.N, T: last.elapsed, Bytes: last.bytes,
		Extra: last.extra}
}

// runIterations runs the body 1 time with a fixed count of iterations.
func runIterations(name string, f func(*B), n int) *B {
	t := &T{name: name}
	b := &B{N: n, common: t, extra: map[string]float64{}}
	parent := current
	current = t
	b.StartTimer()
	dart.Catch(func() { f(b) })
	b.StopTimer()
	t.out.flush()
	if t.cancel != nil {
		t.cancel()
	}
	for i := len(t.cleanups) - 1; i >= 0; i-- {
		dart.Catch(t.cleanups[i])
	}
	current = parent
	if t.failed || t.skipped {
		head := indent(t.level) + "--- " + state(t) + ": " + name + " (" +
			seconds(b.elapsed) + "s)"
		out := []string{head}
		for _, text := range t.logs {
			out = append(out, indentLines(t.level+1, text)...)
		}
		t.report = append(out, t.report...)
	}
	return b
}

// nextCount gives the count of iterations of the next run. It reads the time of
// the last run, exactly as go test does.
func nextCount(last int, elapsed, target int64) int {
	if elapsed <= 0 {
		elapsed = 1
	}
	next := target * int64(last) / elapsed
	next += next / 5
	if limit := int64(last) * 100; next > limit {
		next = limit
	}
	if next <= int64(last) {
		next = int64(last) + 1
	}
	if next > 1000000000 {
		next = 1000000000
	}
	return roundUp(int(next))
}

// roundUp raises a count to the next value of the form 1eX, 2eX, 3eX, or 5eX.
// go test reports such a count, thus a run is easy to compare.
func roundUp(n int) int {
	base := 1
	for base*10 < n {
		base *= 10
	}
	switch {
	case n <= base:
		return base
	case n <= 2*base:
		return 2 * base
	case n <= 3*base:
		return 3 * base
	case n <= 5*base:
		return 5 * base
	}
	return 10 * base
}

// benchLine writes the result line of 1 benchmark. The format is the format of
// go test, thus benchstat reads the output.
//
// The suffix of the name is the count of isolates. All Go code runs in 1
// isolate, thus the suffix is always 1.
func benchLine(name string, b *B) string {
	perOp := float64(b.elapsed) / float64(b.N)
	line := name + "-1\t" + strconv.Itoa(b.N) + "\t" + amount(perOp) + " ns/op"
	if b.bytes > 0 && b.elapsed > 0 {
		rate := float64(b.bytes) * float64(b.N) / 1000000.0 /
			(float64(b.elapsed) / 1000000000.0)
		line += "\t" + amount(rate) + " MB/s"
	}
	for _, m := range b.metrics {
		line += "\t" + m
	}
	return line
}

// amount writes a measured value with about 4 digits that carry meaning.
func amount(v float64) string {
	switch {
	case v >= 100:
		return strconv.FormatFloat(v, 'f', 0, 64)
	case v >= 10:
		return strconv.FormatFloat(v, 'f', 1, 64)
	case v >= 1:
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
	return strconv.FormatFloat(v, 'f', 4, 64)
}

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
func (f *F) ArtifactDir() string               { panic(noFuzz) }
func (f *F) Attr(key, value string)            { panic(noFuzz) }
func (f *F) Chdir(dir string)                  { panic(noFuzz) }
func (f *F) Context() context.Context          { panic(noFuzz) }
func (f *F) Output() io.Writer                 { panic(noFuzz) }
func (f *F) Setenv(key, value string)          { panic(noFuzz) }
func (f *F) TempDir() string                   { panic(noFuzz) }

// CoverBlock holds the coverage of 1 basic block. The numbers count from 1,
// exactly as an editor counts them, and a column counts bytes.
type CoverBlock struct {
	// Line0 is the line of the start of the block.
	Line0 uint32
	// Col0 is the column of the start of the block.
	Col0 uint16
	// Line1 is the line of the end of the block.
	Line1 uint32
	// Col1 is the column of the end of the block.
	Col1 uint16
	// Stmts is the count of statements of the block.
	Stmts uint16
}

// Cover holds the coverage data of a run.
type Cover struct {
	Mode            string
	Counters        map[string][]uint32
	Blocks          map[string][]CoverBlock
	CoveredPackages string
}

// RegisterCover holds the counters of the coverage of the tests. The function
// of Go does nothing as well, because the report of Go reads the counters of
// the runtime.
func RegisterCover(c Cover) {}

// CoverMode gives the mode of the coverage: "set", "count", or "atomic".
//
// gart writes no counter into a package, thus the result is always empty. Go
// gives the same result for a run with no coverage.
func CoverMode() string { return "" }

// Coverage gives the part of the statements that the run reached, from 0 to 1.
//
// gart writes no counter into a package, thus the result is always 0. Go gives
// the same result for a run with no coverage.
func Coverage() float64 { return 0 }

// testBinary marks a run of a test binary. NewM writes it, and the compiler
// writes the call of NewM into the entry point of each test binary.
var testBinary bool

// Testing reports whether the code runs inside a test.
//
// Go writes the answer at the link of the binary, and gart writes it when the
// entry point builds the state of the run. Thus a package initializer of a
// test binary reads false here, and it reads true in Go.
func Testing() bool { return testBinary }
