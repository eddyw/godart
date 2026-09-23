package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewApp(t *testing.T) {
	if _, err := exec.LookPath("flutter"); err != nil {
		t.Skip("Flutter SDK is unavailable")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join(root, "testdata/flutter/starter_test.dart"))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODART_ROOT", root)
	t.Chdir(dir)
	if err := newApp([]string{"starter"}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(dir, "starter"))
	a, err := findApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.gen(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("test", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("test/starter_test.dart", fixture, 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"analyze", "lib", "test"}, {"test", "test/starter_test.dart"}} {
		cmd := exec.Command("flutter", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("flutter %v: %v\n%s", args, err, out)
		}
	}
}
