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
	runCompilerRecipes := map[string]struct {
		recipe string
		flags  []string
	}{
		"runcallersplit": {"run -gcflags -l=4", []string{"-gcflags", "-l=4"}},
		"runnoinline":    {`run -gcflags="-l"`, []string{"-gcflags=-l"}},
		"runnoopt":       {"run -gcflags='all=-N -l'", []string{"-gcflags=all=-N -l"}},
		"runconvert":     {"run -gcflags=-d=converthash=qy", []string{"-gcflags=-d=converthash=qy"}},
		"runsoftfloat":   {"run -gcflags=-d=softfloat", []string{"-gcflags=-d=softfloat"}},
		"runssa":         {"run -gcflags=-d=ssa/check/on", []string{"-gcflags=-d=ssa/check/on"}},
		"runcaller":      {"run -gcflags=-l=4", []string{"-gcflags=-l=4"}},
	}
	for name, recipe := range runCompilerRecipes {
		write("test/"+name+".go", "// "+recipe.recipe+"\npackage main\nfunc main() {}\n")
	}
	write("test/runoptionextra.go", "// run -gcflags=-l=4 -race\npackage main\n")
	write("test/runoptionunknown.go", "// run -gcflags=-unknown\npackage main\n")
	write("test/run.go", "// run\npackage main\nfunc main() {}\n")
	write("test/run.out", "expected\n")
	write("test/compile.go", "// compile\npackage p\n")
	write("test/assembly.go", "// asmcheck\npackage p\n")
	write("test/assemblyflags.go", "// asmcheck -gcflags=-d=ssa/check/on\npackage p\n")
	for _, action := range []string{"builddir", "buildrundir"} {
		write("test/"+action+".go", "// "+action+"\npackage ignored\n")
		write("test/"+action+".dir/a.go", "package main\nfunc F()\nfunc main() { F() }\n")
		write("test/"+action+".dir/a.s", "TEXT ·F(SB),$0-0\nRET\n")
		write("test/"+action+"pure.go", "// "+action+"\npackage ignored\n")
		write("test/"+action+"pure.dir/a.go", "package main\nfunc main() {}\n")
	}
	write("test/reject.go", "// errorcheck\npackage p\n")
	write("test/rejectversion.go", "// errorcheck -lang=go1.17\npackage p\n")
	write("test/rejectversionsuccess.go", "// errorcheck -0 -lang=go1.17\npackage p\n")
	write("test/rejectpanic.go", "// errorcheck -d=panic\npackage p\n")
	write("test/rejectall.go", "// errorcheck -e\npackage p\n")
	write("test/rejectlimited.go", "// errorcheck -e=0\npackage p\n")
	write("test/rejectcomplete.go", "// errorcheck -complete\npackage p\n")
	write("test/rejecttimed.go", "// errorcheck -t 10\npackage p\n")
	write("test/rejecttimedflags.go", "// errorcheck -t 10 -e=0\npackage p\n")
	write("test/build.go", "// build\npackage main\n")
	write("test/buildinline.go", "// build -gcflags=-l=4\npackage p\n")
	write("test/buildflags.go", "// build -race\npackage main\n")
	write("test/rejectflags.go", "// errorcheck -0\npackage p\n")
	write("test/rejectdirectory.go", "// errorcheckdir\npackage ignored\n")
	write("test/rejectdirectory.dir/a.go", "package a\n")
	write("test/rejectdirectoryflags.go", "// errorcheckdir -0\npackage ignored\n")
	write("test/rejectdirectoryflags.dir/a.go", "package a\n")
	write("test/rejectsingle.go", "// errorcheckdir -s\npackage ignored\n")
	write("test/rejectsingle.dir/a.go", "package a\n")
	write("test/groups.go", "// compiledir\npackage ignored\n")
	write("test/groups.dir/a.go", "package a\n")
	write("test/groupssingle.go", "// compiledir -s\npackage ignored\n")
	write("test/groupssingle.dir/a.go", "package a\n")
	for _, workers := range []string{"2", "4"} {
		write("test/groupsworkers"+workers+".go", "// compiledir -c="+workers+"\npackage ignored\n")
		write("test/groupsworkers"+workers+".dir/a.go", "package a\n")
	}
	write("test/groupsrun.go", "// rundir\npackage ignored\n")
	write("test/groupsrun.dir/a.go", "package a\n")
	compileRecipes := map[string]string{
		"compileflags":      "compile -N",
		"compilebounds":     "compile -B",
		"compileworkers2":   "compile -c=2",
		"compileworkers4":   "compile -c=4",
		"compileinline":     "compile -l",
		"compilessa":        "compile -d=ssa/check/on",
		"compilesoftfloat":  "compile -N -d=softfloat",
		"compilecheckptr":   "compile -d=checkptr",
		"compilefuzzer":     "compile -d=libfuzzer",
		"compileseed":       "compile -d=ssa/check/seed",
		"compileseed1":      "compile -d=ssa/check/seed=1",
		"compiledynlink":    "compile -dynlink",
		"compilealias":      "compile -godebug gotypesalias=1",
		"compilefieldtrack": "compile -goexperiment fieldtrack",
	}
	for name, recipe := range compileRecipes {
		write("test/"+name+".go", "// "+recipe+"\npackage p\n")
	}
	write("test/compileextra.go", "// compile -N -l\npackage p\n")
	write("test/compileunknown.go", "// compile -c=8\npackage p\n")
	write("test/compilepackage.go", "// compile -p=p\npackage v4\n")
	write("test/compilemain.go", "// compile -p=main\npackage main\n")
	write("test/compileotherpackage.go", "// compile -p=other\npackage v4\n")
	write("test/generated.go", "// runoutput\npackage main\n")
	write("test/generatedflags.go", "// runoutput -gcflags=-N\npackage main\n")
	write("test/generatedreject.go", "// errorcheckoutput\npackage main\n")
	write("test/generatedrejectfiles.go", "// errorcheckoutput ./extra.go\npackage main\n")
	write("test/generatedrejectflags.go", "// errorcheckoutput -0\npackage main\n")
	write("test/generatedfiles.go", "// runoutput ./extra.go\npackage main\n")
	write("test/extra.go", "// skip\npackage main\n")
	write("test/suffix_windows.go", "// run\npackage main\nfunc main() {}\n")
	write("test/constraints.go", "//go:build dart\n\n// run\npackage main\nfunc main() {}\n")
	write("test/inline.go", "// rundir -l=4\npackage ignored\n")
	write("test/inline.dir/main.go", "package main\nfunc main() {}\n")
	write("test/other/directory.go", "// rundir -G=3\npackage ignored\n")
	write("test/other/directory.dir/main.go", "package main\nfunc main() {}\n")
	write("test/linkx_run.go", "// run\npackage main\nfunc main() {}\n")
	write("test/linkx.go", "// skip\npackage main\nfunc main() {}\n")
	optimizationRecipes := map[string]string{
		"optimizationml":    "errorcheck -0 -m -l",
		"optimizationm":     "errorcheck -0 -m",
		"optimizationm2":    "errorcheck -0 -m=2",
		"optimizationnml":   "errorcheck -0 -N -m -l",
		"optimizationlm":    "errorcheck -0 -l -m",
		"optimizationmm":    "errorcheck -0 -m -m",
		"optimizationsmall": "errorcheck -0 -m -l -smallframes",
		"optimizationlm2":   "errorcheck -0 -l -m=2",
	}
	for name, recipe := range optimizationRecipes {
		write("test/"+name+".go", "// "+recipe+"\npackage p\n")
	}
	write("test/optimizationreject.go", "// errorcheck -m\npackage p\n")
	write("test/optimizationversion.go", "// errorcheck -0 -m -lang=go1.24\npackage p\n")
	ctx := build.Default
	ctx.GOROOT = root
	ctx.GOOS = "linux"
	inv, err := DiscoverLanguage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Cases) != 85 {
		t.Fatalf("got %d cases, want 85", len(inv.Cases))
	}
	cases := map[string]LanguageCase{}
	for _, c := range inv.Cases {
		cases[c.Name] = c
	}
	for _, name := range []string{"assembly", "assemblyflags"} {
		c := cases[name]
		if c.Adapter != "missing" || c.Exclusion == nil || c.Exclusion.Classification != "target-limit" || c.Exclusion.Reason == "" || len(c.Files) != 1 {
			t.Fatalf("native assembly contract lost: %+v", c)
		}
	}
	for _, name := range []string{"builddir", "buildrundir"} {
		c := cases[name]
		if c.Exclusion == nil || c.Exclusion.Classification != "target-limit" || c.Exclusion.Reason == "" || len(c.Files) != 3 || c.Files[2].Path != name+".dir/a.s" || c.Files[2].SHA256 == "" {
			t.Fatalf("native assembly directory contract lost: %+v", c)
		}
		if c := cases[name+"pure"]; c.Exclusion != nil || c.Adapter != "missing" {
			t.Fatalf("a Go-only directory must retain its missing adapter: %+v", c)
		}
	}
	for name, recipe := range optimizationRecipes {
		c := cases[name]
		if c.Adapter != "missing" || c.Exclusion == nil || c.Exclusion.Classification != "target-limit" || c.Exclusion.Reason == "" || c.Recipe != recipe || len(c.Files) != 1 || c.Files[0].SHA256 == "" {
			t.Fatalf("native optimization contract lost: %+v", c)
		}
	}
	for _, name := range []string{"optimizationreject", "optimizationversion", "reject", "rejectversionsuccess"} {
		if c := cases[name]; c.Exclusion != nil {
			t.Fatalf("an unclassified diagnostic contract became a limit: %+v", c)
		}
	}
	if cases["groups"].Adapter != "directory-packages" || cases["groupsrun"].Adapter != "directory-packages" {
		t.Fatal("plain package directory recipes have no adapter")
	}
	for _, workers := range []string{"2", "4"} {
		c := cases["groupsworkers"+workers]
		if c.Adapter != "directory-packages" || c.Recipe != "compiledir -c="+workers || len(c.Files) != 2 {
			t.Fatalf("directory compiler option lost: %+v", c)
		}
	}
	for _, name := range []string{"groupssingle", "rejectsingle"} {
		if c := cases[name]; c.Adapter != "directory-packages" || c.Recipe != c.Action+" -s" || len(c.Files) != 2 {
			t.Fatalf("separate package recipe lost: %+v", c)
		}
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
	if cases["rejectall"].Adapter != "reject-file" || cases["rejectlimited"].Adapter != "reject-file" {
		t.Fatal("the rejection adapter must preserve the diagnostic limit")
	}
	if cases["rejectcomplete"].Adapter != "reject-file" {
		t.Fatal("complete-package diagnostics need the rejection adapter")
	}
	if cases["rejecttimed"].Adapter != "reject-file" || cases["rejecttimedflags"].Adapter != "missing" {
		t.Fatal("the timed rejection adapter must preserve extra recipe options")
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
	if c := cases["buildinline"]; c.Adapter != "build-file" || !slices.Equal(c.NativeFlags, []string{"-gcflags=-l=4"}) {
		t.Fatal("the build adapter must preserve its compiler flag")
	}
	if cases["rejectdirectory"].Adapter != "directory-packages" || cases["rejectdirectoryflags"].Adapter != "missing" {
		t.Fatal("directory diagnostic recipes must preserve their flags")
	}
	if cases["compile"].Adapter != "compile-file" || cases["compileextra"].Adapter != "missing" || cases["compileunknown"].Adapter != "missing" {
		t.Fatal("the compile adapter must not discard recipe flags")
	}
	for name, recipe := range compileRecipes {
		if c := cases[name]; c.Adapter != "compile-file" || c.Recipe != recipe {
			t.Fatalf("the compile adapter lost its recipe: %+v", c)
		}
	}
	if c := cases["compilepackage"]; c.Adapter != "compile-file" || c.Recipe != "compile -p=p" || cases["compileotherpackage"].Adapter != "missing" {
		t.Fatal("the compile adapter must preserve its package option")
	}
	if c := cases["compilemain"]; c.Adapter != "compile-file" || c.Recipe != "compile -p=main" {
		t.Fatal("the compile adapter must preserve the main package option")
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
	if c := cases["inline"]; c.Adapter != "directory-packages" || c.Recipe != "rundir -l=4" || len(c.Files) != 2 {
		t.Fatalf("directory inline option lost: %+v", c)
	}
	if c := cases["other/directory"]; c.Action != "rundir" || c.Recipe != "rundir -G=3" || len(c.Files) != 2 || c.Adapter != "missing" {
		t.Fatalf("companion or recipe lost: %+v", c)
	}
	if c := cases["runarguments"]; c.Adapter != "run-recipe" || !slices.Equal(c.Arguments, []string{"arg1", "arg2"}) {
		t.Fatalf("program arguments lost: %+v", c)
	}
	for name, recipe := range runCompilerRecipes {
		if c := cases[name]; c.Adapter != "run-recipe" || c.Recipe != recipe.recipe || !slices.Equal(c.NativeFlags, recipe.flags) || len(c.Arguments) != 0 || len(c.RunFiles) != 0 {
			t.Fatalf("the run adapter lost its compiler options: %+v", c)
		}
	}
	if cases["runoptionextra"].Adapter != "missing" || cases["runoptionunknown"].Adapter != "missing" {
		t.Fatal("the run adapter must preserve extra or unknown options")
	}
	if c := cases["runconvert"]; c.Exclusion == nil || c.Exclusion.Classification != "target-limit" || c.Exclusion.Reason == "" {
		t.Fatal("experimental native conversion behavior must retain its target limit")
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
		if (cases[name].Adapter != "single-file" && cases[name].Adapter != "compile-file" && cases[name].Adapter != "build-file" && (cases[name].Adapter != "directory-packages" || cases[name].Action == "errorcheckdir")) || len(diagnostics) == 0 {
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
