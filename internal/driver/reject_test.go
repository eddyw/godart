package driver_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// TestRejectReportsTheImport proves that the report names an import outside the
// surface, and not the type errors of the package behind it.
//
// The overlay of another package of the surface breaks such a package. The load
// holds each error of it back, thus the verifier reports the real cause.
func TestRejectReportsTheImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module reject\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "os/user"

func main() {
	user.Current()
}
`)
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	err = driver.Build(driver.Options{
		Dir:      dir,
		Patterns: []string{"./..."},
		Out:      t.TempDir(),
		Runtime:  runtime,
	})
	if err == nil {
		t.Fatal("the build accepted an import outside the surface")
	}
	want := "the package os/user is outside the supported surface"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the report holds no %q:\n%v", want, err)
	}
	if strings.Contains(err.Error(), "/os/user/") {
		t.Errorf("the report holds a type error of the package:\n%v", err)
	}
}
