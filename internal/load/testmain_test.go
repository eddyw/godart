package load_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/load"
)

func TestExternalOnlyFuzzCorpus(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":                       "module example.com/external\n\ngo 1.27\n",
		"external_test.go":             "package external_test\nimport \"testing\"\nfunc FuzzText(f *testing.F) { f.Fuzz(func(t *testing.T, s string) {}) }\n",
		"testdata/fuzz/FuzzText/input": "go test fuzz v1\nstring(\"corpus\")\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := load.LoadTests(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Binaries) != 1 {
		t.Fatalf("got %d binaries, want 1", len(result.Binaries))
	}
	bin := result.Binaries[0]
	if bin.Dir != dir {
		t.Errorf("directory = %q, want %q", bin.Dir, dir)
	}
	if !strings.Contains(string(bin.Source), `Name: "input", Data: "go test fuzz v1\nstring(\"corpus\")\n"`) {
		t.Fatalf("missing corpus in entry point:\n%s", bin.Source)
	}
}

func loadTests(t *testing.T) *load.Result {
	t.Helper()
	dir, err := filepath.Abs("../../testdata/tests/basic")
	if err != nil {
		t.Fatal(err)
	}
	result, err := load.LoadTests(dir, "./...")
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestLoadTestsFindsTheBinary(t *testing.T) {
	result := loadTests(t)
	if len(result.Binaries) != 1 {
		t.Fatalf("got %d test binaries, want 1", len(result.Binaries))
	}
	bin := result.Binaries[0]
	if bin.Under != "example.com/basic/calc" {
		t.Errorf("got the package %q under test, want %q", bin.Under, "example.com/basic/calc")
	}
	want := []string{"TestAdd", "TestDouble", "TestSkipped", "TestCleanup",
		"TestHelperPosition", "TestJoin"}
	if strings.Join(bin.Tests, ",") != strings.Join(want, ",") {
		t.Errorf("got the tests %v, want %v", bin.Tests, want)
	}
	wantBench := []string{"BenchmarkAdd", "BenchmarkDouble", "BenchmarkLoop"}
	if strings.Join(bin.Benchmarks, ",") != strings.Join(wantBench, ",") {
		t.Errorf("got the benchmarks %v, want %v", bin.Benchmarks, wantBench)
	}
	if len(bin.FuzzTargets) != 0 {
		t.Errorf("got the fuzz targets %v, want none", bin.FuzzTargets)
	}
}

func TestLoadTestsWritesTheEntryPoint(t *testing.T) {
	result := loadTests(t)
	bin := result.Binaries[0]
	source := string(bin.Source)
	for _, want := range []string{
		"package main",
		`gartTest "example.com/basic/calc"`,
		`gartXtest "example.com/basic/calc_test"`,
		`{Name: "TestAdd", F: gartTest.TestAdd}`,
		`{Name: "TestJoin", F: gartXtest.TestJoin}`,
		`{Name: "BenchmarkAdd", F: gartTest.BenchmarkAdd}`,
		"testing.MainWith(",
		"func(m *testing.M) { m.Run() }",
	} {
		if !strings.Contains(source, want) {
			t.Errorf("the entry point holds no %q:\n%s", want, source)
		}
	}
	if string(result.Sources[bin.File]) != source {
		t.Error("the overlay does not hold the entry point")
	}
}

func TestLoadTestsGivesTheVariants(t *testing.T) {
	result := loadTests(t)
	want := map[string]int{
		"example.com/basic/calc":                                    1,
		"example.com/basic/calc [example.com/basic/calc.test]":      3,
		"example.com/basic/calc_test [example.com/basic/calc.test]": 1,
		"example.com/basic/calc.test":                               1,
	}
	got := map[string]int{}
	for _, p := range result.Packages {
		if _, ok := want[p.ID]; ok {
			got[p.ID] = len(p.Syntax)
		}
	}
	for id, count := range want {
		if got[id] != count {
			t.Errorf("the package %q holds %d syntax trees, want %d", id, got[id], count)
		}
	}
	if bin := result.Binary("example.com/basic/calc.test"); bin == nil {
		t.Error("the result gives no binary for the identifier of the entry point")
	}
}
