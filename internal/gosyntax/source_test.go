package syntax

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSourceMatchesSDK(t *testing.T) {
	data, err := os.ReadFile("ORIGIN.json")
	if err != nil {
		t.Fatal(err)
	}
	var origin struct {
		Source string
		Files  map[string]string
	}
	if err := json.Unmarshal(data, &origin); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(runtime.GOROOT(), origin.Source)
	files, err := filepath.Glob(filepath.Join(source, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file, "_test.go") && origin.Files[filepath.Base(file)] == "" {
			t.Errorf("SDK parser file is absent: %s", file)
		}
	}
	for name, want := range origin.Files {
		upstream := filepath.Join(source, name)
		if name == "LICENSE" {
			upstream = filepath.Join(runtime.GOROOT(), name)
		}
		for _, path := range []string{name, upstream} {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Error(err)
				continue
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
				t.Errorf("parser source changed: %s; review the SDK update", path)
			}
		}
	}
}
