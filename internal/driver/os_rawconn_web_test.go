package driver_test

import (
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestOSRawConnWebUnavailable(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rawunavailable\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
 "errors"
 "fmt"
 "os"
 "syscall"
)
func main() {
 var f *os.File
 c,err := f.SyscallConn()
 if c != nil || err != os.ErrInvalid { panic("nil receiver") }
 c,err = os.Stdin.SyscallConn()
 if c != nil || err != syscall.ENOSYS || !errors.Is(err,errors.ErrUnsupported) { panic("native descriptor unavailable") }
 fmt.Println("raw native descriptor unavailable",true)
}
`)
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), TrimPath: true}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			got := runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			if got.Exit != 0 || got.Text != "raw native descriptor unavailable true" {
				t.Fatalf("RawConn ENOSYS differs: %+v", got)
			}
		})
	}
}
