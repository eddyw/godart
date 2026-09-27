package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessIdentity(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("POSIX host identity")
	}
	dir := t.TempDir()
	program := `package main
import ("errors"; "encoding/json"; "os"; "slices"; "syscall")
func main() {
 group, err := syscall.Getpgid(0)
 if err != nil || group != syscall.Getpgrp() { panic("process group differs") }
 actual, err := syscall.Getpgid(syscall.Getpid())
 if err != nil || actual != group { panic("explicit process group differs") }
 queries := []func(int)(int,error){syscall.Getpgid}
 // SESSION_QUERY
 for _, query := range queries {
  for _, pid := range []int{-1, 1<<30} {
   value, err := query(pid)
   if value != -1 || !errors.Is(err,syscall.ESRCH) { panic("missing process error differs") }
  }
  value, err := query(1<<32)
  zero, zeroErr := query(0)
  if err != nil || zeroErr != nil || value != zero { panic("pid width differs") }
 }
 groups, err := os.Getgroups()
 if err != nil { panic(err) }
 rawGroups, err := syscall.Getgroups()
 if err != nil { panic(err) }
 if !slices.Equal(groups, rawGroups) || (groups == nil) != (rawGroups == nil) { panic("group identity differs") }
 if syscall.Getpid() != os.Getpid() { panic("process identity differs") }
 if syscall.Getuid() != os.Getuid() || syscall.Geteuid() != os.Geteuid() || syscall.Getgid() != os.Getgid() || syscall.Getegid() != os.Getegid() || syscall.Getppid() != os.Getppid() || syscall.Getpagesize() != os.Getpagesize() { panic("syscall identity differs") }
 json.NewEncoder(os.Stdout).Encode(append([]int{os.Getuid(), os.Geteuid(), os.Getgid(), os.Getegid(), os.Getppid(), os.Getpagesize()}, groups...))
}
`
	sessionQuery := ""
	if runtime.GOOS == "darwin" {
		sessionQuery = ` session, err := syscall.Getsid(0)
 if err != nil || session <= 0 { panic("session lookup failed") }
 actual, err = syscall.Getsid(syscall.Getpid())
 if err != nil || actual != session { panic("explicit session differs") }
 queries = append(queries, syscall.Getsid)
`
	}
	writeFile(t, filepath.Join(dir, "main.go"), strings.Replace(program, " // SESSION_QUERY\n", sessionQuery, 1))

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processidentity\n\ngo 1.27\n\nrequire example.com/testenv v0.0.0\nreplace example.com/testenv => ./testenv\n")
	source := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/exec_unix_test.go"), []string{"command", "command.Info", "command.Start", "command.Stop", "create", "parent", "TestZeroSysProcAttr"})
	source = strings.Replace(source, "package syscall_test", "package main", 1)
	source = strings.ReplaceAll(source, `"internal/testenv"`, `"example.com/testenv"`)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), source)
	if err := os.Mkdir(filepath.Join(dir, "testenv"), 0755); err != nil {
		t.Fatal(err)
	}
	helper := selectedUpstreamSource(t, "../overlay/std/internal/testenv/testenv.go", []string{"MustHaveExec"})
	writeFile(t, filepath.Join(dir, "testenv/go.mod"), "module example.com/testenv\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "testenv/testenv.go"), strings.TrimPrefix(helper, "//go:build dart\n"))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "testenv/LICENSE"), string(license))

	native := filepath.Join(dir, "native")
	run(t, dir, "go", "build", "-o", native, ".")
	want := runProgram(t, dir, native)
	var observed []int
	if err := json.Unmarshal([]byte(want), &observed); err != nil {
		t.Fatal(err)
	}
	if len(observed) < 6 || observed[4] != os.Getpid() {
		t.Fatalf("native parent contract: %v", observed)
	}
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	t.Run("vm", func(t *testing.T) {
		if got := runProgram(t, out, "dart", "bin/main.dart"); got != want {
			t.Errorf("identity mismatch: Go %s Dart %s", want, got)
		}
	})
	t.Run("aot", func(t *testing.T) {
		exe := filepath.Join(out, "identity")
		run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", exe)
		if got := runProgram(t, out, exe); got != want {
			t.Errorf("identity mismatch: Go %s Dart %s", want, got)
		}
	})
	testOut := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: testOut, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, testOut)
	for _, target := range []string{"vm", "aot"} {
		t.Run("upstream/"+target, func(t *testing.T) {
			args := []string{"-test.v", "-test.run=^TestZeroSysProcAttr$"}
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(testOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, testOut, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "syscall-process-group", "syscall", "^TestZeroSysProcAttr$", target, dir, result.Text, result.Exit)
		})
	}
}
