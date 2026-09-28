package driver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

// TestDeterministic proves that equal input gives byte equal output. The
// incremental compiler of Dart needs this rule, and so does the cache.
func TestDeterministic(t *testing.T) {
	for _, program := range []string{"multi", "funcpc", "literalnames"} {
		t.Run(program, func(t *testing.T) {
			source, err := filepath.Abs(filepath.Join("../../testdata/programs", program))
			if err != nil {
				t.Fatal(err)
			}
			first := buildOnce(t, source)
			second := buildOnce(t, source)
			if len(first) == 0 {
				t.Fatal("the build wrote no file")
			}
			if len(first) != len(second) {
				t.Fatalf("the 2 builds wrote %d files and %d files", len(first), len(second))
			}
			for name, text := range first {
				if second[name] != text {
					t.Errorf("the file %s differs between 2 builds", name)
				}
			}

		})
	}
}

// TestGolden compares the emitted Dart with the text of the golden file. Run
// `go test ./internal/driver -update` to write the golden files again.
func TestGolden(t *testing.T) {
	source, err := filepath.Abs("../../testdata/programs/multi")
	if err != nil {
		t.Fatal(err)
	}
	files := buildOnce(t, source)
	root, err := filepath.Abs("../../testdata/golden/multi")
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range files {
		if !strings.HasSuffix(name, ".dart") {
			continue
		}
		path := filepath.Join(root, name)
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(text), 0o666); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("the golden file %s is missing: %v", name, err)
			continue
		}
		if string(want) != text {
			t.Errorf("the file %s differs from the golden file", name)
		}
	}
}

func buildOnce(t *testing.T, source string) map[string]string {
	t.Helper()
	runtime, err := filepath.Abs("../../runtime/goruntime")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := driver.Build(driver.Options{
		Dir:      source,
		Patterns: []string{"./..."},
		Out:      out,
		Runtime:  runtime,
		Format:   true,
		NoCache:  true,
		TrimPath: true,
	}); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	err = filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(out, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, ".gart") || rel == "pubspec.yaml" {
			return nil
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = string(text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
