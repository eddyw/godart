package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDarwinBpfInstructionValues(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("event value reference requires a selected 64-bit host")
	}
	const fixture = `package main

import (
	"fmt"
	"syscall"
)

func main() {
	cases := [][4]int{{0, 0, 0, 0}, {1, 1, 1, 1}, {-1, -1, -1, -1}, {65536, 4294967296, 256, 257}, {65535, 4294967295, 255, 255}, {9223372036854775807, 9223372036854775807, -9223372036854775807, -9223372036854775807}, {-9223372036854775807, -9223372036854775807, 9223372036854775807, 9223372036854775807}}
	for i, args := range cases {
		stmt := syscall.BpfStmt(args[0], args[1])
		stmtSibling := syscall.BpfStmt(args[0], args[1])
		jump := syscall.BpfJump(args[0], args[1], args[2], args[3])
		sibling := syscall.BpfJump(args[0], args[1], args[2], args[3])
		fmt.Println("stmt", i, stmt.Code, stmt.Jt, stmt.Jf, stmt.K, stmt != nil, stmt != stmtSibling)
		fmt.Println("jump", i, jump.Code, jump.Jt, jump.Jf, jump.K, jump != nil, jump != sibling)
		copy := *jump
		alias := jump
		alias.K ^= 0xffffffff
		alias.Jt ^= 0xff
		fmt.Println("alias", i, jump.K, jump.Jt, copy.K, copy.Jt, *sibling == copy, copy == *jump)
		fmt.Println("retained", i, jump.Code == copy.Code, jump.Jf == copy.Jf, *stmt == *stmtSibling)
	}
}
`
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			reference, dir, out := t.TempDir(), t.TempDir(), t.TempDir()
			if err := os.Mkdir(filepath.Join(reference, "reference"), 0755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(reference, "go.mod"), "module example.com/eventreference\n\ngo 1.27\n")
			writeFile(t, filepath.Join(reference, "reference", "types.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/ztypes_darwin_"+arch+".go"), []string{"BpfInsn"}))
			source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/bpf_bsd.go"), []string{"BpfStmt", "BpfJump"})
			source = strings.Replace(source, "//go:build darwin || dragonfly || freebsd || netbsd || openbsd\n", "", 1)
			writeFile(t, filepath.Join(reference, "reference", "setter.go"), source)
			writeFile(t, filepath.Join(reference, "main.go"), strings.Replace(fixture, "\"syscall\"", "syscall \"example.com/eventreference/reference\"", 1))
			native := filepath.Join(t.TempDir(), "native")
			run(t, reference, "go", "build", "-o", native, ".")
			want := upstreamOutcome(t, reference, os.Environ(), runLimit, native)
			if want.Exit != 0 || want.Text != "stmt 0 0 0 0 0 true true\njump 0 0 0 0 0 true true\nalias 0 4294967295 255 0 0 true false\nretained 0 true true true\nstmt 1 1 0 0 1 true true\njump 1 1 1 1 1 true true\nalias 1 4294967294 254 1 1 true false\nretained 1 true true true\nstmt 2 65535 0 0 4294967295 true true\njump 2 65535 255 255 4294967295 true true\nalias 2 0 0 4294967295 255 true false\nretained 2 true true true\nstmt 3 0 0 0 0 true true\njump 3 0 0 1 0 true true\nalias 3 4294967295 255 0 0 true false\nretained 3 true true true\nstmt 4 65535 0 0 4294967295 true true\njump 4 65535 255 255 4294967295 true true\nalias 4 0 0 4294967295 255 true false\nretained 4 true true true\nstmt 5 65535 0 0 4294967295 true true\njump 5 65535 1 1 4294967295 true true\nalias 5 0 254 4294967295 1 true false\nretained 5 true true true\nstmt 6 1 0 0 1 true true\njump 6 1 255 255 1 true true\nalias 6 4294967294 0 1 255 true false\nretained 6 true true true" {
				t.Fatalf("original SDK filter values: %+v", want)
			}
			t.Log("original SDK filter values", arch, want.Text)
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
						t.Fatalf("SDK filter values differ: Go=%+v Gart=%+v", want, got)
					}
				})
			}
		})
	}
}
