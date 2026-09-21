// Command testingextra proves the names that the testing overlay gained: the
// process methods of T and of B, the memory fields of BenchmarkResult, and the
// coverage names of the package.
//
// A plain program holds no test harness. Thus the program drives a T that no
// harness owns, exactly as testdata/programs/httptestextra does, and it drives
// a B through testing.Benchmark, which builds a whole run.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func main() {
	tRun()
	benchRun()
	resultRun()
	coverRun()
	shapeRun()
}

// tRun proves each new method of T. No harness owns the test, thus no cleanup
// of the test runs, and the program moves the process back itself.
func tRun() {
	t := &testing.T{}

	dir := t.TempDir()
	fmt.Println("temp dir absolute:", filepath.IsAbs(dir))

	old, _ := os.Getwd()
	t.Chdir(dir)
	moved, _ := os.Getwd()
	fmt.Println("chdir moved the process:", moved != old)
	fmt.Println("chdir wrote PWD:", os.Getenv("PWD") == dir)
	os.Chdir(old)

	w := t.Output()
	fmt.Println("output gives a writer:", w != nil)
	fmt.Fprint(w, "this line stays in the log of the test\n")
	fmt.Println("output left the test clean:", !t.Failed())

	t.Attr("harness", "gart")
	fmt.Println("attr left the test clean:", !t.Failed())
}

// sink holds the result of the work of the benchmark, thus no compiler drops
// the loop.
var sink int

// work costs about 1 microsecond. A body that costs nothing would raise the
// count of iterations to the limit, and the run would take too long.
func work() {
	total := 0
	for i := 0; i < 1000; i++ {
		total += i * 3
	}
	sink = total
}

func benchRun() {
	var (
		ctx       context.Context
		tempOK    bool
		tempAbs   bool
		envInside string
		wdMoved   bool
		pwdOK     bool
		outCount  int
		outErr    error
		elapsedOK bool
		attrClean bool
		tempName  string
	)
	wdBefore, _ := os.Getwd()
	os.Setenv("BENCHVAR", "before")

	result := testing.Benchmark(func(b *testing.B) {
		b.StopTimer()
		ctx = b.Context()

		dir := b.TempDir()
		tempName = dir
		info, cause := os.Stat(dir)
		tempOK = cause == nil && info.IsDir()
		tempAbs = filepath.IsAbs(dir)

		b.Setenv("BENCHVAR", "inside")
		envInside = os.Getenv("BENCHVAR")

		b.Chdir(dir)
		moved, _ := os.Getwd()
		wdMoved = moved != wdBefore
		pwdOK = os.Getenv("PWD") == dir

		outCount, outErr = fmt.Fprint(b.Output(), "a line of the benchmark\n")

		b.Attr("kind", "extra")
		attrClean = !b.Failed()

		b.SetParallelism(2)

		b.StartTimer()
		for i := 0; i < b.N; i++ {
			work()
		}
		b.StopTimer()
		elapsedOK = b.Elapsed() >= 0
		b.StartTimer()
	})

	wdAfter, _ := os.Getwd()
	_, statCause := os.Stat(tempName)

	fmt.Println("bench context runs:", ctx != nil)
	fmt.Println("bench context ends with the run:", ctx.Err() == context.Canceled)
	fmt.Println("bench temp dir holds a directory:", tempOK)
	fmt.Println("bench temp dir absolute:", tempAbs)
	fmt.Println("bench temp dir gone after the run:", os.IsNotExist(statCause))
	fmt.Println("bench setenv inside:", envInside)
	fmt.Println("bench setenv after the run:", os.Getenv("BENCHVAR"))
	fmt.Println("bench chdir moved the process:", wdMoved)
	fmt.Println("bench chdir wrote PWD:", pwdOK)
	fmt.Println("bench chdir moved back:", wdAfter == wdBefore)
	fmt.Println("bench output wrote:", outCount, outErr)
	fmt.Println("bench elapsed:", elapsedOK)
	fmt.Println("bench attr left the run clean:", attrClean)
	fmt.Println("bench ran iterations:", result.N > 0)
}

// resultRun proves the memory names of BenchmarkResult. A literal holds the
// numbers, because a measured run gives a count of iterations that the 2
// runtimes do not share.
func resultRun() {
	r := testing.BenchmarkResult{
		N:         50,
		T:         5000,
		Bytes:     2048,
		MemAllocs: 400,
		MemBytes:  3200,
		Extra:     map[string]float64{"items/op": 2.5, "zunits/op": 0.0065, "aunits/op": 9.9994},
	}
	fmt.Println("result memory:", r.MemAllocs, r.MemBytes)
	fmt.Println("result per operation:", r.NsPerOp(), r.AllocsPerOp(), r.AllocedBytesPerOp())
	fmt.Printf("result memory line: %q\n", r.MemString())
	fmt.Println("result extra:", r.Extra["items/op"])
	fmt.Printf("result line: %q\n", r.String())

	over := testing.BenchmarkResult{
		N: 10, T: 1000, MemAllocs: 40, MemBytes: 800,
		Extra: map[string]float64{"ns/op": 7, "allocs/op": 3, "B/op": 9},
	}
	fmt.Println("extra beats the count:", over.NsPerOp(), over.AllocsPerOp(),
		over.AllocedBytesPerOp())
	fmt.Printf("override line: %q\n", over.String())

	empty := testing.BenchmarkResult{}
	fmt.Println("empty result:", empty.NsPerOp(), empty.AllocsPerOp(),
		empty.AllocedBytesPerOp())
	fmt.Printf("empty line: %q\n", empty.String())
}

// coverRun proves the coverage names. gart writes no counter into a package,
// and a program that go build made holds no counter either, thus the 2 runs
// report the same numbers.
func coverRun() {
	fmt.Println("cover mode empty:", testing.CoverMode() == "")
	fmt.Println("coverage:", testing.Coverage())

	block := testing.CoverBlock{Line0: 3, Col0: 5, Line1: 9, Col1: 11, Stmts: 4}
	fmt.Println("block:", block.Line0, block.Col0, block.Line1, block.Col1, block.Stmts)

	c := testing.Cover{
		Mode:            "set",
		Counters:        map[string][]uint32{"main.go": {1, 0, 1}},
		Blocks:          map[string][]testing.CoverBlock{"main.go": {block}},
		CoveredPackages: "example.com/testingextra",
	}
	testing.RegisterCover(c)
	fmt.Println("cover:", c.Mode, c.Counters["main.go"], len(c.Blocks["main.go"]),
		c.CoveredPackages)
	fmt.Println("cover mode after the register:", testing.CoverMode() == "")
	fmt.Println("testing inside a plain program:", testing.Testing())
}

// deadliner holds the shape of Deadline.
type deadliner interface {
	Deadline() (time.Time, bool)
}

// artifacts holds the shape of ArtifactDir.
type artifacts interface {
	ArtifactDir() string
}

// process holds the shape of each method that reaches the process.
type process interface {
	ArtifactDir() string
	Attr(key, value string)
	Chdir(dir string)
	Context() context.Context
	Output() io.Writer
	Setenv(key, value string)
	TempDir() string
}

// shapeRun proves the shape of each name that this program cannot call.
//
// Deadline and ArtifactDir need the flags that testing.Init writes, thus Go
// stops a program that calls them outside go test. Each method of F needs a
// fuzz run, and gart runs no fuzz target.
func shapeRun() {
	var (
		_ deadliner = (*testing.T)(nil)
		_ artifacts = (*testing.T)(nil)
		_ artifacts = (*testing.B)(nil)
		_ process   = (*testing.F)(nil)
	)
	// The method expressions prove that TB carries each new method.
	var (
		_ = testing.TB.ArtifactDir
		_ = testing.TB.Attr
		_ = testing.TB.Chdir
		_ = testing.TB.Context
		_ = testing.TB.Output
		_ = testing.TB.Setenv
		_ = testing.TB.TempDir
	)
	fmt.Println("T, B, and F hold the shape of each name")
}
