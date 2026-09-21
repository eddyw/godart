package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestUpstreamCompileBlockers(t *testing.T) {
	data, err := os.ReadFile("../parity/testdata/coverage-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var coverage []parity.Coverage
	if err := json.Unmarshal(data, &coverage); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range coverage {
		if c.Status == "compile-blocked" || c.Status == "runtime-blocked" {
			names = append(names, c.Package)
		}
	}
	selected := testShard(t, names)
	position := regexp.MustCompile(`^.*?\.go:\d+:\d+: `)
	for _, c := range coverage {
		if !selected[c.Package] {
			continue
		}
		t.Run(c.Package, func(t *testing.T) {
			t.Parallel()
			_, err := driver.BuildWithStats(driver.Options{Dir: filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(c.Package)), Patterns: []string{"."}, Out: t.TempDir(), Runtime: runtimeDir(t), Tests: true, TrimPath: true})
			if c.Status == "runtime-blocked" {
				if err != nil {
					t.Fatalf("runtime-blocked package no longer compiles: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("package now compiles: classify and execute its upstream tests")
			}
			first, _, _ := strings.Cut(err.Error(), "\n")
			got := position.ReplaceAllString(first, "")
			want := c.Diagnostic
			if diagnostic, ok := c.PlatformDiagnostics[runtime.GOOS+"-"+runtime.GOARCH]; ok {
				want = diagnostic
			}
			if got != want {
				t.Fatalf("unclassified compile rejection:\nwant: %s\ngot: %s", want, got)
			}
		})
	}
}
