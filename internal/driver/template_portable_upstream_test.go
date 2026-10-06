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

func TestTemplatePortableOriginal(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/text/template")
	names := map[string][]string{
		"example_test.go":     nil,
		"examplefunc_test.go": nil,
		"exec_test.go": {
			"T", "U", "W", "S", "I", "myError", "cmpTest", "cmpTests", "TestComparison",
			"execErrorText", "TestExecError", "CustomError", "CustomError.Error", "TestExecError_CustomError",
			"TestJSEscaping", "Tree", "treeTemplate", "TestTree", "TestEvalFieldErrors",
			"debug", "testTemplates", "alwaysErrorText", "alwaysError", "ErrorWriter", "ErrorWriter.Write",
			"funcNameTestFunc", "testBadFuncName", "V", "V.String",
			"TestExecuteOnNewTemplate", "TestMessageForExecuteEmpty", "TestFinalForPrintf", "TestMissingMapKey",
			"TestUnterminatedStringError", "TestExecuteGivesExecError", "TestGoodFuncNames", "TestBadFuncNames",
			"TestBlock", "TestAddrOfIndex", "TestInterfaceValues", "TestIssue31810", "TestIssue43065", "TestIssue39807", "TestIssue48215",
		},
		"multi_test.go": {
			"noError", "hasError", "multiParseTest", "multiParseTests", "cloneText1", "cloneText2", "cloneText3",
			"TestClone", "TestMultiParse", "TestAddParseTree", "TestAddParseTreeToUnparsedTemplate", "TestRedefinition",
			"TestEmptyTemplateCloneCrash", "TestTemplateLookUp", "TestNew", "TestParse", "TestEmptyTemplate", "TestIssue19294", "TestAddToZeroTemplate",
		},
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
	const filter = "^(ExampleTemplate|ExampleTemplate_block|ExampleTemplate_func|ExampleTemplate_funcs|ExampleTemplate_if|TestClone|TestComparison|TestExecError|TestExecError_CustomError|TestJSEscaping|TestTree|TestEvalFieldErrors|TestExecuteOnNewTemplate|TestMessageForExecuteEmpty|TestFinalForPrintf|TestMissingMapKey|TestUnterminatedStringError|TestExecuteGivesExecError|TestGoodFuncNames|TestBadFuncNames|TestBlock|TestAddrOfIndex|TestInterfaceValues|TestIssue31810|TestIssue43065|TestIssue39807|TestIssue48215|TestMultiParse|TestAddParseTree|TestAddParseTreeToUnparsedTemplate|TestRedefinition|TestEmptyTemplateCloneCrash|TestTemplateLookUp|TestNew|TestParse|TestEmptyTemplate|TestIssue19294|TestAddToZeroTemplate)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "template-portable-original", "text/template", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
