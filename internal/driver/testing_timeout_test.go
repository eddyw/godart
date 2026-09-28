package driver_test

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"golang.org/x/tools/txtar"
)

func TestTestingTimeoutUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("the original timeout scripts require a full run")
	}
	cli := filepath.Join(t.TempDir(), "gart")
	run(t, "", "go", "build", "-o", cli, "../../cmd/gart")
	for _, name := range []string{"test_deadline.txt", "test_timeout.txt", "test_benchmark_timeout.txt"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/cmd/go/testdata/script", name))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("upstream=%s Go=%s SHA256=%x", name, runtime.Version(), sha256.Sum256(source))
			archive := txtar.Parse(source)
			dir := t.TempDir()
			for _, file := range archive.Files {
				if !filepath.IsLocal(file.Name) {
					t.Fatalf("nonlocal upstream fixture: %s", file.Name)
				}
				path := filepath.Join(dir, file.Name)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, string(file.Data))
			}
			license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			out := t.TempDir()
			var native, gart programResult
			commands := 0
			for _, line := range strings.Split(string(archive.Comment), "\n") {
				switch {
				case line == "", strings.HasPrefix(line, "#"), line == "[short] skip":
				case line == "env GO111MODULE=off":
					t.Setenv("GO111MODULE", "off")
				case line == "cd a":
					dir = filepath.Join(dir, "a")
				case strings.HasPrefix(line, "go test "):
					commands++
					args := strings.Fields(strings.TrimPrefix(line, "go "))
					native = programOutcome(t, dir, "go", args...)
					gartArgs := append([]string{"test", "-o", out, "-runtime", runtimeDir(t)}, args[1:]...)
					if name == "test_timeout.txt" {
						// An explicit file avoids the unsupported GOPATH import identity.
						for i, arg := range gartArgs {
							if arg == "." {
								gartArgs[i] = "./timeout_test.go"
							}
						}
					}
					gart = programOutcome(t, dir, cli, gartArgs...)
					if native.Exit != 0 || gart.Exit != 0 {
						t.Fatalf("%s\nGo: %+v\nGart: %+v", line, native, gart)
					}
				case strings.HasPrefix(line, "stdout "):
					pattern := strings.Trim(strings.TrimPrefix(line, "stdout "), "'")
					re := regexp.MustCompile(pattern)
					if commands == 0 || !re.MatchString(native.Text) || !re.MatchString(gart.Text) {
						t.Fatalf("%s\nGo: %+v\nGart: %+v", line, native, gart)
					}
				default:
					t.Fatalf("unsupported upstream command: %s", line)
				}
			}
			if commands == 0 {
				t.Fatal("empty upstream script")
			}
		})
	}
}

func TestTestingTimeoutTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("the original deadline script requires a full run")
	}
	for _, script := range []string{"test_deadline.txt", "test_json_timeout.txt"} {
		t.Run(script, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/cmd/go/testdata/script", script))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("upstream=%s Go=%s SHA256=%x", script, runtime.Version(), sha256.Sum256(source))
			dir := t.TempDir()
			for _, file := range txtar.Parse(source).Files {
				if !filepath.IsLocal(file.Name) {
					t.Fatalf("nonlocal upstream fixture: %s", file.Name)
				}
				writeFile(t, filepath.Join(dir, file.Name), string(file.Data))
			}
			license, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "LICENSE"), string(license))
			native := filepath.Join(t.TempDir(), "native.test")
			run(t, dir, "go", "test", "-c", "-o", native)
			out := t.TempDir()
			stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Binaries) != 1 {
				t.Fatalf("got %d binaries", len(stats.Binaries))
			}
			prepareDart(t, out)
			cases := [][]string{{"-test.timeout=0", "-test.run=^TestNoDeadline$"}, {"-test.timeout=1m", "-test.run=^(TestDeadlineWithinMinute|TestSubtestDeadlineWithinMinute)$"}}
			if script == "test_json_timeout.txt" {
				cases = [][]string{{"-test.timeout=1ms"}}
			}
			for _, args := range cases {
				t.Run(strings.Join(args, ","), func(t *testing.T) {
					want := programOutcome(t, dir, native, args...)
					for _, target := range []string{"vm", "aot", "js", "wasm"} {
						t.Run(target, func(t *testing.T) {
							var got programResult
							if target == "vm" {
								got = programOutcome(t, dir, "dart", append([]string{filepath.Join(out, stats.Binaries[0].Entry)}, args...)...)
							} else {
								got = runUpstreamTarget(t, target, out, stats.Binaries[0].Entry, dir, args)
							}
							if script == "test_json_timeout.txt" {
								for _, result := range []programResult{want, got} {
									if result.Exit != 2 || !strings.Contains(result.Text, "panic: test timed out after 1ms") {
										t.Fatalf("timeout result: %+v", result)
									}
								}
							} else if want.Exit != 0 || got.Exit != 0 || !strings.Contains(got.Text, "PASS") {
								t.Fatalf("Go: %+v\nGart: %+v", want, got)
							}
						})
					}
				})
			}
		})
	}
}
