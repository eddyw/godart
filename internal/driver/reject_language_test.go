package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestRejectFilesOfGoTest(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.Adapter == "reject-file" {
			cases = append(cases, c)
		}
	}
	runDiagnosticCases(t, inv, cases)
}

func TestSuccessfulDiagnosticsOfGoTest(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.Adapter == "diagnostic-file" {
			cases = append(cases, c)
		}
	}
	if len(cases) != 1 {
		t.Fatal("the original successful diagnostic fixtures changed")
	}
	runDiagnosticCases(t, inv, cases)
}

func TestGeneratedRejectFilesOfGoTest(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.Adapter == "generated-rejection" {
			cases = append(cases, c)
		}
	}
	if len(cases) != 4 {
		t.Fatal("the original generated rejection fixtures changed")
	}
	runDiagnosticCases(t, inv, cases)
}

func TestRejectionContracts(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		switch c.Name {
		case "cannotassign", "func1", "embedfunc", "blank1", "directive", "directive2",
			"fixedbugs/bug398", "fixedbugs/issue16369", "fixedbugs/issue56103",
			"fixedbugs/issue18459", "fixedbugs/issue18882", "linkname3",
			"fixedbugs/issue20027", "fixedbugs/issue42058a", "fixedbugs/issue42058b", "fixedbugs/issue49767",
			"fixedbugs/bug121", "fixedbugs/issue11610", "fixedbugs/issue13266", "syntax/semi7",
			"fixedbugs/issue49814", "fixedbugs/issue78355",
			"bombad", "fixedbugs/bug014", "fixedbugs/bug068", "fixedbugs/bug163",
			"fixedbugs/issue11359", "fixedbugs/issue15611", "fixedbugs/issue30722",
			"fixedbugs/issue4405", "fixedbugs/issue4468", "import6", "fixedbugs/issue11362",
			"fixedbugs/bug136", "fixedbugs/bug179", "fixedbugs/issue7538a", "goto", "label", "label1",
			"fixedbugs/issue48230", "fixedbugs/issue23586", "syntax/typesw", "fixedbugs/issue32133", "fixedbugs/issue50372":
			cases = append(cases, c)
		}
	}
	if len(cases) != 44 {
		t.Fatal("the original rejection fixtures changed")
	}
	runDiagnosticCases(t, inv, cases)
}

func runDiagnosticCases(t *testing.T, inv *parity.LanguageInventory, cases []parity.LanguageCase) {
	t.Helper()
	var names []string
	for _, c := range cases {
		names = append(names, c.Name)
	}
	selected := testShard(t, names)
	var outcomes []*languageOutcome
	for _, c := range cases {
		if selected[c.Name] {
			contract := "reject"
			if c.Adapter == "generated-rejection" {
				contract = "errorcheckoutput"
			} else if c.Adapter == "diagnostic-file" {
				contract = "diagnostics"
			}
			outcomes = append(outcomes, &languageOutcome{Fixture: c, Contract: contract, Status: "unobserved"})
		}
	}
	retainLanguageEvidence(t, "vm", inv, outcomes)
	matcherDir := t.TempDir()
	license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	matcher := selectedUpstreamSource(t, filepath.Join(build.Default.GOROOT, "src/cmd/internal/testdir/testdir_test.go"), []string{
		"splitOutput", "errorCheck", "matchPrefix", "partitionStrings", "wantedError", "errRx", "wantedErrors", "replacePrefix",
	})
	matcher += `
type test struct { *testing.T; file string }
func (t test) goFileName() string { return t.file }
func TestDiagnostics(t *testing.T) {
	file := os.Getenv("GART_ERRORCHECK_SOURCE")
	data, err := os.ReadFile(os.Getenv("GART_ERRORCHECK_DIAGNOSTICS"))
	if err != nil { t.Fatal(err) }
	if err := (test{t, file}).errorCheck(string(data), false, file, filepath.Base(file)); err != nil { t.Fatal(err) }
}
`
	writeFile(t, filepath.Join(matcherDir, "go.mod"), "module example.com/diagnostics\n\ngo 1.27\n")
	writeFile(t, filepath.Join(matcherDir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(matcherDir, "matcher_test.go"), matcher)
	executable := filepath.Join(matcherDir, "matcher")
	run(t, matcherDir, "go", "test", "-c", "-o", executable)
	config := filepath.Join(matcherDir, "importcfg")
	writeFile(t, config, run(t, "", "go", "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std"))
	column := regexp.MustCompile(`(?m)^(.*?\.go:[0-9]+):[0-9]+:`)
	for _, result := range outcomes {
		t.Run(result.Fixture.Name, func(t *testing.T) {
			t.Parallel()
			t.Cleanup(func() {
				switch {
				case t.Failed():
					result.Status = "failed"
				case t.Skipped():
					result.Status = "excluded"
				case result.Fixture.ExpectedFailure:
					result.Status = "upstream-expected-failure"
				default:
					result.Status = "passed"
				}
			})
			c := result.Fixture
			if !c.NativeEligible || !c.GartEligible {
				t.Skipf("upstream build constraints: native=%t Gart=%t", c.NativeEligible, c.GartEligible)
			}
			result.WrapperSHA256 = languageDigest([]byte(matcher))
			sourcePath := filepath.Join(build.Default.GOROOT, "test", c.File)
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if c.Adapter == "generated-rejection" {
				result.Generator = &languageExecution{}
				root := filepath.Join(build.Default.GOROOT, "test")
				args := []string{"run", "-gcflags=all=" + inv.GCFlags, sourcePath}
				for _, file := range c.GeneratorFiles {
					args = append(args, filepath.Join(root, file))
				}
				result.GeneratedBytes = []byte(recordLanguageRaw(t, result.Generator, dir, runLimit, "go", args...))
				if result.Generator.Exit != 0 {
					t.Fatalf("native generator exited %d\n%s", result.Generator.Exit, result.GeneratedBytes)
				}
				source = result.GeneratedBytes
				sourcePath = filepath.Join(dir, "generated.go")
				writeFile(t, sourcePath, string(source))
			}
			args := []string{"tool", "compile", "-p=p", "-d=panic", "-C", "-e", "-importcfg=" + config, "-o", "native.o"}
			language := "1.27"
			if c.ModuleGoVersion != "" {
				language = c.ModuleGoVersion
				args = append(args, "-lang=go"+language)
				result.ModuleSource = "module example.com/rejectfixture\n\ngo " + language + "\n"
				result.ModuleSHA256 = languageDigest([]byte(result.ModuleSource))
			}
			result.NativeDiagnostics = recordLanguage(t, &result.Native, dir, runLimit, "go", append(args, sourcePath)...)
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/rejectfixture\n\ngo "+language+"\n")
			copyPath := filepath.Join(dir, filepath.Base(c.File))
			writeFile(t, copyPath, string(source))
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			if c.Exclusion == nil {
				start := time.Now()
				pattern := copyPath
				if c.ModuleGoVersion != "" {
					pattern = "."
				}
				err = driver.Build(driver.Options{Dir: dir, Patterns: []string{pattern}, Out: t.TempDir(), Runtime: runtimeDir(t)})
				result.Gart = languageExecution{Ran: true, Seconds: time.Since(start).Seconds()}
				if err != nil {
					result.Gart.Exit = 1
					result.BuildError = err.Error()
					result.GartDiagnostics = column.ReplaceAllString(strings.ReplaceAll(err.Error(), copyPath, sourcePath), "$1:")
				}
				result.Gart.OutputSHA256 = languageDigest([]byte(result.GartDiagnostics))
			}
			for _, match := range []struct {
				name, diagnostics string
				execution         *languageExecution
				mismatch          *string
			}{
				{"native", result.NativeDiagnostics, &result.NativeMatch, &result.NativeMismatch},
				{"gart", result.GartDiagnostics, &result.GartMatch, &result.GartMismatch},
			} {
				if match.name == "gart" && c.Exclusion != nil {
					continue
				}
				path := filepath.Join(dir, match.name+".txt")
				writeFile(t, path, match.diagnostics)
				*match.mismatch = recordLanguage(t, match.execution, dir, runLimit, "env", "GART_ERRORCHECK_SOURCE="+sourcePath, "GART_ERRORCHECK_DIAGNOSTICS="+path, executable, "-test.run=^TestDiagnostics$")
				if c.ExpectedFailure {
					if match.execution.Exit != 1 {
						t.Errorf("%s: upstream expects a diagnostic mismatch, got exit %d", match.name, match.execution.Exit)
					}
				} else if match.execution.Exit != 0 {
					t.Errorf("%s assertions:\n%s", match.name, *match.mismatch)
				}
			}
			if c.Adapter == "diagnostic-file" {
				if result.Native.Exit != 0 || result.Gart.Exit != 0 {
					t.Errorf("expected successful compilation: native=%d Gart=%d", result.Native.Exit, result.Gart.Exit)
				}
			} else if result.Native.Exit <= 0 || c.Exclusion == nil && result.Gart.Exit != 1 {
				t.Errorf("expected compiler rejection: native=%d Gart=%d", result.Native.Exit, result.Gart.Exit)
			}
			if c.Exclusion != nil {
				t.Skip(c.Exclusion.Classification + ": " + c.Exclusion.Reason)
			}
		})
	}
}
