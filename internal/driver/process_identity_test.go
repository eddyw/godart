package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestProcessIdentity(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("POSIX host identity")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/processidentity\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import ("encoding/json"; "os")
func main() {
 groups, err := os.Getgroups()
 if err != nil { panic(err) }
 json.NewEncoder(os.Stdout).Encode(append([]int{os.Getuid(), os.Geteuid(), os.Getgid(), os.Getegid(), os.Getppid(), os.Getpagesize()}, groups...))
}
`)
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
}
