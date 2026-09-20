package driver_test

import (
	"errors"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func recordLanguage(t *testing.T, result *languageExecution, dir string, limit time.Duration, name string, args ...string) string {
	t.Helper()
	text := strings.ReplaceAll(recordLanguageRaw(t, result, dir, limit, name, args...), "\r\n", "\n")
	result.OutputSHA256 = languageDigest([]byte(text))
	return text
}

func recordLanguageRaw(t *testing.T, result *languageExecution, dir string, limit time.Duration, name string, args ...string) string {
	t.Helper()
	start := time.Now()
	text, err, ended := output(dir, nil, limit, name, args...)
	*result = languageExecution{Command: append([]string{name}, args...), Ran: true, TimedOut: !ended, OutputSHA256: languageDigest(text), Seconds: time.Since(start).Seconds()}
	if err != nil {
		result.Exit = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.Exit = exit.ExitCode()
		}
	}
	t.Logf("command=%s %v duration=%.3fs", name, args, time.Since(start).Seconds())
	if !ended {
		t.Fatalf("%s exceeded %v\n%s", name, limit, text)
	}
	return string(text)
}

func runLanguage(t *testing.T, result *languageExecution, dir string, limit time.Duration, name string, args ...string) string {
	t.Helper()
	text := recordLanguage(t, result, dir, limit, name, args...)
	if result.Exit != 0 {
		t.Fatalf("%s exited %d\n%s", name, result.Exit, text)
	}
	return text
}

// TestRunFilesOfGoTest compiles each file of the test directory of the Go
// repository whose recipe is `run`. Both executions must satisfy the
// original expected output and exit successfully.
func TestRunFilesOfGoTest(t *testing.T) {
	root := filepath.Join(build.Default.GOROOT, "test")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("the machine holds no test directory of Go: %v", err)
	}
	var cases []parity.LanguageCase
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range inv.Cases {
		if c.Adapter != "single-file" {
			continue
		}
		cases = append(cases, c)
	}
	runLanguageCases(t, "vm", inv, cases)
}

func TestRunOutputFilesOfGoTest(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.Adapter == "generated-file" {
			cases = append(cases, c)
		}
	}
	runLanguageCases(t, "vm", inv, cases)
}

func runLanguageCases(t *testing.T, target string, inv *parity.LanguageInventory, cases []parity.LanguageCase) {
	t.Helper()
	root := filepath.Join(build.Default.GOROOT, "test")
	runtime := runtimeDir(t)
	var names []string
	for _, c := range cases {
		names = append(names, c.Name)
	}
	selected := testShard(t, names)
	var outcomes []*languageOutcome
	for _, c := range cases {
		if selected[c.Name] {
			contract := "run"
			if c.Adapter == "generated-file" {
				contract = "runoutput"
			}
			outcomes = append(outcomes, &languageOutcome{Fixture: c, Contract: contract, Status: "unobserved"})
		}
	}
	retainLanguageEvidence(t, target, inv, outcomes)
	for _, outcome := range outcomes {
		c := outcome.Fixture
		name := c.Name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			t.Cleanup(func() {
				switch {
				case t.Failed():
					outcome.Status = "failed"
				case t.Skipped():
					outcome.Status = "excluded"
				default:
					outcome.Status = "passed"
				}
			})
			if !c.NativeEligible || !c.GartEligible {
				t.Skipf("upstream build constraints exclude native=%t Gart=%t", !c.NativeEligible, !c.GartEligible)
			}
			source, err := os.ReadFile(filepath.Join(root, c.File))
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Base(c.File)
			if c.Exclusion != nil {
				t.Skip(c.Exclusion.Classification + ": " + c.Exclusion.Reason)
			}
			dir := t.TempDir()
			if c.Adapter == "generated-file" {
				outcome.Generator = &languageExecution{}
				outcome.GeneratedSource = recordLanguageRaw(t, outcome.Generator, dir, runLimit, "go", "run", "-gcflags=all="+inv.GCFlags, filepath.Join(root, c.File))
				if outcome.Generator.Exit != 0 {
					t.Fatalf("native generator exited %d\n%s", outcome.Generator.Exit, outcome.GeneratedSource)
				}
				source = []byte(outcome.GeneratedSource)
			}
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/gotest\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, file), string(source))
			license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			want, err := os.ReadFile(filepath.Join(root, name+".out"))
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			outcome.ExpectedSHA256 = languageDigest(want)
			native := runLanguage(t, &outcome.Native, dir, runLimit, "go", "run", "-gcflags=all="+inv.GCFlags, file)
			if native != string(want) {
				t.Fatalf("native output differs from upstream fixture\nwant:\n%s\ngot:\n%s", want, native)
			}
			out := filepath.Join(dir, "out")
			err = driver.Build(driver.Options{
				Trace:    traceBuild(t),
				Dir:      dir,
				Patterns: []string{"./..."},
				Out:      out,
				Runtime:  runtime,
				Format:   false,
			})
			if err != nil {
				outcome.BuildError = err.Error()
			}
			if want := c.Rejection; want != nil {
				if err == nil {
					t.Fatal("the build now passes: remove its expected rejection")
				}
				got := strings.Split(strings.ReplaceAll(err.Error(), dir, "$DIR"), "\n")
				slices.Sort(got)
				if !slices.Equal(got, want) {
					t.Fatalf("the rejection changed\nwant:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
				}
				t.Skipf("known rejection:\n%v", err)
			}
			if err != nil {
				t.Fatalf("unexpected build failure:\n%v", err)
			}
			prepareDart(t, out)
			limit := 2 * time.Minute
			if name == "fixedbugs/issue13169" || name == "fixedbugs/issue78081" || target == "js" && name == "copy" {
				// These upstream stress cases perform 1 million channel transfers
				// or 104 million recursive calls. Copy covers 43,560 slice combinations.
				limit = 5 * time.Minute
			}
			command := "dart"
			args := []string{"--packages=.dart_tool/package_config.json", "bin/main.dart"}
			switch target {
			case "aot":
				run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
				command = filepath.Join(out, "app")
				args = nil
			case "js":
				run(t, out, "dart", "compile", "js", "bin/main.dart", "-o", "main.js")
				writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis;\nrequire('./main.js');\n")
				command = "node"
				args = []string{"run.js"}
			case "wasm":
				run(t, out, "dart", "compile", "wasm", "bin/main.dart", "-o", "main.wasm")
				writeFile(t, filepath.Join(out, "run.mjs"), wasmRunner)
				command = "node"
				args = []string{"run.mjs"}
			}
			got := runLanguage(t, &outcome.Gart, out, limit, command, args...)
			if got != string(want) {
				t.Errorf("Gart output differs from upstream fixture\nwant:\n%s\ngot:\n%s", want, got)
			}

		})
	}
}
