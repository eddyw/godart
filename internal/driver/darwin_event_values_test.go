package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDarwinEventValues(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("event value reference requires a selected 64-bit host")
	}
	const fixture = `package main

import (
	"fmt"
	"runtime"
	"syscall"
)

func nilEvent() {
	defer func() {
		value := recover()
		_, typed := value.(runtime.Error)
		fmt.Println("nil", value != nil, typed, fmt.Sprint(value))
	}()
	syscall.SetKevent(nil, 1, 2, 3)
}
func main() {
	marker := byte(29)
	original := syscall.Kevent_t{Fflags: 0xfedcba98, Data: -9223372036854775807, Udata: &marker}
	cases := [][3]int{{0, 0, 0}, {1, 32767, 65535}, {-1, 32768, 65536}, {-9223372036854775807, -32769, -1}, {9223372036854775807, 65535, 131071}}
	for i, args := range cases {
		value := original
		syscall.SetKevent(&value, args[0], args[1], args[2])
		fmt.Println("event", i, value.Ident, value.Filter, value.Flags, value.Fflags == original.Fflags, value.Data == original.Data, value.Udata == original.Udata, *value.Udata)
		fmt.Println("untouched", i, value.Fflags, value.Data)
	}
	nilEvent()
	fmt.Println("original", original.Ident, original.Filter, original.Flags, marker)
}
`
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			reference, dir, out := t.TempDir(), t.TempDir(), t.TempDir()
			if err := os.Mkdir(filepath.Join(reference, "reference"), 0755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(reference, "go.mod"), "module example.com/eventreference\n\ngo 1.27\n")
			writeFile(t, filepath.Join(reference, "reference", "types.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/ztypes_darwin_"+arch+".go"), []string{"Kevent_t"}))
			writeFile(t, filepath.Join(reference, "reference", "setter.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_darwin_"+arch+".go"), []string{"SetKevent"}))
			writeFile(t, filepath.Join(reference, "main.go"), strings.Replace(fixture, "\"syscall\"", "syscall \"example.com/eventreference/reference\"", 1))
			native := filepath.Join(t.TempDir(), "native")
			run(t, reference, "go", "build", "-o", native, ".")
			want := upstreamOutcome(t, reference, os.Environ(), runLimit, native)
			if want.Exit != 0 || want.Text != "event 0 0 0 0 true true true 29\nuntouched 0 4275878552 -9223372036854775807\nevent 1 1 32767 65535 true true true 29\nuntouched 1 4275878552 -9223372036854775807\nevent 2 18446744073709551615 -32768 0 true true true 29\nuntouched 2 4275878552 -9223372036854775807\nevent 3 9223372036854775809 32767 65535 true true true 29\nuntouched 3 4275878552 -9223372036854775807\nevent 4 9223372036854775807 -1 65535 true true true 29\nuntouched 4 4275878552 -9223372036854775807\nnil true true runtime error: invalid memory address or nil pointer dereference\noriginal 0 0 0 29" {
				t.Fatalf("original SDK event values: %+v", want)
			}
			t.Log("original SDK event values", arch, want.Text)
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/eventvalues\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "main.go"), fixture)
			t.Setenv("GOOS", "darwin")
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
						t.Fatalf("SDK event values differ: Go=%+v Gart=%+v", want, got)
					}
				})
			}
		})
	}
}
