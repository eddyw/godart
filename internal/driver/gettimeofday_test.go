package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestNativeGettimeofday(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("gettimeofday requires Linux or macOS")
	}
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/gettimeofday\n\ngo 1.27\n")
	program := `package main
import("fmt";"syscall";"time")
func main(){
 fmt.Println("nil",syscall.Gettimeofday(nil))
 tv:=syscall.Timeval{Sec:-1,Usec:-1}
 before:=time.Now().Truncate(time.Microsecond)
 err:=syscall.Gettimeofday(&tv)
 after:=time.Now()
 stamp:=time.Unix(int64(tv.Sec),int64(tv.Usec)*1000)
 fmt.Println("value",err,tv.Sec>0,tv.Usec>=0&&tv.Usec<1000000,!stamp.Before(before)&&!stamp.After(after))
}
`
	if runtime.GOOS == "darwin" {
		program = strings.Replace(program, "before:=time.Now()", "tv.Pad_cgo_0=[4]byte{1,2,3,4}\nbefore:=time.Now()", 1)
		program = strings.Replace(program, "after:=time.Now()", "fmt.Println(\"padding\",tv.Pad_cgo_0)\nafter:=time.Now()", 1)
	}
	writeFile(t, filepath.Join(dir, "main.go"), program)
	upstream := selectedUpstreamSource(t, filepath.Join(runtime.GOROOT(), "src/syscall/syscall_test.go"), []string{"TestGettimeofday"})
	upstream = strings.Replace(upstream, "package syscall_test", "package main", 1)
	writeFile(t, filepath.Join(dir, "upstream_test.go"), upstream)
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))

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
	inventory, err := parity.Discover(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	testOut := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: testOut, Runtime: runtimeDir(t), Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, testOut)
	for _, target := range []string{"vm", "aot"} {
		t.Run("upstream/"+target, func(t *testing.T) {
			args := upstreamTestArgs("-test.run=^TestGettimeofday$")
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(testOut, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, testOut, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstreamInventory(t, "syscall-gettimeofday", "syscall", "^TestGettimeofday$", target, dir, result.Text, result.Exit, *inventory, nil, nil, nil, false)
		})
	}

}
