package driver_test

import (
	"encoding/json"
	"errors"
	"fmt"
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
	settings := result.Environment
	var env []string
	if len(settings) > 0 {
		env = os.Environ()
		for key, value := range settings {
			env = append(env, key+"="+value)
		}
	}
	start := time.Now()
	text, err, ended := output(dir, env, limit, name, args...)
	*result = languageExecution{Environment: settings, Command: append([]string{name}, args...), Ran: true, TimedOut: !ended, OutputSHA256: languageDigest(text), Seconds: time.Since(start).Seconds()}
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
		if c.Adapter != "single-file" && c.Adapter != "run-recipe" {
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

func TestRunDirectoryFilesOfGoTest(t *testing.T) {
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	var cases []parity.LanguageCase
	for _, c := range inv.Cases {
		if c.Adapter == "directory-module" {
			cases = append(cases, c)
		}
	}
	for _, target := range []string{"vm", "aot", "js", "wasm"} {
		t.Run(target, func(t *testing.T) { runLanguageCases(t, target, inv, cases) })
	}
}

func runLanguageCases(t *testing.T, target string, inv *parity.LanguageInventory, cases []parity.LanguageCase) {
	t.Helper()
	var importcfg, importcfgHash string
	for _, c := range cases {
		if c.HostTools {
			importcfg = filepath.Join(t.TempDir(), "importcfg")
			config := run(t, "", "go", "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std")
			writeFile(t, importcfg, config)
			importcfgHash = languageDigest([]byte(config))
			break
		}
	}
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
			} else if c.Adapter == "directory-module" {
				contract = "runindir"
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
			if c.Adapter == "directory-module" {
				settings := map[string]string{"GOPATH": dir}
				outcome.Native.Environment = settings
				outcome.Gart.Environment = settings
				dir = filepath.Join(dir, "src", filepath.Base(c.Name)+".dir")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if c.Adapter == "generated-file" {
				outcome.Generator = &languageExecution{}
				args := []string{"run", "-gcflags=all=" + inv.GCFlags, filepath.Join(root, c.File)}
				for _, file := range c.GeneratorFiles {
					args = append(args, filepath.Join(root, file))
				}
				outcome.GeneratedSource = recordLanguageRaw(t, outcome.Generator, dir, runLimit, "go", args...)
				if outcome.Generator.Exit != 0 {
					t.Fatalf("native generator exited %d\n%s", outcome.Generator.Exit, outcome.GeneratedSource)
				}
				source = []byte(outcome.GeneratedSource)
			}
			if c.Adapter == "directory-module" {
				prefix := c.Name + ".dir/"
				for _, entry := range c.Files {
					if !strings.HasPrefix(entry.Path, prefix) {
						continue
					}
					data, err := os.ReadFile(filepath.Join(root, entry.Path))
					if err != nil {
						t.Fatal(err)
					}
					path := filepath.Join(dir, strings.TrimPrefix(entry.Path, prefix))
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					writeFile(t, path, string(data))
				}
				outcome.ModuleSource = fmt.Sprintf("module %s.dir\ngo %s\n", filepath.Base(c.Name), c.ModuleGoVersion)
				outcome.ModuleSHA256 = languageDigest([]byte(outcome.ModuleSource))
				writeFile(t, filepath.Join(dir, "go.mod"), outcome.ModuleSource)
				file = "."
			} else {
				writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/gotest\n\ngo 1.27\n")
				writeFile(t, filepath.Join(dir, file), string(source))
			}
			for _, companion := range c.RunFiles {
				data, err := os.ReadFile(filepath.Join(root, companion))
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(dir, companion), string(data))
			}
			companions := map[string]string{}
			for _, companion := range c.Companions {
				data, err := os.ReadFile(filepath.Join(root, companion))
				if err != nil {
					t.Fatal(err)
				}
				companions[companion] = string(data)
				writeFile(t, filepath.Join(dir, companion), string(data))
			}
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
			if c.HostTools {
				settings := map[string]string{"GOROOT": build.Default.GOROOT, "STDLIB_IMPORTCFG": importcfg}
				outcome.Native.Environment = settings
				outcome.Gart.Environment = settings
				outcome.ImportConfigSHA256 = importcfgHash
			}
			nativeArgs := append([]string{"run", "-gcflags=all=" + inv.GCFlags}, c.NativeFlags...)
			nativeArgs = append(nativeArgs, file)
			nativeArgs = append(nativeArgs, c.RunFiles...)
			nativeArgs = append(nativeArgs, c.Arguments...)
			native := runLanguage(t, &outcome.Native, dir, runLimit, "go", nativeArgs...)
			if native != string(want) {
				t.Fatalf("native output differs from upstream fixture\nwant:\n%s\ngot:\n%s", want, native)
			}
			out := filepath.Join(dir, "out")
			patterns := []string{"./..."}
			if len(c.RunFiles) > 0 {
				patterns = append([]string{file}, c.RunFiles...)
			}
			if len(c.Companions) > 0 || c.Adapter == "directory-module" {
				patterns = []string{file}
			}
			options := driver.Options{
				BuildTags: slices.Clone(c.BuildTags),
				Trace:     traceBuild(t),
				Dir:       dir,
				Patterns:  patterns,
				Out:       out,
				Runtime:   runtime,
				Format:    false,
			}
			outcome.GartBuildTags = slices.Clone(options.BuildTags)
			err = driver.Build(options)
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
			for companion, data := range companions {
				writeFile(t, filepath.Join(out, companion), data)
			}
			limit := 2 * time.Minute
			if name == "fixedbugs/issue13169" || name == "fixedbugs/issue78081" || target == "js" && name == "copy" {
				// These upstream stress cases perform 1 million channel transfers
				// or 104 million recursive calls. Copy covers 43,560 slice combinations.
				limit = 5 * time.Minute
			}
			entry := "bin/main.dart"
			if len(c.Arguments) > 0 && (target == "js" || target == "wasm") {
				encoded, err := json.Marshal(c.Arguments)
				if err != nil {
					t.Fatal(err)
				}
				outcome.WrapperSource = "import 'bin/main.dart' as program;\nvoid main() { program.main(" + strings.ReplaceAll(string(encoded), "$", `\$`) + "); }\n"
				outcome.WrapperSHA256 = languageDigest([]byte(outcome.WrapperSource))
				entry = "arguments.dart"
				writeFile(t, filepath.Join(out, entry), outcome.WrapperSource)
			}
			command := "dart"
			args := []string{"--packages=.dart_tool/package_config.json", "bin/main.dart"}
			switch target {
			case "aot":
				run(t, out, "dart", "compile", "exe", "bin/main.dart", "-o", "app")
				command = filepath.Join(out, "app")
				args = nil
			case "js":
				run(t, out, "dart", "compile", "js", entry, "-o", "main.js")
				writeFile(t, filepath.Join(out, "run.js"), "globalThis.self = globalThis;\nrequire('./main.js');\n")
				command = "node"
				args = []string{"run.js"}
			case "wasm":
				compiled := programOutcomeWithLimit(t, out, 9*time.Minute, "dart", "compile", "wasm", entry, "-o", "main.wasm")
				if compiled.Exit != 0 {
					t.Fatalf("Wasm compilation failed:\n%s", compiled.Text)
				}
				writeFile(t, filepath.Join(out, "run.mjs"), wasmRunner)
				command = "node"
				args = []string{"run.mjs"}
			}
			runDir := out
			if c.Adapter == "directory-module" {
				runDir = dir
				if target == "vm" {
					args = []string{"--packages=" + filepath.Join(out, ".dart_tool/package_config.json"), filepath.Join(out, "bin/main.dart")}
				} else if target == "js" || target == "wasm" {
					args = []string{filepath.Join(out, args[0])}
				}
			}
			args = append(args, c.Arguments...)
			got := runLanguage(t, &outcome.Gart, runDir, limit, command, args...)
			if got != string(want) {
				t.Errorf("Gart output differs from upstream fixture\nwant:\n%s\ngot:\n%s", want, got)
			}

		})
	}
}
