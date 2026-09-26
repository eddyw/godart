package driver_test

import (
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestCompileFilesOfGoTest(t *testing.T) {
	runCompilationCases(t, "compile-file")
}

func TestBuildFilesOfGoTest(t *testing.T) {
	runCompilationCases(t, "build-file")
}

func runCompilationCases(t *testing.T, adapter string) {
	t.Helper()
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range inv.Cases {
		if c.Adapter == adapter {
			names = append(names, c.Name)
		}
	}
	selected := testShard(t, names)
	var outcomes []*languageOutcome
	for _, c := range inv.Cases {
		if c.Adapter == adapter && selected[c.Name] {
			outcomes = append(outcomes, &languageOutcome{Fixture: c, Contract: c.Action, Status: "unobserved"})
		}
	}
	retainLanguageEvidence(t, "vm", inv, outcomes)
	config := filepath.Join(t.TempDir(), "importcfg")
	imports := run(t, "", "go", "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std")
	writeFile(t, config, imports)
	var compiler string
	for _, result := range outcomes {
		if compileEnvironment(result.Fixture.Recipe) != nil {
			compiler = filepath.Join(t.TempDir(), "gart.exe")
			run(t, "", "go", "build", "-o", compiler, "../../cmd/gart")
			break
		}
	}
	for _, result := range outcomes {
		t.Run(result.Fixture.Name, func(t *testing.T) {
			t.Parallel()
			t.Cleanup(func() {
				switch {
				case t.Failed():
					result.Status = "failed"
				case t.Skipped():
					result.Status = "excluded"
				default:
					result.Status = "passed"
				}
			})
			c := result.Fixture
			if env := compileEnvironment(c.Recipe); env != nil {
				result.Native.Environment = env
				result.Gart.Environment = env
				result.GartBuild = &languageExecution{Environment: env}
			}
			if !c.NativeEligible || !c.GartEligible {
				t.Skipf("upstream build constraints: native=%t Gart=%t", c.NativeEligible, c.GartEligible)
			}
			sourcePath := filepath.Join(build.Default.GOROOT, "test", c.File)
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if c.Action == "build" {
				runLanguage(t, &result.Native, dir, runLimit, "go", "build", "-gcflags=all="+inv.GCFlags, "-o", "native.o", sourcePath)
			} else {
				packageOption := "-p=p"
				if c.Recipe == "compile -p=main" {
					packageOption = "-p=main"
				}
				command := []string{"go", "tool", "compile", "-e", packageOption, "-importcfg=" + config}
				if c.Recipe != "compile -p=p" && c.Recipe != "compile -p=main" && result.GartBuild == nil {
					command = append(command, strings.Fields(c.Recipe)[1:]...)
				}
				command = append(command, "-o", "native.o", sourcePath)
				runLanguage(t, &result.Native, dir, runLimit, command[0], command[1:]...)
			}
			if c.Exclusion != nil {
				t.Skip(c.Exclusion.Classification + ": " + c.Exclusion.Reason)
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, c.File, source, parser.PackageClauseOnly)
			if err != nil {
				t.Fatal(err)
			}
			pkg := file.Name.Name
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/compilefixture\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, "fixture.go"), string(source))
			license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			out := t.TempDir()
			if result.GartBuild != nil {
				runLanguage(t, result.GartBuild, dir, runLimit, compiler, "build", "-format=false", "-runtime", runtimeDir(t), "-o", out, ".")
			} else {
				err = driver.Build(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t)})
			}
			if err != nil {
				result.BuildError = err.Error()
			}
			if c.Rejection != nil {
				if err == nil {
					t.Fatal("the build now passes: remove its expected rejection")
				}
				got := strings.Split(strings.ReplaceAll(err.Error(), dir, "$DIR"), "\n")
				slices.Sort(got)
				if !slices.Equal(got, c.Rejection) {
					t.Fatalf("the rejection changed\nwant:\n%s\ngot:\n%s", strings.Join(c.Rejection, "\n"), strings.Join(got, "\n"))
				}
				t.Skipf("known rejection:\n%v", err)
			}
			if err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			library := "package:compilefixture/" + pkg + ".dart"
			if pkg == "main" {
				library = "bin/main.dart"
			}
			wrapper := "import '" + library + "';\nvoid main() {}\n"
			result.WrapperSHA256 = languageDigest([]byte(wrapper))
			writeFile(t, filepath.Join(out, "compile.dart"), wrapper)
			entry := "compile.dart"
			if c.Action == "build" {
				if pkg == "main" {
					entry = "bin/main.dart"
				}
				data, err := os.ReadFile(filepath.Join(out, entry))
				if err != nil {
					t.Fatal(err)
				}
				result.WrapperSource = string(data)
				result.WrapperSHA256 = languageDigest(data)
			}
			runLanguage(t, &result.Gart, out, runLimit, "dart", "compile", "kernel", entry, "-o", "compile.dill")
		})
	}
}

func compileEnvironment(recipe string) map[string]string {
	var key, value string
	switch recipe {
	case "compile -godebug gotypesalias=1":
		key, value = "GODEBUG", "gotypesalias=1"
	case "compile -goexperiment fieldtrack":
		key, value = "GOEXPERIMENT", "fieldtrack"
	default:
		return nil
	}
	if previous := os.Getenv(key); previous != "" {
		value = previous + "," + value
	}
	return map[string]string{key: value}
}
