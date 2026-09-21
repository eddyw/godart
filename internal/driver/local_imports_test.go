package driver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
)

func TestLocalModuleImports(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"app/go.mod":       "module m\n\ngo 1.27\n\nrequire n v0.0.0\nreplace n => ../dep\n",
		"dep/go.mod":       "module n\n\ngo 1.27\n",
		"dep/value.go":     "package n\nconst Value = 7\n",
		"app/sub/value.go": "package sub\nimport \"n\"\nconst Value = n.Value\n",
		"app/local_test.go": `package m
import ("math"; "m/sub"; "testing")
func TestValue(t *testing.T) { if math.Abs(-sub.Value) != 7 { t.Fatal(sub.Value) } }
`,
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, path, source)
	}
	dir := filepath.Join(root, "app")
	for _, pattern := range []string{".", "local_test.go"} {
		t.Run(pattern, func(t *testing.T) {
			native := programOutcome(t, dir, "go", "test", "-v", pattern)
			var output strings.Builder
			code, err := driver.Test(driver.Options{Dir: dir, Patterns: []string{pattern}, Out: t.TempDir(), Runtime: runtimeDir(t)}, driver.TestOptions{Verbose: true}, &output)
			if err != nil || native.Exit != 0 || code != 0 {
				t.Fatalf("Go: %+v\nGart: %d %v\n%s", native, code, err, output.String())
			}
			if strings.TrimSpace(normalize(native.Text)) != strings.TrimSpace(normalize(output.String())) {
				t.Fatalf("Go:\n%s\nGart:\n%s", native.Text, output.String())
			}
		})
	}
}
