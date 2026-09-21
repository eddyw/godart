// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dart

// Package testing gives the subset of the standard testing package that the
// gart compiler supports.
//
// The compiler writes the entry point of a test binary, and that entry point
// calls MainWith. The output holds the format of `go test`, without the source
// position of a log line. Read docs/parity.md for each gap.
package testing

import (
	"context"
	"flag"
	"fmt"
	"internal/linkhook"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

// InternalExample names 1 example function, with the output that the run must
// give. An empty output marks an example that the harness only compiles.
type InternalExample struct {
	Name string
	F    func()
	// Output holds the text that the example must write.
	Output string
	// Unordered reports whether the lines of the output can stand in any
	// order. The comparison then sorts the 2 texts.
	Unordered bool
}

// FuzzCorpus holds the compiler snapshot of source corpus files, in filename order.
var FuzzCorpus map[string][]InternalFuzzInput

type InternalFuzzInput struct {
	Name string
	Data string
}

// InternalFuzzTarget names 1 fuzz target. The compiler writes the list.
type InternalFuzzTarget struct {
	Name string
	Fn   func(f *F)
}

// testDeps holds the calls that a generated main gives the harness. Go holds
// an interface of the same name, and it also keeps the name unexported.
//
// Go asks for the profile calls and for the calls that generate new inputs too.
// gart writes no profile, and it generates no input, thus the interface holds
// the match function alone. Read docs/parity.md.
type testDeps interface {
	MatchString(pat, str string) (bool, error)
}

// Init registers the flags of the harness with the package flag.
//
// A TestMain function of Go calls it in front of flag.Parse, thus the parse
// takes each name that starts with -test and reports no unknown flag.
//
// readFlags reads the arguments of the program again, thus each value below
// reaches the run through that path. The 3 flags with a target write the same
// variable that readFlags writes.
func Init() {
	if flagsReady {
		return
	}
	flagsReady = true
	flag.BoolVar(&verbose, "test.v", verbose, "verbose: print additional output")
	flag.BoolVar(&short, "test.short", short, "run smaller test suite to save time")
	flag.BoolVar(&failFast, "test.failfast", false, "do not start new tests after the first test failure")
	flag.StringVar(&runFilter, "test.run", runFilter,
		"run only tests and examples matching `regexp`")
	flag.StringVar(&skipFilter, "test.skip", skipFilter, "exclude tests and examples matching `regexp`")
	flag.String("test.bench", "", "run only benchmarks matching `regexp`")
	flag.String("test.benchtime", "1s", "run each benchmark for duration `d`")
	flag.Int("test.count", 1, "run tests and benchmarks `n` times")
	flag.Int("test.benchcount", 0, "the fixed count of iterations of 1 benchmark")
	flag.String("test.coverprofile", "", "write a coverage profile to `file`")
}

// flagsReady marks a call of Init. Go also holds such a flag, because a second
// call of flag.BoolVar with 1 name panics.
var flagsReady bool

// MainStart builds the state of a test binary. A generated main of Go calls
// it, and the entry point of gart calls NewM.
func MainStart(deps testDeps, tests []InternalTest, benchmarks []InternalBenchmark,
	fuzzTargets []InternalFuzzTarget, examples []InternalExample) *M {
	if deps != nil && MatchString == nil {
		MatchString = deps.MatchString
	}
	m := NewM(tests, benchmarks)
	m.examples = examples
	m.fuzzTargets = fuzzTargets
	return m
}

// RunTests runs each test of the list and reports whether each one passed.
//
// Go marks the function as deprecated, because a generated main calls
// MainStart. The name stands here for a program that reads the old form.
func RunTests(matchString func(pat, str string) (bool, error), tests []InternalTest) bool {
	if MatchString == nil {
		MatchString = matchString
	}
	ran, ok := runList(tests, readFlags())
	if !ran {
		emit(NoTestsToRun)
	}
	return ok
}

// RunBenchmarks runs each benchmark of the list. Go marks the function as
// deprecated.
func RunBenchmarks(matchString func(pat, str string) (bool, error), benchmarks []InternalBenchmark) {
	if MatchString == nil {
		MatchString = matchString
	}
	flags := readFlags()
	flags.benchmarks = true
	for _, bc := range benchmarks {
		if !matches(bc.Name, flags.bench) || skips(bc.Name) {
			continue
		}
		runBenchmark(bc.Name, bc.F, flags)
	}
}

// RunExamples runs each example of the list and reports whether each one
// passed. Go marks the function as deprecated.
//
// The harness holds the text of standard output while the example runs, and it
// compares that text with the output of the example.
func RunExamples(matchString func(pat, str string) (bool, error), examples []InternalExample) bool {
	if MatchString == nil {
		MatchString = matchString
	}
	ok := true
	for _, ex := range examples {
		if !matches(ex.Name, runFilter) {
			continue
		}
		if !runExample(ex) {
			ok = false
		}
	}
	return ok
}

// runExample runs 1 example and reports whether its output matches.
func runExample(ex InternalExample) bool {
	if verbose {
		emit("=== RUN   " + ex.Name)
	}
	start := dart.MonoNanos()
	text := captured(ex.F)
	elapsed := seconds(dart.MonoNanos() - start)
	got := strings.TrimSpace(text)
	want := strings.TrimSpace(ex.Output)
	fail := ""
	if ex.Unordered {
		if sortedLines(got) != sortedLines(want) {
			fail = "got:\n" + text + "\nwant (unordered):\n" + ex.Output + "\n"
		}
	} else if got != want {
		fail = "got:\n" + got + "\nwant:\n" + want + "\n"
	}
	if fail != "" {
		fmt.Print("--- FAIL: " + ex.Name + " (" + elapsed + "s)\n" + fail)
		return false
	}
	if verbose {
		emit("--- PASS: " + ex.Name + " (" + elapsed + "s)")
	}
	return true
}

// captured runs a function and gives the text that it wrote to standard
// output. A panic of the function still gives the text back to the terminal.
func captured(f func()) (text string) {
	dart.CaptureOut(true)
	defer func() { text = dart.CaptureOut(false) }()
	f()
	return ""
}

// sortedLines sorts the lines of a text. An unordered example reads it.
func sortedLines(text string) string {
	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		for k := i; k > 0 && lines[k] < lines[k-1]; k-- {
			lines[k], lines[k-1] = lines[k-1], lines[k]
		}
	}
	return strings.Join(lines, "\n")
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

// short holds the state of the -short flag of gart test. readFlags writes it.
var short bool

// Short reports whether the build asked for the short form of a test.
func Short() bool { return short }

// current holds the test of each goroutine that runs a test. AllocsPerRun
// reads it, because the function takes no TB.
//
// A parallel test runs on a goroutine of its own, thus 1 variable cannot hold
// the answer. The number of the goroutine is the key.
var current = map[int]*T{}

// currentTest gives the test that the goroutine of the caller runs.
func currentTest() *T { return current[dart.GoID()] }

// AllocsPerRun skips the test that runs now.
//
// The Dart VM gives no count of allocations for 1 operation. A result of 0
// would make an allocation test pass, thus the call reports a skip with the
// cause.
func AllocsPerRun(runs int, f func()) float64 {
	t := currentTest()
	if t == nil {
		panic("testing.AllocsPerRun runs outside a test")
	}
	t.Skipf("gart counts no allocation, thus AllocsPerRun skips the test")
	return 0
}

// T holds the state of 1 test.
type T struct {
	names      *subtestNames
	name       string
	level      int
	failed     bool
	skipped    bool
	logs       []string
	cleanups   []func()
	tempDirMu  sync.Mutex
	tempDir    string
	tempDirSeq int
	ctx        context.Context
	cancel     context.CancelFunc
	// artifactDir holds the directory of ArtifactDir. Each call of 1 test
	// gives the same directory, thus the name stands here.
	artifactDir string
	// out holds the writer of Output. It keeps the part of a line that no
	// newline closed yet.
	out *outputWriter
	// report holds the result line of each subtest, and of the subtests of a
	// subtest. Go writes them after the line of the parent.
	report []string
	// start holds the time of the start of the test, in nanoseconds. Parallel
	// writes it again, thus the pause holds no time.
	start int64
	// stopping marks the unwind of a test that FailNow or SkipNow stopped. A
	// deferred log line of that unwind holds no position, because Go holds the
	// position of its own runtime there.
	stopping bool
	// gate carries the state of the test to the runner of its level: 1 for a
	// pause of Parallel, and 2 for the end.
	gate chan int
	// resume wakes a test that Parallel paused. The runner of the level writes
	// it after each test of the level started.
	resume chan int
	// parallel marks a test that called Parallel.
	parallel bool
	// parent names the test that started this one. Setenv, Chdir, and
	// cryptotest.SetGlobalRandom each reach the whole process, thus each 1
	// rejects a test whose chain of parents holds a parallel test.
	parent *T
	// synctest marks the T of the body of a synctest bubble. Such a test reads
	// a fake clock and it holds no result line of its own, thus Deadline,
	// Parallel, and Run each panic on it.
	synctest bool
	// group runs the subtests of this test. Read the type group.
	group *group
}

const parallelConflict = `testing: test using t.Setenv, t.Chdir, or cryptotest.SetGlobalRandom can not use t.Parallel`

func init() {
	linkhook.CheckParallel = func(v any) { v.(*T).checkParallel() }
}

// checkParallel panics when this test or an ancestor of it called Parallel.
// Such a test runs at the same time as another test, and each caller of this
// check writes a value that the whole process reads.
func (t *T) checkParallel() {
	for c := t; c != nil; c = c.parent {
		if c.parallel {
			panic(parallelConflict)
		}
	}
}

func (t *T) Name() string { return t.name }

func (t *T) Failed() bool { return t.failed }

func (t *T) Skipped() bool { return t.skipped }

// Helper marks a helper function. A log line of a helper holds the position of
// the caller of the helper.
func (t *T) Helper() { markHelper() }

// Parallel marks the test as a test that runs at the same time as each other
// parallel test of its level.
//
// The call stops the test and gives control back to the runner. The runner
// then starts each test of the level that is not parallel. After that pass, it
// wakes each parallel test, and the tests run at the same time on the 1
// isolate of gart. A test that waits for input or for output thus lets another
// test run.
func (t *T) Parallel() {
	if t.parallel {
		panic("testing: t.Parallel called multiple times")
	}
	if t.synctest {
		panic("testing: t.Parallel called inside synctest bubble")
	}
	if t.gate == nil {
		// A benchmark holds a T that no group started, thus no runner waits.
		return
	}
	t.parallel = true
	t.resume = make(chan int)
	if verbose {
		emit("=== PAUSE " + t.name)
	}
	t.gate <- 1
	<-t.resume
	current[dart.GoID()] = t
	if verbose {
		emit("=== CONT  " + t.name)
	}
	// Go drops the time of the pause, thus the line of the result holds the
	// time of the work alone.
	t.start = dart.MonoNanos()
}

// group runs the tests of 1 level. The runner of a test binary holds 1 group
// for the tests of the binary, and each test holds 1 group for its subtests.
type group struct {
	// parked holds each test of the level that Parallel stopped.
	parked []*T
	// done takes each parallel test that ended. The order of the reads is the
	// order of the ends, exactly as go test writes it.
	done chan *T
}

// run starts 1 test on a goroutine of its own and waits for the pause or for
// the end of it.
func (g *group) run(t *T, call func(*T)) {
	if g.done == nil {
		g.done = make(chan *T)
	}
	t.gate = make(chan int)
	go func() {
		tRunner(t, call)
		if t.parallel {
			g.done <- t
			return
		}
		t.gate <- 2
	}()
	if <-t.gate == 1 {
		g.parked = append(g.parked, t)
	}
}

// wait wakes each test that Parallel stopped, and it gives each one in the
// order of its end.
func (g *group) wait() []*T {
	list := g.parked
	g.parked = nil
	for _, t := range list {
		t.resume <- 1
	}
	out := make([]*T, 0, len(list))
	for range list {
		out = append(out, <-g.done)
	}
	return out
}

func (t *T) Cleanup(f func()) { t.cleanups = append(t.cleanups, f) }

func (t *T) runCleanups() {
	for len(t.cleanups) != 0 {
		i := len(t.cleanups) - 1
		cleanup := t.cleanups[i]
		t.cleanups[i] = nil
		t.cleanups = t.cleanups[:i]
		dart.Catch(cleanup)
	}
}

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
	t.checkParallel()
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
func (t *T) Chdir(dir string) {
	t.checkParallel()
	oldPath, cause := os.Getwd()
	if cause != nil {
		t.Fatal(cause)
	}
	old, cause := os.Open(oldPath)
	if cause != nil {
		t.Fatal(cause)
	}
	t.Cleanup(func() {
		defer old.Close()
		if cause := old.Chdir(); cause != nil {
			panic("testing.Chdir: " + cause.Error())
		}
	})
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
}

// Deadline gives the time at which the binary passes the timeout of the run.
//
// gart holds no flag for a timeout, thus the flag of the result is always
// false. Go reports the same for a run with the timeout 0.
func (t *T) Deadline() (deadline time.Time, ok bool) {
	if t.synctest {
		// A test of a bubble reads a fake clock, thus a deadline of the real
		// clock holds no meaning for it.
		panic("testing: t.Deadline called inside synctest bubble")
	}
	return time.Time{}, false
}

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
	t.stopping = true
	dart.Stop()
}

func (t *T) SkipNow() {
	t.skipped = true
	t.stopping = true
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
	if t.synctest {
		panic("testing: t.Run called inside synctest bubble")
	}
	return t.runSub(subName(name), f)
}

// runSub runs a subtest under the exact name. Run maps a space of the name to
// an underscore first, and the seed corpus of a fuzz target keeps the name of
// its file, exactly as go test keeps it.
func (t *T) runSub(name string, f func(*T)) bool {
	if shouldFailFast() {
		return true
	}
	if t.names == nil {
		t.names = &subtestNames{subNames: make(map[string]int32)}
	}
	sub := &T{name: t.names.unique(t.name, name), names: t.names, level: t.level + 1, parent: t}
	if !matches(sub.name, runFilter) || skips(sub.name) {
		return true
	}
	if t.group == nil {
		t.group = &group{}
	}
	t.group.run(sub, f)
	if sub.parallel {
		// Go reports true for a subtest that stands in the parallel pass,
		// because the body of it did not fail yet.
		return true
	}
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
func logPos(stopping bool) string {
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
	if pos := logPos(t.stopping); pos != "" {
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
	tests       []InternalTest
	benchmarks  []InternalBenchmark
	examples    []InternalExample
	fuzzTargets []InternalFuzzTarget
	// code holds the exit status after Run.
	code int
	// ran marks a call of Run.
	ran bool
}

// NewM builds the state of a test binary. The compiler writes the call into the
// entry point.
func NewM(tests []InternalTest, benchmarks []InternalBenchmark) *M {
	Init()
	testBinary = true
	return &M{tests: tests, benchmarks: benchmarks}
}

// Run runs each test and each benchmark, and it gives the exit status. A status
// of 0 means that each one passed.
func (m *M) Run() int {
	m.ran = true
	if !flag.Parsed() {
		flag.Parse()
	}
	flags := readFlags()
	ran, ok := runList(m.tests, flags)
	failed := !ok
	// Go runs each fuzz target after the tests and in front of the examples.
	if fuzzRan, fuzzOk := runFuzzTargets(m.fuzzTargets, flags); fuzzRan {
		ran = true
		failed = failed || !fuzzOk
	}
	// Go runs each example after the tests and before the benchmarks.
	for _, ex := range m.examples {
		if !matches(ex.Name, flags.run) || skips(ex.Name) {
			continue
		}
		ran = true
		if !runExample(ex) {
			failed = true
		}
	}
	if flags.benchmarks {
		for _, bc := range m.benchmarks {
			if !matches(bc.Name, flags.bench) || skips(bc.Name) {
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
		writeCoverage()
		m.code = 1
		return 1
	}
	emit("PASS")
	writeCoverage()
	m.code = 0
	return 0
}

// runList runs each test of a list and reports whether 1 test ran and whether
// each test passed.
//
// The flag -count runs the whole list again, exactly as go test does. A test
// that Parallel stopped waits for the end of the pass of its round.
func runList(tests []InternalTest, flags options) (ran, ok bool) {
	ok = true
	for round := 0; round < flags.rounds; round++ {
		if shouldFailFast() {
			break
		}
		g := &group{}
		for _, tc := range tests {
			if shouldFailFast() {
				break
			}
			if !matches(tc.Name, flags.run) || skips(tc.Name) {
				continue
			}
			ran = true
			t := &T{name: tc.Name}
			f := tc.F
			g.run(t, f)
			if t.parallel {
				continue
			}
			for _, l := range t.report {
				emit(l)
			}
			if t.failed {
				ok = false
			}
		}
		// The parallel pass. Each test that Parallel stopped runs now, and the
		// report of each one follows the order of its end.
		for _, t := range g.wait() {
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

// writeCoverage writes the line of the coverage of a run that holds counters.
// A run with no counter writes no line, exactly as go test does.
func writeCoverage() {
	if CoverMode() == "" {
		return
	}
	if _, total := coverCounts(); total == 0 {
		emit("coverage: [no statements]")
		return
	}
	emit("coverage: " + strconv.FormatFloat(Coverage()*100, 'f', 1, 64) +
		"% of statements")
}

// Main runs a legacy custom test binary and exits with its result.
func Main(matchString func(pat, str string) (bool, error), tests []InternalTest,
	benchmarks []InternalBenchmark, examples []InternalExample) {
	MatchString = matchString
	m := NewM(tests, benchmarks)
	m.examples = examples
	os.Exit(m.Run())
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
func MainWith(tests []InternalTest, benchmarks []InternalBenchmark,
	fuzzTargets []InternalFuzzTarget, examples []InternalExample, f func(*M)) {
	m := NewM(tests, benchmarks)
	m.fuzzTargets = fuzzTargets
	m.examples = examples
	f(m)
	if !m.ran {
		emit("testing: warning: no tests to run")
		return
	}
	Finish(m, m.code)
}

// runFilter holds the pattern of -test.run. Run of T reads it for a subtest.
var runFilter string
var skipFilter string

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
	// rounds is the count of runs of the whole list of the tests. The flag
	// -count of gart test gives it, and the value is 1 by default.
	rounds int
}

// defaultBenchNanos is the length that 1 benchmark targets. Go uses 1 second.
const defaultBenchNanos = 1000000000

// readFlags reads the flags of the arguments of the program, with the rules of
// the flag package of Go. A flag that is not a bool takes its value after an
// equal sign or in the next argument. The flags end at the first argument that
// holds no dash and at a lone "--".
func readFlags() options {
	out := options{nanoseconds: defaultBenchNanos, rounds: 1}
	args := dart.Args()
	for i := 0; i < len(args); i++ {
		name, value, hasValue, ok := splitFlag(args[i])
		if !ok {
			break
		}
		switch name {
		case "test.v":
			verbose = boolValue(value, hasValue)
		case "test.short":
			short = boolValue(value, hasValue)
		case "test.failfast":
			failFast = boolValue(value, hasValue)
		case "test.count", "test.run", "test.skip", "test.bench", "test.benchtime", "test.benchcount":
			if !hasValue {
				if i+1 >= len(args) {
					emit("flag needs an argument: -" + name)
					dart.Exit(2)
				}
				i++
				value = args[i]
			}
			out.set(name, value)
		}
	}
	return out
}

// splitFlag reads the name and the value of 1 argument. The flag package of Go
// accepts 1 dash or 2 in front of the name. ok is false for an argument that
// is no flag.
func splitFlag(arg string) (name, value string, hasValue, ok bool) {
	if len(arg) < 2 || arg[0] != '-' || arg == "--" {
		return "", "", false, false
	}
	arg = arg[1:]
	if arg[0] == '-' {
		arg = arg[1:]
	}
	name, value, hasValue = strings.Cut(arg, "=")
	return name, value, hasValue, true
}

// boolValue reads the value of a bool flag. The bare flag means true.
func boolValue(value string, hasValue bool) bool {
	if !hasValue {
		return true
	}
	v, err := strconv.ParseBool(value)
	return err == nil && v
}

// set writes the value of 1 flag that takes a value.
func (o *options) set(name, value string) {
	switch name {
	case "test.count":
		if v, err := strconv.Atoi(value); err == nil {
			o.rounds = v
		}
	case "test.skip":
		skipFilter = value
	case "test.run":
		o.run = value
		runFilter = value
	case "test.bench":
		o.benchmarks = true
		o.bench = value
	case "test.benchtime":
		if v, err := strconv.Atoi(value); err == nil {
			o.nanoseconds = int64(v)
		}
	case "test.benchcount":
		if v, err := strconv.Atoi(value); err == nil {
			o.count = v
		}
	}
}

// MatchString matches 1 element of a test name against 1 part of the pattern of
// -test.run. The entry point of a test binary gives it, because this package
// imports no regexp: regexp/syntax imports this package in its own test, and
// the 2 imports would give a cycle.
var MatchString func(pattern, name string) (bool, error)

// matches reports whether a test name passes a filter. match.go holds the rule,
// and the driver of the compiler reads the same file.
func skips(name string) bool { return SkipName(name, skipFilter, MatchString) }

func matches(name, pattern string) bool {
	return MatchName(name, pattern, MatchString)
}

// tRunner runs 1 test body. It puts the result line of the test in front of the
// result lines of the subtests, exactly as go test does.
func tRunner(t *T, call func(*T)) {
	if verbose {
		emit("=== RUN   " + t.name)
	}
	t.start = dart.MonoNanos()
	id := dart.GoID()
	parent := current[id]
	current[id] = t
	dart.Catch(func() { call(t) })
	t.stopping = false
	// The parallel subtests of this test run now. Go runs them after the body
	// and in front of the cleanups.
	if t.group != nil {
		for _, sub := range t.group.wait() {
			t.report = append(t.report, sub.report...)
			if sub.failed {
				t.failed = true
			}
		}
	}
	current[id] = t
	t.out.flush()
	if t.cancel != nil {
		t.cancel()
	}
	t.runCleanups()
	current[id] = parent
	if t.failed {
		failedTests = true
	}
	head := fmt.Sprintf("%s--- %s: %s (%ss)", indent(t.level), state(t), t.name,
		seconds(dart.MonoNanos()-t.start))
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
	return BenchmarkResult{N: last.N, T: time.Duration(last.elapsed), Bytes: last.bytes,
		Extra: last.extra}
}

// runIterations runs the body 1 time with a fixed count of iterations.
func runIterations(name string, f func(*B), n int) *B {
	t := &T{name: name}
	b := &B{N: n, common: t, extra: map[string]float64{}}
	id := dart.GoID()
	parent := current[id]
	current[id] = t
	b.StartTimer()
	dart.Catch(func() { f(b) })
	b.StopTimer()
	t.out.flush()
	if t.cancel != nil {
		t.cancel()
	}
	t.runCleanups()
	current[id] = parent
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

// cover holds the counters that SetCover took.
var cover Cover

// RegisterCover does nothing. The function of Go does nothing as well, because
// the coverage of Go moved into the runtime. SetCover holds the counters of a
// run of gart.
func RegisterCover(c Cover) {}

// SetCover holds the counters of the coverage of a run. The entry point of a
// test binary of gart calls it, because the flag -cover of gart test writes a
// counter into each statement of the package under test.
//
// Go holds no such name: its own entry point gives the mode through the
// interface testDeps of MainStart, and that interface names a profile of the
// runtime that gart holds not.
func SetCover(c Cover) { cover = c }

// CoverMode gives the mode of the coverage: "set", "count", or "atomic". A run
// with no coverage gives an empty text.
func CoverMode() string { return cover.Mode }

// Coverage gives the part of the statements that the run reached, from 0 to 1.
// A run with no coverage gives 0.
func Coverage() float64 {
	covered, total := coverCounts()
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total)
}

// coverCounts gives the count of the statements that the run reached and the
// count of the statements of each block.
func coverCounts() (covered, total int) {
	for name, counts := range cover.Counters {
		blocks := cover.Blocks[name]
		for i, count := range counts {
			if i >= len(blocks) {
				break
			}
			total += int(blocks[i].Stmts)
			if count > 0 {
				covered += int(blocks[i].Stmts)
			}
		}
	}
	return covered, total
}

// testBinary marks a run of a test binary. NewM writes it, and the compiler
// writes the call of NewM into the entry point of each test binary.
var testBinary bool

// Testing reports whether the code runs inside a test.
//
// Go writes the answer at the link of the binary, and gart writes it when the
// entry point builds the state of the run. Thus a package initializer of a
// test binary reads false here, and it reads true in Go.
func Testing() bool { return testBinary }

var failFast bool
var failedTests bool

func shouldFailFast() bool { return failFast && failedTests }
