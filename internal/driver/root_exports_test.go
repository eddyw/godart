package driver_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestRootNativeExports(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("root descriptors require a native filesystem")
	}
	dir := filepath.Join(runtime.GOROOT(), "src/os")
	tests := []string{"TestRootChtimes", "TestRootMultiOpen", "TestRootMultiChmod", "TestRootMultiCreate", "TestRootMultiLink", "TestRootMultiLstat", "TestRootMultiMkdir", "TestRootMultiMkdirAllShallow", "TestRootMultiMkdirAllDeep", "TestRootMultiRename", "TestRootMultiReadFile", "TestRootMultiStat", "TestRootMultiRemove", "TestRootMultiRemoveAll", "TestRootMultiChtimes", "TestRootMultiReadlink", "TestRootMultiWriteFile", "TestRootMultiOpenFile"}
	declarations := append([]string{"makefs", "hasLink", "rootTest", "rootTestCases", "errEndsTest", "rootMultiTest", "testVerbose", "errSkipRootConsistencyCheck", "runRootMultiTest", "runRootMultiTest2", "errAny", "runRootMultiTestDescs", "newRootTest", "testFileKind", "testFileUnused", "testFileRef", "testFileRefs", "testFileLimitedRefs", "testFileDesc", "rootComprehensive", "allTestFileDescs", "dirTreeContents", "testRootMultiMkdirAll"}, tests...)
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		switch filepath.Base(path) {
		case "root_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, declarations))
		case "os_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"hasNoatime"}))
		case "export_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, []string{"Atime", "ErrPathEscapes"}))
		case "export_unix_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, filepath.Join(dir, "stat_"+runtime.GOOS+".go"), []string{"atime"}))
		default:
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
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d test binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	filter := "^(" + strings.Join(tests, "|") + ")$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = upstreamOutcome(t, dir, nil, runLimit, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "root-native-exports", "os", filter, target, dir, result.Text, result.Exit, true)
		})
	}
}
