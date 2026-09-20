package parity

import (
	"go/build"
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageInventory(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("VERSION", "go1.27.1\n")
	write("src/cmd/internal/testdir/testdir_test.go", `package testdir_test; var dirs=[]string{".","other"}`)
	write("test/run.go", "// run\npackage main\nfunc main() {}\n")
	write("test/run.out", "expected\n")
	write("test/suffix_windows.go", "// run\npackage main\nfunc main() {}\n")
	write("test/constraints.go", "//go:build dart\n\n// run\npackage main\nfunc main() {}\n")
	write("test/other/directory.go", "// rundir -G=3\npackage ignored\n")
	write("test/other/directory.dir/main.go", "package main\nfunc main() {}\n")
	ctx := build.Default
	ctx.GOROOT = root
	ctx.GOOS = "linux"
	inv, err := DiscoverLanguage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Cases) != 4 {
		t.Fatalf("got %d cases, want 4", len(inv.Cases))
	}
	cases := map[string]LanguageCase{}
	for _, c := range inv.Cases {
		cases[c.Name] = c
	}
	if c := cases["suffix_windows"]; !c.NativeEligible || c.GartEligible {
		t.Fatalf("runner suffix rules lost: %+v", c)
	}
	if c := cases["constraints"]; c.NativeEligible || !c.GartEligible || c.Adapter != "missing" {
		t.Fatalf("constraint distinction lost: %+v", c)
	}
	if c := cases["other/directory"]; c.Action != "rundir" || c.Recipe != "rundir -G=3" || len(c.Files) != 2 || c.Adapter != "missing" {
		t.Fatalf("companion or recipe lost: %+v", c)
	}
	if c := cases["run"]; c.Adapter != "single-file" || len(c.Files) != 2 {
		t.Fatalf("run adapter or expected output lost: %+v", c)
	}
	write("test/other/directory.dir/main.go", "package main\nfunc main() { println(1) }\n")
	changed, err := DiscoverLanguage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if inv.SourceSHA256 == changed.SourceSHA256 {
		t.Fatal("companion source change escaped identity")
	}
}

func TestLanguagePolicy(t *testing.T) {
	inv, err := DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var sources map[string]struct {
		SourceSHA256 string `json:"source_sha256"`
		RunnerSHA256 string `json:"runner_sha256"`
		Recipes      int    `json:"recipes"`
	}
	readManifest(t, "language-sources.json", &sources)
	reviewed, ok := sources[inv.GoVersion]
	if !ok || reviewed.SourceSHA256 != inv.SourceSHA256 || reviewed.RunnerSHA256 != inv.Runner.SHA256 || reviewed.Recipes != len(inv.Cases) {
		t.Fatal("upstream language sources changed without review")
	}
	policy, err := ReadLanguagePolicy()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]LanguageCase{}
	for _, c := range inv.Cases {
		cases[c.Name] = c
	}
	for name, skip := range policy.Skips {
		if cases[name].Adapter != "single-file" || skip.Reason == "" {
			t.Errorf("invalid language omission: %s", name)
		}
		switch skip.Classification {
		case "implementation-gap", "test-adapter", "target-limit":
		default:
			t.Errorf("unclassified omission: %s", name)
		}
		if _, ok := policy.Rejections[name]; ok {
			t.Errorf("duplicate skip and rejection: %s", name)
		}
	}
	for name, diagnostics := range policy.Rejections {
		if cases[name].Adapter != "single-file" || len(diagnostics) == 0 {
			t.Errorf("invalid language rejection: %s", name)
		}
	}
}
