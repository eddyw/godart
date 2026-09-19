package parity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/eddyw/godart/internal/overlay"
)

type SourceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Test struct {
	Name     string `json:"name"`
	File     string `json:"file"`
	Kind     string `json:"kind"`
	Eligible bool   `json:"eligible"`
}

type Package struct {
	Path        string       `json:"path"`
	Tests       []Test       `json:"tests"`
	Files       []SourceFile `json:"files"`
	Seeds       []SourceFile `json:"seeds"`
	TestPatches []SourceFile `json:"test_patches,omitempty"`
}

type Inventory struct {
	GoVersion    string       `json:"go_version"`
	GOOS         string       `json:"goos"`
	GOARCH       string       `json:"goarch"`
	BuildTags    []string     `json:"build_tags"`
	ToolTags     []string     `json:"tool_tags"`
	Cgo          bool         `json:"cgo"`
	SourceSHA256 string       `json:"source_sha256"`
	Fixtures     []SourceFile `json:"fixtures"`
	Packages     []Package    `json:"packages"`
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// InventoryName separates source fingerprints by SDK version and target.
func InventoryName(ctx build.Context) (string, error) {
	data, err := os.ReadFile(filepath.Join(ctx.GOROOT, "VERSION"))
	if err != nil {
		return "", err
	}
	version := strings.Split(string(data), "\n")[0]
	return filepath.Join(version, ctx.GOOS+"-"+ctx.GOARCH+".json"), nil
}

// Discover reads source before compilation, including tests in ineligible files.
func Discover(ctx build.Context) (*Inventory, error) {
	version, err := os.ReadFile(filepath.Join(ctx.GOROOT, "VERSION"))
	if err != nil {
		return nil, err
	}
	inv := &Inventory{GoVersion: strings.Split(string(version), "\n")[0], GOOS: ctx.GOOS, GOARCH: ctx.GOARCH, BuildTags: ctx.BuildTags, ToolTags: ctx.ToolTags, Cgo: ctx.CgoEnabled}
	h := sha256.New()
	root := filepath.Join(ctx.GOROOT, "src")
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if d.Name() == "_go.mod" {
			original, err := os.ReadFile(filepath.Join(filepath.Dir(path), "go.mod"))
			if err == nil && bytes.Equal(data, original) {
				return nil
			}
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%s\n", filepath.ToSlash(rel), digest(data))
		return nil
	})
	if err != nil {
		return nil, err
	}
	zone, err := os.ReadFile(filepath.Join(ctx.GOROOT, "lib/time/zoneinfo.zip"))
	if err != nil {
		return nil, err
	}
	inv.Fixtures = []SourceFile{{"lib/time/zoneinfo.zip", digest(zone)}}
	inv.SourceSHA256 = hex.EncodeToString(h.Sum(nil))
	patches := map[string][]SourceFile{}
	for _, p := range overlay.Packages() {
		for _, f := range append(p.Files, p.Patches...) {
			if strings.HasSuffix(strings.TrimSuffix(f.Name, ".patch"), "_test.go") {
				patches[p.Path] = append(patches[p.Path], SourceFile{f.Name, digest(f.Content)})
			}
		}
	}
	paths := overlay.Surface()
	slices.Sort(paths)
	paths = slices.Compact(paths)
	for _, path := range paths {
		dir := filepath.Join(root, filepath.FromSlash(path))
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		p := Package{Path: path, TestPatches: patches[path]}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			p.Files = append(p.Files, SourceFile{name, digest(data)})
			if !strings.HasSuffix(name, "_test.go") {
				continue
			}
			eligible, err := ctx.MatchFile(dir, name)
			if err != nil {
				return nil, err
			}
			f, err := parser.ParseFile(token.NewFileSet(), name, data, parser.ParseComments)
			if err != nil {
				return nil, err
			}
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				n := fn.Name.Name
				for _, kind := range []string{"Test", "Fuzz", "Benchmark"} {
					if testName(n, kind) && n != "TestMain" {
						p.Tests = append(p.Tests, Test{n, name, kind, eligible})
					}
				}
			}
			for _, ex := range doc.Examples(f) {
				if ex.Output != "" || ex.EmptyOutput {
					p.Tests = append(p.Tests, Test{"Example" + ex.Name, name, "Example", eligible})
				}
			}
		}
		seeds := filepath.Join(dir, "testdata", "fuzz")
		if _, err := os.Stat(seeds); err == nil {
			err = filepath.WalkDir(seeds, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				p.Seeds = append(p.Seeds, SourceFile{filepath.ToSlash(rel), digest(data)})
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		slices.SortFunc(p.Tests, func(a, b Test) int { return strings.Compare(a.File+"/"+a.Name, b.File+"/"+b.Name) })
		inv.Packages = append(inv.Packages, p)
	}
	return inv, nil
}

func testName(name, prefix string) bool {
	rest, ok := strings.CutPrefix(name, prefix)
	if !ok {
		return false
	}
	for _, r := range rest {
		return !unicode.IsLower(r)
	}
	return true
}
