package driver_test

import (
	"archive/zip"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestTimeLayout(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/time")
	names := map[string][]string{
		"mono_test.go":     nil,
		"tick_test.go":     {"TestTicker", "TestTickerStopWithDirectInitialization", "TestTeardown", "TestTick", "TestNewTickerLtZeroDuration", "TestTickerResetLtZeroDuration", "TestManualTicker", "TestAfterTimes", "TestTickTimes", "TestChan", "timer", "tickerTimer", "testTimerChan"},
		"sleep_test.go":    {"TestAfterFuncStarvation", "TestAfterQueuing", "TestTimerStopStress", "TestSleepZeroDeadlock", "TestIssue5745", "TestTimerModifiedEarlier", "TestAdjustTimers", "TestMultiWakeupTimer", "testAfterQueuing", "testAfterQueuing1", "slots", "afterResult", "await", "TestAfterTick", "TestAfterStop", "TestReset", "newTimerFunc", "testAfterStop", "testReset", "TestStopResult", "TestResetResult", "testStopResetResult", "TestAfterFunc", "TestTickerStress", "TestTickerConcurrentStress", "TestOverflowSleep", "TestZeroTimerResetPanics", "TestZeroTimerStopPanics", "checkZeroPanicString"},
		"zoneinfo_test.go": {"TestVersion3", "TestFirstZone", "TestEarlyLocation", "TestTzset", "TestTzsetName", "TestTzsetOffset", "TestMalformedTZData", "TestLocationNames"},
		"format_test.go":   {"TestLoadLocationZipFile", "nextStdChunkTests", "TestNextStdChunk", "TestAppendInt", "ParseTimeZoneTest", "parseTimeZoneTests", "TestParseTimeZone", "TestQuote"},
		"time_test.go":     {"TestTimeIsDST", "TestTimeWithZoneTransition", "TestZoneBounds", "parsedTime", "TimeTest", "utctests", "TestTimeAddSecOverflow", "TestUnmarshalInvalidTimes", "minDuration", "maxDuration", "subTests", "TestSub", "daysInTests", "TestDaysIn", "TestConcurrentTimerReset", "TestConcurrentTimerResetStop"},
		"export_test.go":   {"DisablePlatformSources", "SetMono", "GetMono", "UnixToInternal", "NextStdChunk", "StdChunkNames", "AppendInt", "ParseTimeZone", "Quote", "Tzset", "TzsetName", "TzsetOffset"},
		"internal_test.go": {"origPlatformZoneSources", "disablePlatformSources", "NotMonoNegativeTime", "MinMonoTime", "MaxMonoTime", "DaysIn"},
	}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		if declarations, ok := names[filepath.Base(path)]; ok {
			if filepath.Base(path) == "export_test.go" || filepath.Base(path) == "internal_test.go" {
				sources[path] = []byte(timeExportSource(t, path, declarations))
			} else {
				sources[path] = []byte(selectedUpstreamSource(t, path, declarations))
			}
		} else {
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + f.Name.Name + "\n")
		}
	}
	zones, err := zip.OpenReader(filepath.Join(runtime.GOROOT(), "lib/time/zoneinfo.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer zones.Close()
	zone, err := zones.Open("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(zone)
	zone.Close()
	if err != nil {
		t.Fatal(err)
	}
	zonePath := filepath.Join(dir, "zoneinfo_test.go")
	zoneSource := strings.Replace(string(sources[zonePath]), "package time_test", "package time_test\nimport _ \"time/tzdata\"", 1)
	zoneSource += "\nfunc init() { loc, err := time.LoadLocationFromTZData(\"Local\", []byte(" + strconv.Quote(string(data)) + ")); if err != nil { panic(err) }; *time.Local = *loc }\n"
	sources[zonePath] = []byte(zoneSource)
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	const filter = "^(TestHasMonotonicClock|TestMonotonicAdd|TestMonotonicSub|TestMonotonicOverflow|TestMonotonicString|TestTimeAddSecOverflow|TestUnmarshalInvalidTimes|TestNextStdChunk|TestAppendInt|TestParseTimeZone|TestQuote|TestSub|TestDaysIn|TestTzset|TestTzsetName|TestTzsetOffset|TestMalformedTZData|TestLocationNames|TestAfterFunc|TestTickerStress|TestTickerConcurrentStress|TestOverflowSleep|TestZeroTimerResetPanics|TestZeroTimerStopPanics|TestConcurrentTimerReset|TestConcurrentTimerResetStop|TestChan|TestStopResult|TestResetResult|TestTicker|TestTickerStopWithDirectInitialization|TestTeardown|TestTick|TestNewTickerLtZeroDuration|TestTickerResetLtZeroDuration|TestManualTicker|TestAfterTimes|TestTickTimes|TestAfterTick|TestAfterStop|TestReset|TestAfterFuncStarvation|TestAfterQueuing|TestTimerStopStress|TestSleepZeroDeadlock|TestIssue5745|TestTimerModifiedEarlier|TestAdjustTimers|TestMultiWakeupTimer|TestLoadLocationZipFile|TestTimeIsDST|TestTimeWithZoneTransition|TestZoneBounds|TestVersion3|TestFirstZone|TestEarlyLocation)$"
	args := upstreamTestArgs("-test.run=" + filter)
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) {
			var result programResult
			if target == "vm" {
				result = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
			} else {
				result = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
			}
			compareUpstream(t, "time-layout", "time", filter, target, dir, result.Text, result.Exit)
		})
	}
}

func timeExportSource(t *testing.T, path string, names []string) string {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, path, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	out.Write(source[:fs.Position(file.Name.End()).Offset])
	out.WriteByte('\n')
	found := 0
	for _, name := range names {
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == name {
				fmt.Fprintf(&out, "//line %s:%d\n", filepath.Base(path), fs.Position(function.Pos()).Line)
				out.Write(source[fs.Position(function.Pos()).Offset:fs.Position(function.End()).Offset])
				out.WriteByte('\n')
				found++
				continue
			}
			group, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, item := range group.Specs {
				value, ok := item.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, id := range value.Names {
					if id.Name != name {
						continue
					}
					if len(value.Names) != 1 || len(value.Values) != 1 {
						t.Fatalf("upstream helper %s changed shape", name)
					}
					fmt.Fprintf(&out, "//line %s:%d\n%s ", filepath.Base(path), fs.Position(value.Pos()).Line, group.Tok)
					out.Write(source[fs.Position(value.Pos()).Offset:fs.Position(value.End()).Offset])
					out.WriteByte('\n')
					found++
				}
			}
		}
	}
	if found != len(names) {
		t.Fatalf("got %d time helpers, want %d", found, len(names))
	}
	return out.String()
}
