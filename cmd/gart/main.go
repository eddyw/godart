// Command gart compiles Go source code into Dart source code.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

const version = "0.1.0-dev"

const usage = `gart compiles Go source code into Dart source code.

Usage:

	gart <command> [package ...]

The commands are:

	build      compile the packages into a Dart package
	load       load the packages and list them
	version    print the version of the compiler

The build command takes these flags:

	-o dir         write the Dart package into dir
	-runtime dir   the path of the goruntime Dart package
	-format=false  do not run dart format on the output
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
		return runBuild(args[1:])
	case "load":
		return runLoad(args[1:])
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runBuild(args []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	out := flags.String("o", "dartout", "the output directory")
	runtime := flags.String("runtime", "", "the path of the goruntime package")
	doFormat := flags.Bool("format", true, "run dart format on the output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	patterns := flags.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	return driver.Build(driver.Options{
		Dir:      dir,
		Patterns: patterns,
		Out:      *out,
		Runtime:  *runtime,
		Format:   *doFormat,
	})
}

func runLoad(patterns []string) error {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	pkgs, err := load.Load(dir, patterns...)
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		fmt.Printf("%s\t%d files\n", p.PkgPath, len(p.CompiledGoFiles))
	}
	return nil
}
