package chdirrename

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreRenamedDirectory(t *testing.T) {
	root := t.TempDir()
	start, other, moved := filepath.Join(root, "start"), filepath.Join(root, "other"), filepath.Join(root, "moved")
	for _, dir := range []string{start, other} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(start)
	t.Run("inner", func(t *testing.T) {
		t.Chdir(other)
		if err := os.Rename(start, moved); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(start, 0700); err != nil {
			t.Fatal(err)
		}
	})
	actual, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(moved)
	if err != nil {
		t.Fatal(err)
	}
	if actual != want {
		t.Fatal("cleanup restored a replacement directory")
	}
}
