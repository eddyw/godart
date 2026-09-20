package driver_test

import (
	"go/build"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

// runWithin runs a program with a time limit. A file of the test directory
// can hold a loop that never ends under the block model, and the limit keeps
// the suite alive. WaitDelay closes the pipes when a child keeps them open.
//
// The 3 results are the output, whether the program failed, and whether the
// run ended inside the limit.
func runWithin(t *testing.T, dir string, limit time.Duration, name string, args ...string) (string, bool, bool) {
	t.Helper()
	start := time.Now()
	text, err, ok := output(dir, nil, limit, name, args...)
	t.Logf("command=%s %v duration=%.3fs", name, args, time.Since(start).Seconds())
	return cutTrace(string(text)), err != nil, ok
}

// cutExitStatus drops the line that `go run` writes when the program ends with
// a value that is not 0. `dart run` writes no such line, thus the comparison
// reads the exit status of the 2 runs on its own.
func cutExitStatus(text string) string {
	lines := strings.Split(text, "\n")
	n := len(lines)
	if n > 0 && strings.HasPrefix(lines[n-1], "exit status ") {
		return strings.TrimRight(strings.Join(lines[:n-1], "\n"), "\n")
	}
	return text
}

// TestRunFilesOfGoTest compiles each file of the test directory of the Go
// repository whose header is `// run`, runs the Dart output, and compares it
// with the output of go run. The directory proves the language, thus a file
// outside the subset skips.
func TestRunFilesOfGoTest(t *testing.T) {
	policy, err := parity.ReadLanguagePolicy()
	if err != nil {
		t.Fatal(err)
	}
	rejections := policy.Rejections
	root := filepath.Join(build.Default.GOROOT, "test")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("the machine holds no test directory of Go: %v", err)
	}
	runtime := runtimeDir(t)
	type sourceCase struct {
		name, file, dir string
		source          []byte
	}
	var cases []sourceCase
	var names []string
	inv, err := parity.DiscoverLanguage(build.Default)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range inv.Cases {
		if c.Adapter != "single-file" {
			continue
		}
		source, err := os.ReadFile(filepath.Join(root, c.File))
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, sourceCase{c.Name, filepath.Base(c.File), filepath.Dir(c.File), source})
		names = append(names, c.Name)
	}
	selected := testShard(t, names)
	for _, c := range cases {
		if !selected[c.name] {
			continue
		}
		name, source := c.name, c.source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := build.Default
			ctx.BuildTags = append(slices.Clone(ctx.BuildTags), "dart")
			eligible, err := ctx.MatchFile(filepath.Join(root, c.dir), c.file)
			if err != nil {
				t.Fatal(err)
			}
			if !eligible {
				t.Skipf("upstream build constraints exclude %s/%s", ctx.GOOS, ctx.GOARCH)
			}
			if omission, ok := policy.Skips[name]; ok {
				t.Skip(omission.Classification + ": " + omission.Reason)
			}
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/gotest\n\ngo 1.27\n")
			writeFile(t, filepath.Join(dir, c.file), string(source))
			out := filepath.Join(dir, "out")
			err = driver.Build(driver.Options{
				Trace:    traceBuild(t),
				Dir:      dir,
				Patterns: []string{"./..."},
				Out:      out,
				Runtime:  runtime,
				Format:   false,
			})
			if want, known := rejections[name]; known {
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
			want, wantFail, ran := runWithin(t, dir, runLimit, "go", "run", c.file)
			if !ran {
				t.Errorf("go run did not end within %v", runLimit)
				return
			}
			want = cutExitStatus(want)
			prepareDart(t, out)
			limit := 2 * time.Minute
			if name == "fixedbugs/issue13169" || name == "fixedbugs/issue78081" {
				// These upstream stress cases perform 1 million channel transfers
				// or 104 million recursive calls.
				limit = 5 * time.Minute
			}
			got, gotFail, ended := runWithin(t, out, limit, "dart", "--packages=.dart_tool/package_config.json", "bin/main.dart")
			if !ended {
				t.Errorf("the run did not end within %v", limit)
				return
			}
			if got != want {
				t.Errorf("the output differs\nGo:\n%s\nDart:\n%s", want, got)
			}
			if gotFail != wantFail {
				t.Errorf("the exit status differs: go failed %v, and dart failed %v",
					wantFail, gotFail)
			}
		})
	}
}
