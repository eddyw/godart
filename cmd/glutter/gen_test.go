package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
