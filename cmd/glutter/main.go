// Command glutter runs a Flutter application whose widget tree comes from Go.
// It is to the flutter command what gart is to the go command.
//
// Usage:
//
//	glutter <command> [argument ...]
//
// The commands are:
//
//	run        compile the Go source, hold it under watch, and run the application
//	build      compile the Go source, then run flutter build
//	test       compile the Go source, then run flutter test
//	gen        compile the Go source and write the host
//	bind       generate application widget and service bindings
//	create     write the platform runner of the application
//	doctor     report the Flutter version against the node set
//	new        write a new application
//	version    print the version of the tool
//
// Each other argument goes to the flutter command with no change.
//
// glutter reads the layout of the application from pubspec.yaml. The defaults
// are the directory go for the Go module, gen for the Dart output, and
// lib/widgets_gen.dart for the host. A glutter block of pubspec.yaml holds
// another 1:
//
//	glutter:
//	  module: go
//	  out: gen
//	  host: lib/widgets_gen.dart
package main

import (
	"fmt"
	"os"

	"github.com/eddyw/godart/internal/notices"
)

const usage = `glutter runs a Flutter application whose widget tree comes from Go.

Usage:

	glutter <command> [argument ...]

The commands are:

	run        compile the Go source, hold it under watch, and run the application
	build      compile the Go source, then run flutter build
	test       compile the Go source, then run flutter test
	gen        compile the Go source and write the host
	bind       generate widget and service bindings from glutter.bindings.json
	create     write the platform runner of the application
	doctor     report the Flutter version against the node set
	new        write a new application
	version    print the version of the tool
	licenses   print bundled license notices

Each other argument goes to the flutter command with no change.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "licenses":
		fmt.Print(notices.Compiler)
		return
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	case "version":
		fmt.Println("glutter " + version)
		return
	case "new":
		err = newApp(args)
	case "gen":
		err = withApp(args, func(a *app, rest []string) error { return a.gen() })
	case "bind":
		err = withApp(args, func(a *app, rest []string) error {
			return run1(a.nodegenPath(), "dart", "run", "bin/nodegen.dart", "bind", a.dir)
		})
	case "create":
		err = withApp(args, (*app).create)
	case "doctor":
		err = withApp(args, func(a *app, rest []string) error { return a.doctor() })
	case "run":
		err = withApp(args, (*app).run)
	case "build", "test":
		err = withApp(args, func(a *app, rest []string) error {
			return a.after(cmd, rest)
		})
	default:
		err = flutterPass(append([]string{cmd}, args...))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "glutter:", err)
		os.Exit(1)
	}
}

// version names the release of the tool.
const version = "0.1.0"

// withApp reads the application of the working directory and it runs f.
func withApp(args []string, f func(*app, []string) error) error {
	a, err := findApp()
	if err != nil {
		return err
	}
	return f(a, args)
}
