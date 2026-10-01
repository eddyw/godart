package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestResourceUsage(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("resource usage requires a native filesystem")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/resourceusage\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
 "fmt"
 "syscall"
 "time"
)

func must(err error) { if err != nil { panic(err) } }
func result(label string, err error) {
 code, typed := err.(syscall.Errno)
 fmt.Println(label, int(code), typed)
}
func valid(value syscall.Timeval) bool { return value.Sec >= 0 && value.Usec >= 0 && value.Usec < 1000000 }
func cpu(value syscall.Rusage) int64 { return value.Utime.Nano() + value.Stime.Nano() }
func main() {
 var before syscall.Rusage
 must(syscall.Getrusage(syscall.RUSAGE_SELF, &before))
 fmt.Println("self", valid(before.Utime), valid(before.Stime), before.Maxrss > 0, before.Minflt >= 0, before.Majflt >= 0)
 until := time.Now().Add(100*time.Millisecond)
 var count uint64
 for time.Now().Before(until) { count++ }
 var after syscall.Rusage
 must(syscall.Getrusage(syscall.RUSAGE_SELF, &after))
 fmt.Println("actual cpu", cpu(after) > cpu(before), count > 0, after.Maxrss >= before.Maxrss)
 var children syscall.Rusage
 must(syscall.Getrusage(syscall.RUSAGE_CHILDREN, &children))
 fmt.Println("children", valid(children.Utime), valid(children.Stime), children.Maxrss >= 0)
 result("self nil", syscall.Getrusage(syscall.RUSAGE_SELF, nil))
 result("children nil", syscall.Getrusage(syscall.RUSAGE_CHILDREN, nil))
 result("invalid nil", syscall.Getrusage(123456, nil))
 value := syscall.Rusage{Maxrss: 123, Minflt: 456}
 saved := value
 result("invalid", syscall.Getrusage(123456, &value))
 fmt.Println("output preserved", value == saved)
}
`)
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	targets := []string{"vm", "aot"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			var got programResult
			if target == "vm" {
				got = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if got != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, got.Exit, got.Text)
			}
		})
	}
}
