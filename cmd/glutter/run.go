package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/eddyw/godart/internal/driver"
)

// run holds the application on a device and it writes the Dart again after
// each change of the Go source.
//
// The keys of flutter run stay, because the child reads the terminal of the
// user. A build that compiled a package sends SIGUSR1 to flutter run, and
// flutter then does a hot reload.
func (a *app) run(args []string) error {
	if d := deviceOf(args); d != "" {
		if err := a.runner(d); err != nil {
			return err
		}
	}
	if err := a.build(checks{vet: true}, false); err != nil {
		return err
	}
	if _, err := a.writeHost(); err != nil {
		return err
	}

	pidFile := filepath.Join(a.cacheDir(), "run.pid")
	os.Remove(pidFile)
	full := append([]string{"run", "--pid-file=" + pidFile}, args...)
	cmd := exec1(a.dir, "flutter", full)
	if err := cmd.Start(); err != nil {
		return err
	}
	// The watch runs until this process ends. flutter run holds the terminal,
	// thus the user stops the pair with q or with Ctrl-C.
	go a.watch(pidFile)
	return cmd.Wait()
}

// watch compiles the module again after each change, and it asks flutter run
// for a hot reload.
//
// The scan of the node types runs in front of each build, because the node
// filter of the build reads it. It costs about 1 second, and writeHost reads
// the answer of that same run. The application already holds the new code of
// each node type that its host covers, thus a new node type reaches the screen
// 1 reload later.
func (a *app) watch(pidFile string) {
	opt := a.options(false)
	err := driver.WatchWith(opt, os.Stdout, func(stats *driver.Stats, err error) {
		if err != nil || len(stats.Compiled) == 0 {
			return
		}
		reload(pidFile)
		changed, err := a.writeHost()
		if err != nil {
			fmt.Fprintln(os.Stderr, "glutter:", err)
			return
		}
		if changed {
			fmt.Printf("glutter: %s holds a new node type\n", a.host)
			reload(pidFile)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "glutter: watch:", err)
	}
}

// reload sends SIGUSR1 to flutter run. SIGUSR2 would restart the application
// and it would lose the state of the Go program.
func reload(pidFile string) {
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return
	}
	syscall.Kill(pid, syscall.SIGUSR1)
}

// deviceOf gives the device of the arguments, and an empty text when the user
// named none.
func deviceOf(args []string) string {
	for i, s := range args {
		switch {
		case s == "-d" || s == "--device-id":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(s, "--device-id="):
			return strings.TrimPrefix(s, "--device-id=")
		case strings.HasPrefix(s, "-d="):
			return strings.TrimPrefix(s, "-d=")
		}
	}
	return ""
}

// exec1 gives a command that reads and writes the terminal of the user.
func exec1(dir, name string, args []string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd
}
