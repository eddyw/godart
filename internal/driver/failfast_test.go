package driver_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"golang.org/x/tools/txtar"
)

func TestFailFastUpstream(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/cmd/go/testdata/script/test_fail_fast.txt"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("upstream=test_fail_fast.txt Go=%s SHA256=%x", runtime.Version(), sha256.Sum256(source))
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
	cli := filepath.Join(t.TempDir(), "gart")
	run(t, "", "go", "build", "-o", cli, "../../cmd/gart")
	assertion := regexp.MustCompile(`^stdout -count=(\d+) '(.*)'$`)
	var native, gart programResult
	var cliResult *programResult
	commands, assertions := 0, 0
	for _, line := range strings.Split(string(archive.Comment), "\n") {
		if strings.HasPrefix(line, "! go test ") {
			commands++
			cliResult = nil
			args := strings.Fields(strings.TrimPrefix(line, "! go "))
			options := driver.TestOptions{}
			var patterns []string
			for i := 1; i < len(args); i++ {
				switch {
				case strings.HasPrefix(args[i], "-run="):
					options.Run = strings.Trim(strings.TrimPrefix(args[i], "-run="), "'")
					args[i] = "-run=" + options.Run
				case strings.HasPrefix(args[i], "-failfast="):
					options.FailFast, err = strconv.ParseBool(strings.TrimPrefix(args[i], "-failfast="))
					if err != nil {
						t.Fatal(err)
					}
				case args[i] == "-failfast":
					options.FailFast = true
				case args[i] == "-p":
					i++
					if i >= len(args) || args[i] != "1" {
						t.Fatalf("unsupported upstream process count: %s", line)
					}
				default:
					if strings.HasPrefix(args[i], "-") {
						t.Fatalf("unsupported upstream option: %s", line)
					}
					patterns = append(patterns, args[i])
				}
			}
			native = programOutcome(t, dir, "go", args...)
			var output strings.Builder
			code, err := driver.Test(driver.Options{Dir: dir, Patterns: patterns, Out: out, Runtime: runtimeDir(t)}, options, &output)
			if err != nil {
				t.Fatal(err)
			}
			gart = programResult{Text: output.String(), Exit: code}
			if native.Exit == 0 || gart.Exit == 0 {
				t.Fatalf("command must fail: %s\nGo: %+v\nGart: %+v", line, native, gart)
			}
			if commands == 2 {
				result := programOutcome(t, dir, cli, "test", "-o", t.TempDir(), "./failfast_test.go", "-run="+options.Run, "-failfast")
				cliResult = &result
			}
			continue
		}
		if strings.HasPrefix(line, "stdout ") {
			assertions++
			match := assertion.FindStringSubmatch(line)
			if match == nil || commands == 0 {
				t.Fatalf("unsupported upstream assertion: %s", line)
			}
			want, _ := strconv.Atoi(match[1])
			pattern := regexp.MustCompile(match[2])
			t.Run(fmt.Sprintf("command%d-assertion%d", commands, assertions), func(t *testing.T) {
				results := map[string]programResult{"Go": native, "Gart": gart}
				if cliResult != nil {
					results["Gart CLI"] = *cliResult
				}
				for name, result := range results {
					if result.Exit == 0 {
						t.Errorf("%s accepted a failed test command", name)
					}
					if got := len(pattern.FindAllStringIndex(result.Text, -1)); got != want {
						t.Errorf("%s violates %s: got %d\n%s", name, line, got, result.Text)
					}
				}
			})
		}
	}
	if commands != 20 || assertions != 21 {
		t.Fatalf("upstream script changed: %d commands and %d assertions", commands, assertions)
	}
}
