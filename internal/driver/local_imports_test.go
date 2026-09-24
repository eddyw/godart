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

func TestImportPrefixes(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	for name, source := range map[string]string{
		"go.mod":              "module example.com/importprefix\n\ngo 1.27\n",
		"poll/value.go":       "package poll\nfunc Value() int { return 1 }\n",
		"other/poll/value.go": "package poll\nfunc Value() int { return 2 }\n",
		"println/value.go":    "package println\nfunc Value() int { return 4 }\n",
		"main.go": `package main
import(p "example.com/importprefix/poll"; q "example.com/importprefix/other/poll"; lines "example.com/importprefix/println")
func main(){
 sum:=p.Value()+q.Value()+lines.Value()
 for i:=0;i<3;i++{sum+=i}
 println(sum)
}
`,
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, path, source)
	}
	want := programOutcome(t, dir, "go", "run", ".")
	if want.Exit != 0 {
		t.Fatalf("native Go failed: %+v", want)
	}
	if err := driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)}); err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", filepath.Join(out, "bin/main.dart"))
			} else {
				result = runUpstreamTarget(t, target, out, "bin/main.dart", dir, []string{})
			}
			if result != want {
				t.Fatalf("Go:\n%s\nGart (exit %d):\n%s", want.Text, result.Exit, result.Text)
			}
		})
	}
}
