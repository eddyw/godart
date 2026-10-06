package driver_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestTemplateFilesOriginal(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/text/template")
	names := map[string][]string{
		"exec_test.go":         {"debug", "execTest", "testExecute", "add", "count", "dddArg", "echo", "makemap", "mapOfThree", "oneArg", "returnInt", "stringer", "twoArgs", "typeOf", "valueString", "vfunc", "zeroArgs", "V"},
		"multi_test.go":        {"templateFileExecTests", "TestParseFilesWithData", "TestParseGlobWithData"},
		"examplefiles_test.go": nil,
	}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		if declarations, ok := names[filepath.Base(path)]; ok {
			sources[path] = []byte(selectedUpstreamSource(t, path, declarations))
		} else {
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + file.Name.Name + "\n")
		}
	}
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	prepareDart(t, out)
	const filter = "^(TestParseFilesWithData|TestParseGlobWithData|ExampleTemplate_glob|ExampleTemplate_helpers|ExampleTemplate_share)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "template-files-original", "text/template", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
