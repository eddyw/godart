package driver_test

import (
	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestReflectConversions(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/reflect")
	declarations := []string{"V", "integer", "EmptyInterfaceV", "ReaderV", "ReadWriterV", "Empty", "MyStruct", "MyStruct1", "MyStruct2", "MyString", "MyBytes", "MyBytesArrayPtr0", "MyBytesArrayPtr", "MyBytesArray0", "MyBytesArray", "MyRunes", "MyFunc", "MyByte", "MyRune", "MyBytes2", "MyRunes2", "IntChan", "IntChanRecv", "IntChanSend", "BytesChan", "BytesChanRecv", "BytesChanSend", "convertTests", "TestConvert", "TestConvertPanic", "TestIssue80332", "issue80332ZeroLen", "issue80332ZeroLen.Len", "shouldPanic", "TestConvertibleTo", "TestIssue80332Others", "issue80332FakeChan", "issue80332FakeChan.Kind", "issue80332FakeChan.ChanDir", "issue80332FakeMap", "issue80332FakeMap.Kind"}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		switch filepath.Base(path) {
		case "all_test.go":
			sources[path] = []byte(selectedUpstreamSource(t, path, declarations))
		case "export_test.go":
			source := selectedUpstreamSource(t, path, []string{"MakeRO", "IsRO"})
			if strings.Count(source, "v.flag") != 2 || strings.Count(source, "flagStickyRO") != 2 {
				t.Fatal("upstream read-only helpers changed")
			}
			source = strings.ReplaceAll(source, "v.flag", "v.ro")
			source = strings.ReplaceAll(source, "flagStickyRO", "roSticky")
			sources[path] = []byte(source)
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
		t.Fatalf("got %d binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^(TestConvert|TestConvertPanic|TestIssue80332|TestConvertibleTo|TestIssue80332Others)$"
	args := []string{"-test.v", "-test.short=" + strconv.FormatBool(testing.Short()), "-test.run=" + filter}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "reflect-conversion", "reflect", filter, target, dir, result.Text, result.Exit)
		})
	}
}
