package driver_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestUpstreamFuzzReplay(t *testing.T) {
	for _, c := range []struct {
		pkg, file, target string
		declarations      []string
	}{
		{"encoding/csv", "fuzz_test.go", "FuzzRoundtrip", []string{"FuzzRoundtrip"}},
		{"strings", "strings_test.go", "FuzzReplace", []string{"ReplaceTests", "FuzzReplace"}},
		{"encoding/pem", "pem_test.go", "FuzzDecode", []string{"FuzzDecode"}},
	} {
		t.Run(c.pkg, func(t *testing.T) {
			dir := t.TempDir()
			root := filepath.Join(build.Default.GOROOT, "src", c.pkg)
			sourcePath := filepath.Join(root, c.file)
			source := selectedUpstreamSource(t, sourcePath, c.declarations)
			name := filepath.Base(c.pkg)
			if name != "strings" {
				source = strings.Replace(source, "package "+name, "package "+name+"_test\nimport . "+fmt.Sprintf("%q", c.pkg), 1)
			}
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fuzzreplay\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, c.file), source)
			license, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			original, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			report := map[string]any{
				"kind": "fuzz-replay", "status": "incomplete", "package": c.pkg, "target": "vm", "function": c.target,
				"go_version": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH, "short": testing.Short(),
				"source_file": sourcePath, "source_sha256": fmt.Sprintf("%x", sha256.Sum256(original)),
				"fixture_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(source))),
			}
			defer func() {
				if base := os.Getenv("GART_PARITY_REPORT_DIR"); base != "" {
					base = filepath.Join(base, "fuzz-replay", strings.ReplaceAll(c.pkg, "/", "_"))
					if err := os.MkdirAll(base, 0755); err != nil {
						t.Error(err)
						return
					}
					if err := os.CopyFS(filepath.Join(base, "fixture"), os.DirFS(dir)); err != nil {
						t.Error(err)
					}
					data, err := json.MarshalIndent(report, "", "  ")
					if err != nil {
						t.Error(err)
						return
					}
					writeFile(t, filepath.Join(base, "fuzz.json"), string(data)+"\n")
				}
			}()
			corpus := filepath.Join(dir, "testdata", "fuzz", c.target)
			if err := os.MkdirAll(corpus, 0755); err != nil {
				t.Fatal(err)
			}
			copyCorpus := func(from string) int {
				entries, err := os.ReadDir(from)
				if os.IsNotExist(err) {
					return 0
				}
				if err != nil {
					t.Fatal(err)
				}
				count := 0
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					data, err := os.ReadFile(filepath.Join(from, entry.Name()))
					if err != nil {
						t.Fatal(err)
					}
					path := filepath.Join(corpus, entry.Name())
					if old, err := os.ReadFile(path); err == nil {
						if string(old) != string(data) {
							t.Fatal("conflicting corpus contents: " + entry.Name())
						}
						continue
					} else if !os.IsNotExist(err) {
						t.Fatal(err)
					}
					writeFile(t, path, string(data))
					count++
				}
				return count
			}
			report["upstream_corpus"] = copyCorpus(filepath.Join(root, "testdata", "fuzz", c.target))
			cache := t.TempDir()
			budget := "1000x"
			if !testing.Short() {
				budget = "10000x"
			}
			args := []string{"test", "-run=^$", "-fuzz=^" + c.target + "$", "-fuzztime=" + budget, "-parallel=1", "-count=1", "-timeout=2m"}
			if testing.Short() {
				args = append(args, "-short")
			}
			args = append(args, "-args", "-test.fuzzcachedir="+cache)
			report["generation_command"] = append([]string{"go"}, args...)
			generation := programOutcomeWithLimit(t, dir, 3*time.Minute, "go", args...)
			report["generation"] = map[string]any{"exit": generation.Exit, "output": generation.Text}
			report["generated_corpus"] = copyCorpus(filepath.Join(cache, c.target))
			var files []parity.SourceFile
			err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				name, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				files = append(files, parity.SourceFile{Path: filepath.ToSlash(name), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))})
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			report["fixtures"] = files
			if generation.Exit != 0 {
				report["status"] = "failed"
				t.Fatalf("native fuzz generation failed:\n%s", generation.Text)
			}
			args = []string{"test", "-json", "-count=1", "-run=^" + c.target + "$"}
			if testing.Short() {
				args = append(args, "-short")
			}
			nativeRun := programOutcome(t, dir, "go", args...)
			native, err := parity.NativeResults(strings.NewReader(nativeRun.Text), nativeRun.Exit)
			if err != nil {
				t.Fatal(err)
			}
			report["native"] = native
			report["native_output"] = nativeRun.Text
			report["native_command"] = append([]string{"go"}, args...)
			out := t.TempDir()
			stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
			if err != nil {
				t.Fatal(err)
			}
			prepareDart(t, out)
			args = []string{filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.short=" + fmt.Sprint(testing.Short()), "-test.run=^" + c.target + "$"}
			gartRun := programOutcome(t, dir, "dart", args...)
			gart, err := parity.GartResults(strings.NewReader(gartRun.Text), gartRun.Exit)
			if err != nil {
				t.Fatal(err)
			}
			report["gart"] = gart
			report["gart_output"] = gartRun.Text
			report["gart_command"] = append([]string{"dart"}, args...)
			report["dart_version"] = strings.TrimSpace(run(t, "", "dart", "--version"))
			comparison := parity.Compare(native, gart, []string{c.target}, nil)
			report["comparison"] = comparison
			if len(comparison.Problems) != 0 || len(comparison.Passed) < 2 {
				report["status"] = "failed"
				t.Fatalf("fuzz replay needs matching executed inputs: %+v", comparison)
			}
			report["status"] = "passed"
			t.Logf("fuzz replay: %d passed identities, %v new corpus inputs", len(comparison.Passed), report["generated_corpus"])
		})
	}
}
