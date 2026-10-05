package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLinuxArm64Fstatat(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Fstatat requires Linux")
	}
	const fixture = `package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func equal(a, b syscall.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Mode == b.Mode && a.Nlink == b.Nlink && a.Uid == b.Uid && a.Gid == b.Gid && a.Rdev == b.Rdev && a.Size == b.Size && a.Blksize == b.Blksize && a.Blocks == b.Blocks && a.Atim == b.Atim && a.Mtim == b.Mtim && a.Ctim == b.Ctim
}
func main() {
	dir, err := os.MkdirTemp("", "fstatat-contract-")
	must(err)
	defer os.RemoveAll(dir)
	must(os.WriteFile(filepath.Join(dir, "file"), []byte{11, 13, 17}, 0600))
	must(os.WriteFile(filepath.Join(dir, "raw\x80"), []byte{19, 23}, 0600))
	must(os.Symlink("file", filepath.Join(dir, "link")))
	fd, err := syscall.Open(dir, syscall.O_RDONLY, 0)
	must(err)
	defer syscall.Close(fd)
	if len(os.Args) > 1 && os.Args[1] == "nil" {
		err := syscall.Fstatat(fd, "file", nil, 0)
		var code syscall.Errno
		fmt.Println("nil", errors.As(err, &code), int(code))
		return
	}
	cases := []struct {
		name, path string
		fd, flags  int
		reference  string
	}{
		{"file", "file", fd, 0, "file"}, {"follow", "link", fd, 0, "file"}, {"nofollow", "link", fd, 0x100, "link"}, {"raw", "raw\x80", fd, 0, "raw\x80"},
		{"empty", "", fd, 0, ""}, {"empty-path", "", fd, 0x1000, "directory"}, {"missing", "missing", fd, 0, ""}, {"invalid-fd", "file", -1, 0, ""},
		{"nul", "file\x00bad", -1, 0, ""}, {"invalid-flags", "file", fd, 0x400, ""}, {"fd-width", "file", fd + (1 << 32), 0, "file"}, {"flags-width", "link", fd, 0x100 + (1 << 32), "link"},
	}
	initial := syscall.Stat_t{Dev: 37, Ino: 41, Mode: 0xdeadbeef, Size: -43, Mtim: syscall.Timespec{Sec: 47, Nsec: 53}}
	for _, c := range cases {
		values := [3]syscall.Stat_t{initial, initial, initial}
		alias := &values[1]
		err := syscall.Fstatat(c.fd, c.path, alias, c.flags)
		var code syscall.Errno
		exact := values[1] == initial
		if err == nil {
			var reference syscall.Stat_t
			if c.reference == "directory" {
				must(syscall.Fstat(fd, &reference))
			} else if c.reference == "link" {
				must(syscall.Lstat(filepath.Join(dir, c.reference), &reference))
			} else {
				must(syscall.Stat(filepath.Join(dir, c.reference), &reference))
			}
			exact = equal(values[1], reference)
		}
		fmt.Println(c.name, errors.As(err, &code), int(code), exact, values[0] == initial, values[2] == initial, alias == &values[1])
	}
	bytes := make([]byte, 3)
	file, err := syscall.Open(filepath.Join(dir, "file"), syscall.O_RDONLY, 0)
	must(err)
	n, err := syscall.Read(file, bytes)
	must(err)
	must(syscall.Close(file))
	fmt.Println("bytes", n, bytes)
	duplicate, err := syscall.Dup(fd)
	must(err)
	value := initial
	must(syscall.Fstatat(duplicate, "file", &value, 0))
	must(syscall.Close(duplicate))
	value = initial
	err = syscall.Fstatat(duplicate, "file", &value, 0)
	var code syscall.Errno
	fmt.Println("closed", errors.As(err, &code), int(code), value == initial)
}
`
	dir, out, reference := t.TempDir(), t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(reference, "reference"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), fixture)
	writeFile(t, filepath.Join(reference, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(reference, "reference", "wrapper.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_linux_arm64.go"), []string{"Fstatat"}))
	writeFile(t, filepath.Join(reference, "reference", "native.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/zsyscall_linux_"+runtime.GOARCH+".go"), []string{"fstatat"}))
	writeFile(t, filepath.Join(reference, "reference", "errno.go"), selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_unix.go"), []string{"errnoErr", "errEAGAIN", "errEINVAL", "errENOENT"}))
	constant := "SYS_FSTATAT"
	if runtime.GOARCH == "amd64" {
		constant = "SYS_NEWFSTATAT"
	}
	writeFile(t, filepath.Join(reference, "reference", "bindings.go"), `package syscall
import "syscall"
type Stat_t=syscall.Stat_t
type Errno=syscall.Errno
const(EAGAIN=syscall.EAGAIN;EINVAL=syscall.EINVAL;ENOENT=syscall.ENOENT)
var BytePtrFromString=syscall.BytePtrFromString
var Syscall6=syscall.Syscall6
const `+constant+`=syscall.`+constant+"\n")
	writeFile(t, filepath.Join(reference, "main.go"), strings.Replace(strings.Replace(fixture, "\"syscall\"", "\"syscall\"\n reference \"example.com/accept/reference\"", 1), "syscall.Fstatat", "reference.Fstatat", -1))
	native := filepath.Join(t.TempDir(), "native")
	run(t, reference, "go", "build", "-o", native, ".")
	want := map[string]programResult{}
	for _, mode := range []string{"normal", "nil"} {
		args := []string{}
		if mode == "nil" {
			args = []string{"nil"}
		}
		want[mode] = upstreamOutcome(t, reference, nil, runLimit, native, args...)
		expected := "file false 0 true true true true\nfollow false 0 true true true true\nnofollow false 0 true true true true\nraw false 0 true true true true\nempty true 2 true true true true\nempty-path false 0 true true true true\nmissing true 2 true true true true\ninvalid-fd true 9 true true true true\nnul true 22 true true true true\ninvalid-flags true 22 true true true true\nfd-width false 0 true true true true\nflags-width false 0 true true true true\nbytes 3 [11 13 17]\nclosed true 9 true"
		if mode == "nil" {
			expected = "nil true 14"
		}
		actual, wanted := want[mode].Text, expected
		expectedExit := 0
		if runtime.GOARCH == "amd64" && mode == "normal" {
			// The x64 emulator changes native stat fields after failed calls.
			// ARM64 retains them. Only ARM64 supplies that caller-data oracle.
			actualLines, wantedLines := strings.Split(actual, "\n"), strings.Split(wanted, "\n")
			if len(actualLines) != len(wantedLines) {
				t.Fatalf("native Fstatat line count: %s", actual)
			}
			for _, index := range []int{4, 6, 7, 9, 13} {
				left, right := strings.Fields(actualLines[index]), strings.Fields(wantedLines[index])
				if len(left) != len(right) || (left[3] != "true" && left[3] != "false") {
					t.Fatalf("native Fstatat fields: %s", actualLines[index])
				}
				actualLines[index] = strings.Join(append(left[:3], left[4:]...), " ")
				wantedLines[index] = strings.Join(append(right[:3], right[4:]...), " ")
			}
			actual, wanted = strings.Join(actualLines, "\n"), strings.Join(wantedLines, "\n")
		}
		if runtime.GOARCH == "amd64" && mode == "nil" {
			expectedExit = -1
			wanted = ""
			t.Logf("native AMD64 nil metadata signal exit %d", want[mode].Exit)
		}
		if want[mode].Exit != expectedExit || actual != wanted {
			t.Fatalf("native Fstatat %s: %+v, want exit %d text %s", mode, want[mode], expectedExit, wanted)
		}
		t.Logf("original SDK Fstatat %s %s", mode, want[mode].Text)
	}
	if runtime.GOARCH == "arm64" {
		actual := filepath.Join(t.TempDir(), "actual")
		run(t, dir, "go", "build", "-o", actual, ".")
		for _, mode := range []string{"normal", "nil"} {
			args := []string{}
			if mode == "nil" {
				args = []string{"nil"}
			}
			got := upstreamOutcome(t, dir, nil, runLimit, actual, args...)
			if got != want[mode] {
				t.Fatalf("actual public Fstatat %s: got %+v, want %+v", mode, got, want[mode])
			}
			t.Logf("actual public ARM64 Fstatat %s", mode)
		}
	}
	t.Setenv("GOOS", "linux")
	t.Setenv("GOARCH", "arm64")
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			for _, mode := range []string{"normal", "nil"} {
				t.Run(mode, func(t *testing.T) {
					args := []string{}
					if mode == "nil" {
						args = []string{"nil"}
					}
					var got programResult
					if target == "vm" {
						got = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, "bin/main.dart")}, args...)...)
					} else {
						got = runUpstreamTarget(t, target, out, "bin/main.dart", dir, args)
					}
					expected := want[mode]
					if mode == "normal" {
						expected.Text = "file false 0 true true true true\nfollow false 0 true true true true\nnofollow false 0 true true true true\nraw false 0 true true true true\nempty true 2 true true true true\nempty-path false 0 true true true true\nmissing true 2 true true true true\ninvalid-fd true 9 true true true true\nnul true 22 true true true true\ninvalid-flags true 22 true true true true\nfd-width false 0 true true true true\nflags-width false 0 true true true true\nbytes 3 [11 13 17]\nclosed true 9 true"
					}
					if got != expected {
						t.Fatalf("Fstatat %s/%s: got %+v, want %+v", target, mode, got, expected)
					}
				})
			}
		})
	}
	web, webout := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(web, "go.mod"), "module example.com/accept\n\ngo 1.27\n")
	writeFile(t, filepath.Join(web, "main.go"), `package main
import("errors";"fmt";"syscall")
func main(){
 initial:=syscall.Stat_t{Dev:37,Ino:41,Mode:0xdeadbeef,Size:-43,Mtim:syscall.Timespec{Sec:47,Nsec:53}}
 for _,path:=range []string{"file","raw\x80","","file\x00bad"} {
  value:=initial;err:=syscall.Fstatat(-1,path,&value,0);var code syscall.Errno
  fmt.Println("web",errors.As(err,&code),int(code),errors.Is(err,errors.ErrUnsupported),value==initial)
 }
 err:=syscall.Fstatat(-1,"file",nil,0);var code syscall.Errno;fmt.Println("nil",errors.As(err,&code),int(code),errors.Is(err,errors.ErrUnsupported))
}`)
	if err := driver.Build(driver.Options{Dir: web, Patterns: []string{"."}, Out: webout, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, webout)
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, webout, "bin/main.dart", web, []string{})
			expected := "web true 38 true true\nweb true 38 true true\nweb true 38 true true\nweb true 22 false true\nnil true 38 true"
			if got.Exit != 0 || got.Text != expected {
				t.Fatalf("web Fstatat %s: got %+v, want %s", target, got, expected)
			}
		})
	}
}
