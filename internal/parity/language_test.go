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
	write("test/compile.go", "// compile\npackage p\n")
	write("test/reject.go", "// errorcheck\npackage p\n")
	write("test/rejectflags.go", "// errorcheck -0\npackage p\n")
	write("test/compileflags.go", "// compile -N\npackage p\n")
	write("test/generated.go", "// runoutput\npackage main\n")
	write("test/generatedflags.go", "// runoutput extra.go\npackage main\n")
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
	if len(inv.Cases) != 10 {
		t.Fatalf("got %d cases, want 10", len(inv.Cases))
	}
	cases := map[string]LanguageCase{}
	for _, c := range inv.Cases {
		cases[c.Name] = c
	}
	if cases["reject"].Adapter != "reject-file" || cases["rejectflags"].Adapter != "missing" {
		t.Fatal("the rejection adapter must not discard recipe flags")
	}
	if cases["compile"].Adapter != "compile-file" || cases["compileflags"].Adapter != "missing" {
		t.Fatal("the compile adapter must not discard recipe flags")
	}
	if cases["generated"].Adapter != "generated-file" || cases["generatedflags"].Adapter != "missing" {
		t.Fatal("the generator adapter must not discard recipe arguments")
	}
	if c := cases["suffix_windows"]; !c.NativeEligible || c.GartEligible {
		t.Fatalf("runner suffix rules lost: %+v", c)
	}
	if c := cases["constraints"]; c.NativeEligible || !c.GartEligible || c.Adapter != "single-file" {
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
	var targets []string
	readManifest(t, "language-targets.json", &targets)
	seen := map[string]bool{}
	for _, name := range targets {
		fixture, ok := cases[name]
		if !ok || seen[name] || fixture.Adapter != "single-file" || fixture.Exclusion != nil || fixture.Rejection != nil {
			t.Errorf("invalid target fixture: %s", name)
		}
		seen[name] = true
	}

	for name, skip := range policy.Skips {
		if (cases[name].Adapter != "single-file" && cases[name].Adapter != "reject-file") || skip.Reason == "" {
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
		if (cases[name].Adapter != "single-file" && cases[name].Adapter != "compile-file") || len(diagnostics) == 0 {
			t.Errorf("invalid language rejection: %s", name)
		}
	}
}
