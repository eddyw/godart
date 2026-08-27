// Command gart compiles Go source code into Dart source code.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/eddyw/godart/internal/load"
)

const version = "0.1.0-dev"

const usage = `gart compiles Go source code into Dart source code.

Usage:

	gart <command> [package ...]

The commands are:

	load       load the packages and list them
	version    print the version of the compiler
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
	case "load":
		return runLoad(args[1:])
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
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
