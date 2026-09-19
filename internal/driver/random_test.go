package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestRandomReaderFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/randomfailure\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"crypto/rand"
	"errors"
	"fmt"
)
type failingReader struct{}
func (failingReader) Read([]byte) (int, error) { return 0, errors.New("reader failure") }
func main() {
	defer fmt.Println("defer must not run")
	rand.Reader = failingReader{}
	rand.Read(make([]byte, 1))
	fmt.Println("read must not return")
}
`)
	native := filepath.Join(t.TempDir(), "native")
	run(t, dir, "go", "build", "-o", native, ".")
	out := t.TempDir()
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for name, result := range map[string]programResult{
		"Go":   programOutcome(t, dir, native),
		"Gart": programOutcome(t, out, "dart", "bin/main.dart"),
	} {
		if result.Exit != 2 || !strings.HasPrefix(result.Text, "fatal error: crypto/rand: failed to read random data") || strings.Contains(result.Text, "must not") {
			t.Errorf("%s: unexpected fatal outcome: %+v", name, result)
		}
	}
}
