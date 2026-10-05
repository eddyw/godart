package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSyscallWaitStatusValues(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("event value reference requires a selected 64-bit host")
	}
	const fixture = `package main

import (
	"fmt"
	"runtime"
	"syscall"
)

func nilStatus() {
	defer func() {
		value := recover()
		_, typed := value.(runtime.Error)
		fmt.Println("nil", value != nil, typed, fmt.Sprint(value))
	}()
	var value *syscall.WaitStatus
	value.TrapCause()
}
func main() {
	for i, raw := range []uint64{0, 1, 9, 0x89, 0x100, 0x7f, 0xff, 0x117f, 0x137f, 0x57f, 0x1057f, 0xffff, 0x123400, 0xffffffff, 0xffffff00, 0x100000100, 0xffffffffffffffff} {
		value := syscall.WaitStatus(raw)
		original := value
		alias := &value
		fmt.Println("state", i, uint32(value), value.Exited(), value.Signaled(), value.CoreDump(), value.Stopped(), value.Continued(), value.ExitStatus(), int(value.Signal()), int(value.StopSignal()), value.TrapCause())
		fmt.Println("retained", i, value == original, *alias == original)
	}
	nilStatus()
}
`
	expected := map[string]string{"darwin": "state 0 0 true false false false false 0 -1 -1 -1\nretained 0 true true\nstate 1 1 false true false false false -1 1 -1 -1\nretained 1 true true\nstate 2 9 false true false false false -1 9 -1 -1\nretained 2 true true\nstate 3 137 false true true false false -1 9 -1 -1\nretained 3 true true\nstate 4 256 true false false false false 1 -1 -1 -1\nretained 4 true true\nstate 5 127 false false false true false -1 -1 0 -1\nretained 5 true true\nstate 6 255 false false false true false -1 -1 0 -1\nretained 6 true true\nstate 7 4479 false false false false true -1 -1 -1 -1\nretained 7 true true\nstate 8 4991 false false false true false -1 -1 19 -1\nretained 8 true true\nstate 9 1407 false false false true false -1 -1 5 -1\nretained 9 true true\nstate 10 66943 false false false true false -1 -1 5 -1\nretained 10 true true\nstate 11 65535 false false false true false -1 -1 255 -1\nretained 11 true true\nstate 12 1192960 true false false false false 4660 -1 -1 -1\nretained 12 true true\nstate 13 4294967295 false false false true false -1 -1 255 -1\nretained 13 true true\nstate 14 4294967040 true false false false false 16777215 -1 -1 -1\nretained 14 true true\nstate 15 256 true false false false false 1 -1 -1 -1\nretained 15 true true\nstate 16 4294967295 false false false true false -1 -1 255 -1\nretained 16 true true\nnil true true runtime error: invalid memory address or nil pointer dereference", "linux": "state 0 0 true false false false false 0 -1 -1 -1\nretained 0 true true\nstate 1 1 false true false false false -1 1 -1 -1\nretained 1 true true\nstate 2 9 false true false false false -1 9 -1 -1\nretained 2 true true\nstate 3 137 false true true false false -1 9 -1 -1\nretained 3 true true\nstate 4 256 true false false false false 1 -1 -1 -1\nretained 4 true true\nstate 5 127 false false false true false -1 -1 0 -1\nretained 5 true true\nstate 6 255 false false false false false -1 -1 -1 -1\nretained 6 true true\nstate 7 4479 false false false true false -1 -1 17 -1\nretained 7 true true\nstate 8 4991 false false false true false -1 -1 19 -1\nretained 8 true true\nstate 9 1407 false false false true false -1 -1 5 0\nretained 9 true true\nstate 10 66943 false false false true false -1 -1 5 1\nretained 10 true true\nstate 11 65535 false false false false true -1 -1 -1 -1\nretained 11 true true\nstate 12 1192960 true false false false false 52 -1 -1 -1\nretained 12 true true\nstate 13 4294967295 false false false false false -1 -1 -1 -1\nretained 13 true true\nstate 14 4294967040 true false false false false 255 -1 -1 -1\nretained 14 true true\nstate 15 256 true false false false false 1 -1 -1 -1\nretained 15 true true\nstate 16 4294967295 false false false false false -1 -1 -1 -1\nretained 16 true true\nnil true true runtime error: invalid memory address or nil pointer dereference"}
	for _, goos := range []string{"darwin", "linux"} {
		for _, arch := range []string{"amd64", "arm64"} {
			t.Run(goos+"/"+arch, func(t *testing.T) {
				reference, dir, out := t.TempDir(), t.TempDir(), t.TempDir()
				if err := os.Mkdir(filepath.Join(reference, "reference"), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(reference, "go.mod"), "module example.com/eventreference\n\ngo 1.27\n")
				decoder := "syscall_linux.go"
				if goos == "darwin" {
					decoder = "syscall_bsd.go"
				}
				source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall", decoder), []string{"WaitStatus", "mask"})
				source = strings.Replace(source, "//go:build darwin || dragonfly || freebsd || netbsd || openbsd\n", "", 1)
				writeFile(t, filepath.Join(reference, "reference", "decoder.go"), source)
				signalSource := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_unix.go"), []string{"Signal"})
				signalSource = strings.Replace(signalSource, "\"internal/strconv\"", "\"strconv\"", 1)
				writeFile(t, filepath.Join(reference, "reference", "signals.go"), signalSource)
				writeFile(t, filepath.Join(reference, "reference", "constants.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/zerrors_"+goos+"_"+arch+".go"), []string{"SIGSTOP", "SIGTRAP", "signals"}))
				writeFile(t, filepath.Join(reference, "main.go"), strings.Replace(fixture, "\"syscall\"", "syscall \"example.com/eventreference/reference\"", 1))
				native := filepath.Join(t.TempDir(), "native")
				run(t, reference, "go", "build", "-o", native, ".")
				want := upstreamOutcome(t, reference, os.Environ(), runLimit, native)
				if want.Exit != 0 || want.Text != expected[goos] {
					t.Fatalf("original SDK wait values: %+v", want)
				}
				t.Log("original SDK wait values", goos, arch, want.Text)
				writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/eventvalues\n\ngo 1.27\n")
				writeFile(t, filepath.Join(dir, "main.go"), fixture)
				t.Setenv("GOOS", goos)
				t.Setenv("GOARCH", arch)
				t.Setenv("CGO_ENABLED", "0")
				if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
					t.Fatal(err)
				}
				prepareDart(t, out)
				for _, target := range []string{"vm", "aot", "js", "wasm"} {
					t.Run(target, func(t *testing.T) {
						var got programResult
						if target == "vm" {
							got = upstreamOutcome(t, dir, os.Environ(), runLimit, "dart", filepath.Join(out, "bin/main.dart"))
						} else {
							got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
						}
						if got != want {
							t.Fatalf("SDK wait values differ: Go=%+v Gart=%+v", want, got)
						}
					})
				}
			})
		}
	}
}
