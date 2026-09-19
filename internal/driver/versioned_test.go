package driver_test

import (
	"crypto/sha256"
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

	"github.com/eddyw/godart/internal/driver"
)

type versionedFixture struct {
	Source   string   `json:"source"`
	SHA256   string   `json:"sha256"`
	Versions []string `json:"module_versions"`
}

func TestVersionedGoFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/versioned-go-tests.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []versionedFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, fixture := range fixtures {
		names = append(names, filepath.Base(fixture.Source))
	}
	selected := testShard(t, names)
	for _, fixture := range fixtures {
		name := filepath.Base(fixture.Source)
		if !selected[name] {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source, err := os.ReadFile(filepath.Join(build.Default.GOROOT, fixture.Source))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(source)); got != fixture.SHA256 {
				t.Fatalf("upstream source changed: %s sha256=%s; review the fixture before the SDK update", fixture.Source, got)
			}
			t.Logf("source=%s sha256=%s", fixture.Source, fixture.SHA256)
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, name), string(source))
			out := filepath.Join(dir, "out")
			// Reuse the output to cover cache invalidation after a module version change.
			for _, version := range fixture.Versions {
				t.Run(version, func(t *testing.T) {
					writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/versioned\n\ngo "+version+"\n")
					native := filepath.Join(dir, "native")
					text, err, ended := output(dir, nil, runLimit, "go", "build", "-o", native, ".")
					if err != nil || !ended {
						t.Fatalf("native build: %v\n%s", err, text)
					}
					want, wantExit := versionedOutput(t, dir, native)
					stats, err := driver.BuildWithStats(driver.Options{Dir: dir, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Format: false})
					if err != nil {
						t.Fatal(err)
					}
					if !slices.Contains(stats.Compiled, "example.com/versioned") {
						t.Fatal("module version change reused the package cache")
					}
					prepareDart(t, out)
					got, gotExit := versionedOutput(t, out, "dart", "--packages=.dart_tool/package_config.json", "bin/main.dart")
					if got != want || gotExit != wantExit {
						t.Errorf("module Go %s\nGo (exit %d):\n%s\nGart (exit %d):\n%s", version, wantExit, want, gotExit, got)
					}
				})
			}
		})
	}
}

func versionedOutput(t *testing.T, dir, command string, args ...string) (string, int) {
	t.Helper()
	text, err, ended := output(dir, nil, runLimit, command, args...)
	if !ended {
		t.Fatalf("%s exceeded %s", command, runLimit)
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	return strings.ReplaceAll(string(text), "\r\n", "\n"), code
}
