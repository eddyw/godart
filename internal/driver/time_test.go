package driver_test

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/load"
)

func TestMonotonicClock(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src/time")
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := load.Filter{}
	for _, path := range files {
		switch filepath.Base(path) {
		case "mono_test.go":
			sources[path] = []byte(monotonicContracts)
		case "export_test.go":
			sources[path] = []byte(timeExportSource(t, path, []string{"SetMono", "GetMono"}))
		default:
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			sources[path] = []byte("package " + f.Name.Name + "\n")
		}
	}
	native := t.TempDir()
	replacement := filepath.Join(native, "mono_test.go")
	writeFile(t, replacement, monotonicContracts)
	overlay, err := json.Marshal(map[string]any{"Replace": map[string]string{filepath.Join(dir, "mono_test.go"): replacement}})
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(native, "overlay.json")
	writeFile(t, config, string(overlay))
	license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(native, "LICENSE"), string(license))
	const filter = "^(TestMonotonicWallChange|TestMonotonicPreservation|TestMonotonicBounds)$"
	t.Log(run(t, dir, "go", "test", "-overlay="+config, "-run", filter, "-count=1", "-v", "."))
	out := t.TempDir()
	stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TestSources: sources, TrimPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Binaries) != 1 {
		t.Fatalf("got %d binaries", len(stats.Binaries))
	}
	prepareDart(t, out)
	text := runProgram(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.run="+filter)
	t.Log(text)
	for _, name := range []string{"TestMonotonicWallChange", "TestMonotonicPreservation", "TestMonotonicBounds"} {
		if !strings.Contains(text, "--- PASS: "+name) {
			t.Fatalf("missing %s\n%s", name, text)
		}
	}
}

const monotonicContracts = `package time_test

import (
	. "time"
	"strings"
	"testing"
)

func TestMonotonicWallChange(t *testing.T) {
	a := Date(1885, January, 1, 0, 1, 40, 0, UTC)
	SetMono(&a, 1000)
	b := a
	b = b.Add(-Minute)
	SetMono(&b, GetMono(&a)+7)
	if b.Sub(a) != 7 || !b.After(a) || !a.Before(b) || b.Compare(a) != 1 {
		t.Fatal("wall change altered monotonic comparisons")
	}
	SetMono(&b, GetMono(&a))
	if !a.Equal(b) || a.Compare(b) != 0 {
		t.Fatal("equal monotonic clocks differ")
	}
	b = b.Round(0)
	if a.Equal(b) || a.Sub(b) != Minute || a.Compare(b) != 1 {
		t.Fatal("mixed clocks did not use wall time")
	}
}

func TestMonotonicPreservation(t *testing.T) {
	a := Now()
	if a == a.Round(0) || a.Add(Second) == a.Add(Second).Round(0) || a.Add(Second).Sub(a) != Second {
		t.Fatal("Now or Add lost the monotonic clock")
	}
	for _, b := range []Time{a.UTC(), a.Local(), a.In(FixedZone("x", 3600)), a.AddDate(0, 0, 0), a.Round(0), a.Truncate(0), a.Round(Second), a.Truncate(Second)} {
		if b != b.Round(0) {
			t.Fatal("operation retained the monotonic clock")
		}
	}
	data, err := a.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b := a
	if err := b.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if b != b.Round(0) || !a.Equal(b) {
		t.Fatal("binary round trip retained monotonic state or lost wall time")
	}
	text, err := a.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.UnmarshalText(text); err != nil {
		t.Fatal(err)
	}
	if b != b.Round(0) || !a.Equal(b) {
		t.Fatal("text round trip differs")
	}
}

func TestMonotonicBounds(t *testing.T) {
	minimum := Date(1885, January, 1, 0, 0, 0, 0, UTC)
	a := minimum.Add((1<<33 - 1) * Second)
	SetMono(&a, 1)
	if a.Add(Second) != a.Add(Second).Round(0) {
		t.Fatal("wall overflow retained the monotonic clock")
	}
	a = minimum
	SetMono(&a, 1)
	if a.Add(-Second) != a.Add(-Second).Round(0) {
		t.Fatal("wall underflow retained the monotonic clock")
	}
	a = a.Add(Second)
	SetMono(&a, 1<<63 - 1)
	if a.Add(1) != a.Add(1).Round(0) {
		t.Fatal("monotonic overflow retained the clock")
	}
	b := a
	SetMono(&b, -1 << 63)
	if a.Sub(b) != Duration(1<<63 - 1) || b.Sub(a) != Duration(-1 << 63) {
		t.Fatal("Sub did not saturate")
	}
	if !strings.HasSuffix(b.String(), " m=-9223372036.854775808") {
		t.Fatal(b.String())
	}
	SetMono(&b, 12)
	if !strings.HasSuffix(b.String(), " m=+0.000000012") {
		t.Fatal(b.String())
	}
}
`
