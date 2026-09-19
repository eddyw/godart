package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaintRejectsBlockingCallback(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	a := &app{dir: t.TempDir(), root: root, module: "go", out: "gen"}
	if err := os.MkdirAll(a.modulePath(), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod": fmt.Sprintf("module example.com/paint\n\ngo 1.27.0\nrequire gart.dev/ui v0.0.0\nreplace gart.dev/ui => %s\n", filepath.ToSlash(filepath.Join(root, "flutter/ui"))),
		"app.go": `package app
import "gart.dev/ui"
func Tree() ui.Node {
 ch := make(chan int)
 return ui.CustomPaint{Painter: ui.P(ui.Painter{Paint: func(c ui.Canvas, width, height float64) { <-ch }})}
}`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(a.modulePath(), name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	err = a.build(false)
	if err == nil || !strings.Contains(err.Error(), "ui.PaintFunc is a synchronous position") {
		t.Fatalf("expected synchronous paint rejection, got %v", err)
	}
}

func TestBuildDoesNotValidateNativeTests(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	a := &app{dir: t.TempDir(), root: root, module: "go", out: "gen"}
	if err := os.MkdirAll(a.modulePath(), 0755); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"go.mod":       "module example.com/glutter-build\n\ngo 1.27.0\n",
		"main.go":      "package main\nfunc main() {}\n",
		"main_test.go": "package main\nfunc TestUnfinished(\n",
	} {
		if err := os.WriteFile(filepath.Join(a.modulePath(), name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, command := range []string{"vet", "test"} {
		cmd := exec.Command("go", command, "./...")
		cmd.Dir = a.modulePath()
		if err := cmd.Run(); err == nil {
			t.Fatalf("go %s accepted the unfinished test fixture", command)
		}
	}
	if err := a.build(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.outPath(), "bin", "main.dart")); err != nil {
		t.Fatal(err)
	}
}
