// Command gart compiles Go source code into Dart source code.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

const version = "0.1.0-dev"

const usage = `gart compiles Go source code into Dart source code.

Usage:

	gart <command> [package ...]

The commands are:

	build      compile the packages into a Dart package
	run        compile the packages and run package main with the Dart VM
	test       compile the tests and run them with the Dart VM
	watch      compile the packages again after each change
	load       load the packages and list them
	version    print the version of the compiler

The build, run, test, and watch commands take these flags:

	-o dir         write the Dart package into dir
	-runtime dir   the path of the goruntime Dart package
	-format=false  do not run dart format on the output

The run command passes each argument after -- to the program.

The test command takes these flags:

	-v               write 1 line for each test
	-run expr        run only a test whose name matches the expression
	-bench expr      run each benchmark whose name matches the expression
	-benchtime d     the length that 1 benchmark targets, for example 2s
	-benchcount n    the fixed count of iterations of 1 benchmark

A benchmark runs on an ahead of time build, because the warm up cost of the
Dart VM hides the result. The number measures the Dart output, and not Go.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gart:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return errors.New("no command")
	}
	switch args[0] {
	case "version":
		fmt.Println("gart", version)
		return nil
	case "build":
		return runBuild(args[1:], false)
	case "watch":
		return runBuild(args[1:], true)
	case "run":
		return runRun(args[1:])
	case "test":
		return runTest(args[1:])
	case "load":
		return runLoad(args[1:])
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// buildFlags holds the flags that each compile command shares.
type buildFlags struct {
	set     *flag.FlagSet
	out     *string
	runtime *string
	format  *bool
}

func newBuildFlags(name string) *buildFlags {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	return &buildFlags{
		set:     set,
		out:     set.String("o", "dartout", "the output directory"),
		runtime: set.String("runtime", "", "the path of the goruntime package"),
		format:  set.Bool("format", true, "run dart format on the output"),
	}
}

// options parses the flags and gives the input of a build.
func (f *buildFlags) options(args []string) (driver.Options, error) {
	if err := f.set.Parse(args); err != nil {
		return driver.Options{}, err
	}
	dir, err := os.Getwd()
	if err != nil {
		return driver.Options{}, err
	}
	patterns := f.set.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	return driver.Options{
		Dir:      dir,
		Patterns: patterns,
		Out:      *f.out,
		Runtime:  *f.runtime,
		Format:   *f.format,
	}, nil
}

func runBuild(args []string, watch bool) error {
	options, err := newBuildFlags("build").options(args)
	if err != nil {
		return err
	}
	if watch {
		return driver.Watch(options, os.Stdout)
	}
	return driver.Build(options)
}

func runRun(args []string) error {
	build, program := split(args)
	options, err := newBuildFlags("run").options(build)
	if err != nil {
		return err
	}
	code, err := driver.Run(options, program, os.Stdout)
	if err != nil {
		return err
	}
	if code != 0 {
		os.Exit(code)
	}
	return nil
}

func runTest(args []string) error {
	flags := newBuildFlags("test")
	verbose := flags.set.Bool("v", false, "write 1 line for each test")
	pattern := flags.set.String("run", "", "run only a test whose name matches this expression")
	bench := flags.set.String("bench", "", "run each benchmark whose name matches this expression")
	benchTime := flags.set.Duration("benchtime", time.Second,
		"the length that 1 benchmark targets")
	benchCount := flags.set.Int("benchcount", 0,
		"the fixed count of iterations of 1 benchmark")
	options, err := flags.options(args)
	if err != nil {
		return err
	}
	testOptions := driver.TestOptions{
		Verbose:    *verbose,
		Run:        *pattern,
		Benchmarks: given(flags.set, "bench"),
		Bench:      *bench,
		BenchTime:  *benchTime,
		BenchCount: *benchCount,
	}
	code, err := driver.Test(options, testOptions, os.Stdout)
	if err != nil {
		return err
	}
	if code != 0 {
		os.Exit(code)
	}
	return nil
}

// given reports whether the command line held a flag. The presence of -bench
// turns the benchmarks on, and its value filters them.
func given(set *flag.FlagSet, name string) bool {
	found := false
	set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// split cuts the arguments at the first -- element. The first part belongs to
// the compiler, and the second part belongs to the program.
func split(args []string) ([]string, []string) {
	k := slices.Index(args, "--")
	if k < 0 {
		return args, nil
	}
	return args[:k], args[k+1:]
}

func runLoad(patterns []string) error {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	found, err := load.Load(dir, patterns...)
	if err != nil {
		return err
	}
	for _, p := range found.Packages {
		fmt.Printf("%s\t%d files\n", p.PkgPath, len(p.CompiledGoFiles))
	}
	return nil
}
