package driver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// buildEmbed compiles a module of 1 file and gives the report of the build.
func buildEmbed(t *testing.T, main string, files map[string]string) error {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module embedtest\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), main)
	for name, text := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
			t.Fatal(err)
		}
		writeFile(t, full, text)
	}
	return driver.Build(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtimeDir(t),
	})
}

// TestEmbedRejectsTheConstructor proves that a file of the user cannot name
// the constructor of the package embed. The real package of Go holds no such
// name, thus a call would compile here and fail under the go command.
func TestEmbedRejectsTheConstructor(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	"embed"
	"fmt"
)

func main() {
	f := embed.New([]string{"a.txt"}, []string{"a"})
	text, _ := f.ReadFile("a.txt")
	fmt.Println(string(text))
}
`, nil)
	if err == nil {
		t.Fatal("the build accepted a call of embed.New")
	}
	want := "the package embed of Go holds no such name"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// TestEmbedRejectsAPatternThatNamesNoFile follows the go command, which
// rejects a directive whose pattern reaches nothing.
func TestEmbedRejectsAPatternThatNamesNoFile(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	"embed"
	"fmt"
)

//go:embed static/*.txt
var assets embed.FS

func main() { fmt.Println(assets) }
`, map[string]string{"static/keep.bin": "x"})
	if err == nil {
		t.Fatal("the build accepted a pattern that names no file")
	}
	want := "the pattern static/*.txt names no file"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// TestEmbedRejectsAPatternOutsideThePackage keeps a directive inside the
// directory of its package.
func TestEmbedRejectsAPatternOutsideThePackage(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	"embed"
	"fmt"
)

//go:embed ../secret.txt
var assets embed.FS

func main() { fmt.Println(assets) }
`, nil)
	if err == nil {
		t.Fatal("the build accepted a pattern outside the package")
	}
	want := "the pattern ../secret.txt is outside the rules"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}

// TestEmbedRejects2FilesForAString follows the go command, which gives a
// string or a byte slice the content of 1 file.
func TestEmbedRejects2FilesForAString(t *testing.T) {
	err := buildEmbed(t, `package main

import (
	_ "embed"
	"fmt"
)

//go:embed static
var text string

func main() { fmt.Println(text) }
`, map[string]string{"static/a.txt": "a", "static/b.txt": "b"})
	if err == nil {
		t.Fatal("the build accepted 2 files for a string")
	}
	want := "takes 1 file, and the patterns name 2"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
}
