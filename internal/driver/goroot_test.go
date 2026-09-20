package driver_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestGOROOTEnvironment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/goroot\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
    "fmt"
    "go/build"
    "os"
    "runtime"
)
func main() {
    original := runtime.GOROOT()
    fmt.Printf("%q %q\n", original, build.Default.GOROOT)
    os.Setenv("GOROOT", "/changed")
    fmt.Println(runtime.GOROOT() == original)
    os.Unsetenv("GOROOT")
    fmt.Println(runtime.GOROOT() == original)
}
`)
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	var base []string
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "GOROOT=") {
			base = append(base, item)
		}
	}
	for _, root := range []string{"", t.TempDir()} {
		t.Run(fmt.Sprintf("set=%t", root != ""), func(t *testing.T) {
			env := append(base[:len(base):len(base)], "GOROOT="+root)
			got, err, ended := output(out, env, runLimit, "dart", "bin/main.dart")
			want := fmt.Sprintf("%q %q\ntrue\ntrue\n", root, root)
			if err != nil || !ended || string(got) != want {
				t.Fatalf("GOROOT result: ended=%t err=%v\nwant: %s\ngot: %s", ended, err, want, got)
			}
		})
	}
}
