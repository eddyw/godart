package driver_test

import (
	"archive/zip"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/eddyw/godart/internal/driver"
	"github.com/eddyw/godart/internal/parity"
)

func TestSelectedUpstream(t *testing.T) {
	var inventory parity.Inventory
	data, err := os.ReadFile("../parity/testdata/" + runtime.GOOS + "-" + runtime.GOARCH + ".json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	for pkg, file := range map[string]string{
		"sync": "once_test.go", "sync/atomic": "value_test.go",
		"reflect": "visiblefields_test.go",
		"time":    "example_test.go", "os": "env_test.go", "net": "ip_test.go",
	} {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkg))
			source, err := os.ReadFile(filepath.Join(dir, file))
			if err != nil {
				t.Fatal(err)
			}
			fixture := t.TempDir()
			writeFile(t, filepath.Join(fixture, "go.mod"), "module example.com/upstream\n\ngo 1.27\n")
			if pkg == "net" {
				fs := token.NewFileSet()
				f, err := parser.ParseFile(fs, file, source, 0)
				if err != nil {
					t.Fatal(err)
				}
				for i := len(f.Decls) - 1; i >= 0; i-- {
					if fn, ok := f.Decls[i].(*ast.FuncDecl); ok && (strings.HasPrefix(fn.Name.Name, "Benchmark") || fn.Name.Name == "TestNetworkNumberAndMask") {
						start, end := fs.Position(fn.Pos()).Offset, fs.Position(fn.End()).Offset
						source = append(source[:start], source[end:]...)
					}
				}
				source = []byte(strings.Replace(string(source), "package net", "package net_test\nimport . \"net\"", 1))
			}
			writeFile(t, filepath.Join(fixture, file), string(source))
			if pkg == "time" {
				zones, err := zip.OpenReader(filepath.Join(runtime.GOROOT(), "lib/time/zoneinfo.zip"))
				if err != nil {
					t.Fatal(err)
				}
				defer zones.Close()
				for _, zone := range zones.File {
					if zone.Name != "America/Los_Angeles" {
						continue
					}
					r, err := zone.Open()
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(r)
					r.Close()
					if err != nil {
						t.Fatal(err)
					}
					writeFile(t, filepath.Join(fixture, "zone_test.go"), "package time_test\nimport \"time\"\nfunc init() { var err error; time.Local, err = time.LoadLocationFromTZData(\"Local\", []byte("+strconv.Quote(string(data))+")); if err != nil { panic(err) } }\n")
				}
			}

			var names []string
			for _, p := range inventory.Packages {
				if p.Path != pkg {
					continue
				}
				for _, test := range p.Tests {
					if test.File == file && test.Eligible && test.Kind != "Benchmark" {
						names = append(names, regexp.QuoteMeta(test.Name))
					}
				}
			}
			if len(names) == 0 {
				t.Fatal("empty upstream file selection")
			}
			filter := "^(" + strings.Join(names, "|") + ")$"
			out := t.TempDir()
			stats, err := driver.BuildWithStats(driver.Options{Dir: fixture, Patterns: []string{"."}, Out: out, Runtime: runtimeDir(t), Tests: true, TrimPath: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Binaries) != 1 {
				t.Fatalf("got %d binaries", len(stats.Binaries))
			}
			prepareDart(t, out)
			result := programOutcome(t, dir, "dart", filepath.Join(out, stats.Binaries[0].Entry), "-test.v", "-test.short="+strconv.FormatBool(testing.Short()), "-test.run="+filter, "-test.skip="+upstreamSkip(t, pkg))
			compareUpstream(t, pkg+"-selected", pkg, filter, "vm", dir, result.Text, result.Exit)
		})
	}
}
