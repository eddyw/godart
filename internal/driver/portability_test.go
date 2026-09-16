package driver_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestMathPortable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/mathportable\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "math"

func main() {
	if math.Exp(0) != 1 || math.Abs(math.Exp(1)-math.E) > 1e-15 {
		panic("incorrect exponential")
	}
}
`)
	var out bytes.Buffer
	code, err := driver.Run(driver.Options{
		Dir: dir, Patterns: []string{"./..."}, Out: t.TempDir(), Runtime: runtimeDir(t),
	}, nil, &out)
	if err != nil || code != 0 {
		t.Fatalf("portable math failed: code %d, error %v\n%s", code, err, out.String())
	}
}
