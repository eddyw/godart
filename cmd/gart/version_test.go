package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "gart")
	if text, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, text)
	}
	dir := t.TempDir()
	t.Run("without SDKs", func(t *testing.T) {
		cmd := exec.Command(binary, "version")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH=")
		text, err := cmd.CombinedOutput()
		if err != nil || string(text) != "gart "+version+"\n" {
			t.Fatalf("version: %v\n%s", err, text)
		}
	})
	if _, err := exec.LookPath("dart"); err != nil {
		t.Skip("verbose diagnostics require Dart")
	}
	for _, language := range []string{"", "1.21", "1.27"} {
		t.Run("module "+language, func(t *testing.T) {
			source := "module example.com/versions\n"
			if language != "" {
				source += "\ngo " + language + "\n"
			}
			module := filepath.Join(dir, "go.mod")
			if err := os.WriteFile(module, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(binary, "version", "-v", "-frontend=source")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
			text, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("diagnostics: %v\n%s", err, text)
			}
			active := exec.Command("go", "env", "GOVERSION")
			active.Dir, active.Env = dir, cmd.Env
			goVersion, err := active.Output()
			if err != nil {
				t.Fatal(err)
			}
			declared := language
			if declared == "" {
				declared = "not declared"
			}
			for _, want := range []string{
				"compiler build Go: " + runtime.Version() + "\n",
				"active Go: " + strings.TrimSpace(string(goVersion)) + "\n",
				"module: example.com/versions\n",
				"module Go directive: " + declared + "\n",
				"Dart SDK: ",
				"Dart frontend: source (requested source)\n",
			} {
				if !strings.Contains(string(text), want) {
					t.Errorf("missing %q in:\n%s", want, text)
				}
			}
			if strings.Contains(string(text), "warning") {
				t.Errorf("unexpected warning:\n%s", text)
			}
			if after, err := os.ReadFile(module); err != nil || string(after) != source {
				t.Fatalf("diagnostics changed go.mod: %v\n%s", err, after)
			}
		})
	}
	t.Run("outside module", func(t *testing.T) {
		cmd := exec.Command(binary, "version", "-v", "-frontend=source")
		cmd.Dir = t.TempDir()
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
		text, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(text), "module: none\n") {
			t.Fatalf("outside module: %v\n%s", err, text)
		}
	})
	t.Run("missing Dart", func(t *testing.T) {
		goPath, err := exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binary, "version", "-v")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(goPath), "GOWORK=off", "GOTOOLCHAIN=local")
		text, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(text), "dart") {
			t.Fatalf("missing Dart: %v\n%s", err, text)
		}
	})
}
