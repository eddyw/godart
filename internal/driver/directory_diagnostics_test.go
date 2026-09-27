package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func directoryDiagnosticMatcher(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	source := selectedUpstreamSource(t, filepath.Join(build.Default.GOROOT, "src/cmd/internal/testdir/testdir_test.go"), []string{
		"splitOutput", "errorCheck", "matchPrefix", "partitionStrings", "wantedError", "errRx", "wantedErrors", "replacePrefix",
	})
	source += `
type test struct { *testing.T; file string }
func (t test) goFileName() string { return t.file }
func TestDiagnostics(t *testing.T) {
 files := strings.Split(os.Getenv("GART_ERRORCHECK_SOURCES"), "\n")
 data, err := os.ReadFile(os.Getenv("GART_ERRORCHECK_DIAGNOSTICS"))
 if err != nil { t.Fatal(err) }
 var fullshort []string
 for _, file := range files { fullshort = append(fullshort, file, filepath.Base(file)) }
 if err := (test{t, files[0]}).errorCheck(string(data), false, fullshort...); err != nil { t.Fatal(err) }
}
`
	license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/directorydiagnostics\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "matcher_test.go"), source)
	executable := filepath.Join(dir, "matcher")
	run(t, dir, "go", "test", "-c", "-o", executable)
	return executable, source
}

func matchDirectoryDiagnostics(t *testing.T, result *languageOutcome, executable, sourceDir, dir string) {
	t.Helper()
	var paths []string
	for _, pkg := range result.DirectoryPackages {
		for _, name := range pkg.Files {
			paths = append(paths, filepath.Join(sourceDir, name))
		}
	}
	matches := []struct {
		name, diagnostics string
		execution         *languageExecution
		mismatch          *string
	}{
		{"native", result.NativeDiagnostics, &result.NativeMatch, &result.NativeMismatch},
		{"gart", result.GartDiagnostics, &result.GartMatch, &result.GartMismatch},
	}
	if result.DiagnosticExclusion != nil {
		matches = matches[:1]
	}
	for _, match := range matches {
		path := filepath.Join(dir, match.name+".txt")
		writeFile(t, path, match.diagnostics)
		*match.mismatch = recordLanguage(t, match.execution, dir, runLimit, "env", "GART_ERRORCHECK_SOURCES="+strings.Join(paths, "\n"), "GART_ERRORCHECK_DIAGNOSTICS="+path, executable, "-test.run=^TestDiagnostics$")
		if result.Fixture.ExpectedFailure {
			if match.execution.Exit != 1 {
				t.Errorf("%s: upstream expects a diagnostic mismatch, got exit %d", match.name, match.execution.Exit)
			}
		} else if match.execution.Exit != 0 {
			t.Errorf("%s assertions:\n%s", match.name, *match.mismatch)
		}
	}
}
