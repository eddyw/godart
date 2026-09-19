package notices

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sub", "pkg")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"LICENSE": "root license\n", "PATENTS": "patent grant\n",
		"sub/NOTICE.txt": "additional attribution\n", "sub/pkg/code.go": "package pkg\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := Files(dir, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 || files["sub/NOTICE.txt"] != "additional attribution\n" {
		t.Fatalf("unexpected notices: %v", files)
	}
	if _, err := Files(filepath.Dir(root), root); err == nil {
		t.Fatal("accepted a directory outside the source root")
	}
}

func TestCompilerNotices(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-f", `{{with .Module}}{{.Path}} {{.Version}}{{end}}`, "./cmd/gart", "./cmd/glutter")
	cmd.Dir = "../.."
	data, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(Compiler, line+" / LICENSE\n") {
			t.Errorf("missing current dependency notice: %s; run go generate ./internal/notices", line)
		}
	}
	for _, name := range []string{"LICENSE", "runtime/goruntime/LICENSE", "flutter/goflutter/LICENSE", "flutter/ui/LICENSE"} {
		data, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != Project {
			t.Errorf("project license differs: %s", name)
		}
	}
	if !strings.Contains(Compiler, Project) || !strings.Contains(Compiler, "Go SDK / LICENSE\n") {
		t.Fatal("compiler bundle omits project or Go notices")
	}
}
