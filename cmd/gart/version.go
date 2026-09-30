package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/eddyw/godart/internal/driver"
)

func runVersion(args []string) error {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	verbose := flags.Bool("v", false, "report compiler, SDK, module, and frontend versions")
	frontend := flags.String("frontend", "auto", "select auto, resident, or source Dart execution")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("version takes no package arguments")
	}
	fmt.Println("gart", version)
	if !*verbose {
		return nil
	}
	fmt.Println("compiler build Go:", runtime.Version())
	info, err := driver.Diagnose(".", *frontend)
	if err != nil {
		return err
	}
	fmt.Println("active Go:", info.Go.Version)
	fmt.Println("Go source root:", info.Go.Root)
	if info.Go.Module == "" {
		fmt.Println("module: none")
	} else {
		fmt.Println("module:", info.Go.Module)
		language := info.Go.ModuleVersion
		if language == "" {
			language = "not declared"
		}
		fmt.Println("module Go directive:", language)
	}
	fmt.Println("Dart SDK:", info.Dart)
	fmt.Printf("Dart frontend: %s (requested %s)\n", info.Frontend, *frontend)
	return nil
}
