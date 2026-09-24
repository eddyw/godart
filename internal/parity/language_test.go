package parity

import (
	"go/build"
	"os"
	"path/filepath"
	"slices"
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
	write("src/cmd/internal/testdir/testdir_test.go", `package testdir_test; var dirs=[]string{".","other"}; var types2Failures=setOf("reject.go"); var types2Failures32Bit=setOf("compile.go"); func recipe() { switch "" { case "runindir": if modVersion == "" { modVersion = "1.14" } } }`)
	write("test/module.go", "// runindir\npackage ignored\n")
	write("test/moduleflags.go", "// runindir -gomodversion 1.27\npackage ignored\n")
	write("test/moduleextra.go", "// runindir -gomodversion 1.27 -goexperiment fieldtrack\npackage ignored\n")
	write("test/module.dir/main.go", "package main\nfunc main() {}\n")
	write("test/runarguments.go", "// run arg1 arg2\npackage main\nfunc main() {}\n")
	write("test/runfiles.go", "// run ./extra.go\npackage main\nfunc main() {}\n")
	write("test/run.go", "// run\npackage main\nfunc main() {}\n")
	write("test/run.out", "expected\n")
	write("test/compile.go", "// compile\npackage p\n")
	write("test/reject.go", "// errorcheck\npackage p\n")
	write("test/rejectversion.go", "// errorcheck -lang=go1.17\npackage p\n")
	write("test/rejectversionsuccess.go", "// errorcheck -0 -lang=go1.17\npackage p\n")
	write("test/rejectpanic.go", "// errorcheck -d=panic\npackage p\n")
	write("test/build.go", "// build\npackage main\n")
	write("test/buildflags.go", "// build -race\npackage main\n")
	write("test/rejectflags.go", "// errorcheck -0\npackage p\n")
	write("test/rejectdirectory.go", "// errorcheckdir\npackage ignored\n")
	write("test/rejectdirectory.dir/a.go", "package a\n")
	write("test/rejectdirectoryflags.go", "// errorcheckdir -0\npackage ignored\n")
	write("test/rejectdirectoryflags.dir/a.go", "package a\n")
	write("test/groups.go", "// compiledir\npackage ignored\n")
	write("test/groups.dir/a.go", "package a\n")
	write("test/groupsrun.go", "// rundir\npackage ignored\n")
	write("test/groupsrun.dir/a.go", "package a\n")
	write("test/compileflags.go", "// compile -N\npackage p\n")
	write("test/generated.go", "// runoutput\npackage main\n")
	write("test/generatedflags.go", "// runoutput -gcflags=-N\npackage main\n")
	write("test/generatedreject.go", "// errorcheckoutput\npackage main\n")
	write("test/generatedrejectfiles.go", "// errorcheckoutput ./extra.go\npackage main\n")
	write("test/generatedrejectflags.go", "// errorcheckoutput -0\npackage main\n")
	write("test/generatedfiles.go", "// runoutput ./extra.go\npackage main\n")
	write("test/extra.go", "// skip\npackage main\n")
	write("test/suffix_windows.go", "// run\npackage main\nfunc main() {}\n")
	write("test/constraints.go", "//go:build dart\n\n// run\npackage main\nfunc main() {}\n")
	write("test/other/directory.go", "// rundir -G=3\npackage ignored\n")
	write("test/other/directory.dir/main.go", "package main\nfunc main() {}\n")
	write("test/linkx_run.go", "// run\npackage main\nfunc main() {}\n")
	write("test/linkx.go", "// skip\npackage main\nfunc main() {}\n")
	ctx := build.Default
	ctx.GOROOT = root
	ctx.GOOS = "linux"
	inv, err := DiscoverLanguage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Cases) != 31 {
		t.Fatalf("got %d cases, want 31", len(inv.Cases))
	}
	cases := map[string]LanguageCase{}
	for _, c := range inv.Cases {
		cases[c.Name] = c
	}
	if cases["groups"].Adapter != "directory-packages" || cases["groupsrun"].Adapter != "directory-packages" {
		t.Fatal("plain package directory recipes have no adapter")
	}
	if c := cases["module"]; c.Adapter != "directory-module" || c.ModuleGoVersion != "1.14" || len(c.Files) != 2 {
		t.Fatalf("directory module inputs lost: %+v", c)
	}
	if c := cases["moduleflags"]; c.Adapter != "directory-module" || c.ModuleGoVersion != "1.27" {
		t.Fatalf("explicit module version lost: %+v", c)
	}
	if cases["moduleextra"].Adapter != "missing" {
		t.Fatal("the directory adapter must not discard recipe options")
	}
	if !cases["reject"].ExpectedFailure {
		t.Fatal("upstream expected failure lost")
	}
	if cases["reject"].Adapter != "reject-file" || cases["rejectpanic"].Adapter != "reject-file" || cases["rejectflags"].Adapter != "missing" {
		t.Fatal("the rejection adapter must not discard recipe flags")
	}
	if cases["rejectversion"].Adapter != "reject-file" || cases["rejectversion"].ModuleGoVersion != "1.17" || cases["rejectversionsuccess"].Adapter != "diagnostic-file" || cases["rejectversionsuccess"].ModuleGoVersion != "1.17" {
		t.Fatal("the rejection adapter must preserve the language version and expected exit")
	}
	if cases["build"].Adapter != "build-file" || cases["buildflags"].Adapter != "missing" {
		t.Fatal("the build adapter must preserve recipe flags")
	}
	if cases["rejectdirectory"].Adapter != "directory-packages" || cases["rejectdirectoryflags"].Adapter != "missing" {
		t.Fatal("directory diagnostic recipes must preserve their flags")
	}
	if cases["compile"].Adapter != "compile-file" || cases["compileflags"].Adapter != "missing" {
		t.Fatal("the compile adapter must not discard recipe flags")
	}
	if cases["generatedreject"].Adapter != "generated-rejection" || cases["generatedrejectfiles"].Adapter != "generated-rejection" || cases["generatedrejectflags"].Adapter != "missing" {
		t.Fatal("generated rejection recipes must preserve their arguments")
	}
	if cases["generated"].Adapter != "generated-file" || cases["generatedflags"].Adapter != "missing" {
		t.Fatal("the generator adapter must not discard recipe arguments")
	}
	if c := cases["generatedfiles"]; c.Adapter != "generated-file" || !slices.Equal(c.GeneratorFiles, []string{"extra.go"}) || len(c.Files) != 2 || c.Files[1].Path != "extra.go" {
		t.Fatalf("generator companion source lost: %+v", c)
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
	if c := cases["runarguments"]; c.Adapter != "run-recipe" || !slices.Equal(c.Arguments, []string{"arg1", "arg2"}) {
		t.Fatalf("program arguments lost: %+v", c)
	}
	if c := cases["runfiles"]; c.Adapter != "run-recipe" || !slices.Equal(c.RunFiles, []string{"extra.go"}) || len(c.Files) != 2 || c.Files[1].Path != "extra.go" {
		t.Fatalf("run companion source lost: %+v", c)
	}
	if c := cases["run"]; c.Adapter != "single-file" || len(c.Files) != 2 {
		t.Fatalf("run adapter or expected output lost: %+v", c)
	}
	if c := cases["linkx_run"]; !c.HostTools || len(c.Companions) != 1 || c.Companions[0] != "linkx.go" || len(c.Files) != 2 || c.Files[1].Path != "linkx.go" {
		t.Fatalf("host tool inputs lost: %+v", c)
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

func TestLanguageGeneratorFiles(t *testing.T) {
	for recipe, want := range map[string][]string{
		"runoutput":                   nil,
		"errorcheckoutput":            nil,
		"errorcheckoutput ./index.go": {"index.go"},
		"runoutput ./rotate.go":       {"rotate.go"},
		"runoutput a.go b.go":         {"a.go", "b.go"},
	} {
		got, ok := languageGeneratorFiles(recipe)
		if !ok || !slices.Equal(got, want) {
			t.Fatalf("%q: %q, %t", recipe, got, ok)
		}
	}
	for _, recipe := range []string{"", "run", "errorcheckoutput -0", "errorcheckoutput ../other.go", "runoutput -gcflags=-N", "runoutput argument", "runoutput ../other.go", "runoutput /tmp/other.go", `runoutput "a.go"`, `runoutput a\b.go`} {
		if _, ok := languageGeneratorFiles(recipe); ok {
			t.Fatalf("accepted unsupported generator recipe %q", recipe)
		}
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
		if (cases[name].Adapter != "single-file" && cases[name].Adapter != "run-recipe" && cases[name].Adapter != "reject-file" && cases[name].Adapter != "directory-module" && cases[name].Adapter != "build-file") || skip.Reason == "" {
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
		if (cases[name].Adapter != "single-file" && cases[name].Adapter != "compile-file" && cases[name].Adapter != "build-file") || len(diagnostics) == 0 {
			t.Errorf("invalid language rejection: %s", name)
		}
	}
}

func TestLanguageExpectedFailures(t *testing.T) {
	source := []byte(`package testdir; var types2Failures=setOf("one.go"); var types2Failures32Bit=setOf("two.go")`)
	for _, arch := range []string{"amd64", "arm64", "386", "arm", "mips", "mipsle"} {
		got, err := languageExpectedFailures(source, arch)
		if err != nil {
			t.Fatal(err)
		}
		narrow := arch != "amd64" && arch != "arm64"
		if !got["one.go"] || got["two.go"] != narrow {
			t.Fatalf("%s: %v", arch, got)
		}
	}
	for _, invalid := range []string{`package p`, `package p; var types2Failures=[]string{"one.go"}`, `package p; var types2Failures=setOf(name)`} {
		if _, err := languageExpectedFailures([]byte(invalid), "amd64"); err == nil {
			t.Fatal("accepted an unrecognized upstream failure list")
		}
	}
}

func TestLanguageModuleVersion(t *testing.T) {
	for _, text := range []string{"1.14", "1.27"} {
		source := []byte(`package p; func f() { switch "" { case "runindir": if modVersion == "" { modVersion = "` + text + `" } } }`)
		got, err := languageModuleVersion(source)
		if err != nil || got != text {
			t.Fatalf("got %q, %v", got, err)
		}
	}
	for _, source := range []string{
		`package p`,
		`package p; func f() { switch "" { case "runindir": modVersion = "1.14" } }`,
		`package p; func f() { switch "" { case "runindir": if modVersion == "" { modVersion = other } } }`,
		`package p; func f() { switch "" { case "runindir": if modVersion == "" { modVersion = "invalid" } } }`,
	} {
		if _, err := languageModuleVersion([]byte(source)); err == nil {
			t.Fatal("accepted an unrecognized upstream module default")
		}
	}
}

func TestLanguageDirectoryAssembly(t *testing.T) {
	for _, arch := range []string{"amd64", "386", "arm64"} {
		ctx := build.Default
		ctx.GOARCH = arch
		inventory, err := DiscoverLanguage(ctx)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, fixture := range inventory.Cases {
			if fixture.Name != "fixedbugs/issue15609" {
				continue
			}
			found = true
			if (fixture.Exclusion != nil) != (arch != "arm64") {
				t.Fatalf("%s: %+v", arch, fixture)
			}
		}
		if !found {
			t.Fatal("upstream assembly fixture disappeared")
		}
	}
}

func TestLanguageModuleRecipe(t *testing.T) {
	for recipe, want := range map[string]string{
		"runindir":                      "",
		"runindir -gomodversion 1.23":   "1.23",
		`runindir -gomodversion "1.23"`: "1.23",
		"runindir -gomodversion '1.23'": "1.23",
		"runindir\t-gomodversion\t1.14": "1.14",
		`runindir -gomodversion ""`:     "",
	} {
		got, ok := languageModuleRecipe(recipe)
		if !ok || got != want {
			t.Fatalf("%q: %q, %t", recipe, got, ok)
		}
	}
	for _, recipe := range []string{
		"runindir -gomodversion", "runindir -gomodversion invalid",
		`runindir -gomodversion "1.23`, "runindir -gomodversion '1.23\"",
		"runindir -gomodversion 1.23 -goexperiment fieldtrack",
		"runindir -goexperiment fieldtrack", "runindir -gomodversion=1.23",
	} {
		if _, ok := languageModuleRecipe(recipe); ok {
			t.Fatalf("accepted unsupported recipe %q", recipe)
		}
	}
}

func TestRunRecipeInputs(t *testing.T) {
	for _, test := range []struct {
		recipe      string
		files, args []string
		valid       bool
	}{
		{"run arg1 arg2", nil, []string{"arg1", "arg2"}, true},
		{"run ./extra.go arg1 data.go", []string{"extra.go"}, []string{"arg1", "data.go"}, true},
		{"run -race", nil, nil, false},
		{"run ../outside.go", nil, nil, false},
		{"run /outside.go", nil, nil, false},
		{`run "quoted argument"`, nil, nil, false},
	} {
		files, args, valid := languageRunRecipe(test.recipe)
		if !slices.Equal(files, test.files) || !slices.Equal(args, test.args) || valid != test.valid {
			t.Errorf("%q: files=%v args=%v valid=%t", test.recipe, files, args, valid)
		}
	}
}
