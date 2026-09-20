// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in internal/overlay/std/LICENSE.

package parity

import (
	"embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

//go:embed testdata/language-skips.json testdata/language-rejections.json
var languagePolicy embed.FS

type LanguagePolicy struct {
	Skips      map[string]Omission `json:"skips"`
	Rejections map[string][]string `json:"rejections"`
}

func ReadLanguagePolicy() (*LanguagePolicy, error) {
	policy := &LanguagePolicy{}
	for path, dest := range map[string]any{"language-skips.json": &policy.Skips, "language-rejections.json": &policy.Rejections} {
		data, err := languagePolicy.ReadFile("testdata/" + path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, dest); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

type LanguageCase struct {
	Exclusion      *Omission    `json:"exclusion,omitempty"`
	Rejection      []string     `json:"rejection,omitempty"`
	Name           string       `json:"name"`
	File           string       `json:"file"`
	Recipe         string       `json:"recipe"`
	Action         string       `json:"action"`
	NativeEligible bool         `json:"native_eligible"`
	GartEligible   bool         `json:"gart_eligible"`
	Adapter        string       `json:"adapter"`
	Files          []SourceFile `json:"files"`
}

type LanguageInventory struct {
	GCFlags      string         `json:"go_gcflags"`
	GoVersion    string         `json:"go_version"`
	GOOS         string         `json:"goos"`
	GOARCH       string         `json:"goarch"`
	BuildTags    []string       `json:"build_tags"`
	ToolTags     []string       `json:"tool_tags"`
	Cgo          bool           `json:"cgo"`
	Runner       SourceFile     `json:"runner"`
	SourceSHA256 string         `json:"source_sha256"`
	Cases        []LanguageCase `json:"cases"`
}

// DiscoverLanguage inventories the recipes selected by the active Go test runner.
// An available adapter records no successful execution.
func DiscoverLanguage(ctx build.Context) (*LanguageInventory, error) {
	policy, err := ReadLanguagePolicy()
	if err != nil {
		return nil, err
	}
	const runner = "src/cmd/internal/testdir/testdir_test.go"
	source, err := os.ReadFile(filepath.Join(ctx.GOROOT, runner))
	if err != nil {
		return nil, err
	}
	dirs, err := languageDirectories(source)
	if err != nil {
		return nil, err
	}
	version, err := os.ReadFile(filepath.Join(ctx.GOROOT, "VERSION"))
	if err != nil {
		return nil, err
	}
	inv := &LanguageInventory{GoVersion: strings.Split(string(version), "\n")[0], GCFlags: os.Getenv("GO_GCFLAGS"), GOOS: ctx.GOOS, GOARCH: ctx.GOARCH, BuildTags: slices.Clone(ctx.BuildTags), ToolTags: slices.Clone(ctx.ToolTags), Cgo: ctx.CgoEnabled, Runner: SourceFile{Path: runner, SHA256: digest(source)}}
	gart := ctx
	gart.BuildTags = append(slices.Clone(ctx.BuildTags), "dart")
	root := filepath.Join(ctx.GOROOT, "test")
	identity := runner + "\x00" + inv.Runner.SHA256 + "\n"
	for _, dir := range dirs {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			file := filepath.ToSlash(filepath.Join(dir, entry.Name()))
			data, err := os.ReadFile(filepath.Join(root, file))
			if err != nil {
				return nil, err
			}
			recipe := languageRecipe(string(data))
			fields := strings.Fields(recipe)
			if len(fields) == 0 {
				return nil, fmt.Errorf("missing upstream language recipe: %s", file)
			}
			native := languageEligible(string(data), ctx, inv.GCFlags)
			eligible, err := gart.MatchFile(filepath.Join(root, dir), entry.Name())
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			c := LanguageCase{Name: strings.TrimSuffix(file, ".go"), File: file, Recipe: recipe, Action: fields[0], NativeEligible: native, GartEligible: eligible, Adapter: "missing", Files: []SourceFile{{Path: file, SHA256: digest(data)}}}
			if recipe == "run" {
				c.Adapter = "single-file"
			}
			for _, suffix := range []string{".dir", ".out"} {
				companion := filepath.Join(root, c.Name+suffix)
				if _, err := os.Stat(companion); os.IsNotExist(err) {
					continue
				} else if err != nil {
					return nil, err
				}
				err := filepath.WalkDir(companion, func(path string, d fs.DirEntry, err error) error {
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
					relative, err := filepath.Rel(root, path)
					if err != nil {
						return err
					}
					c.Files = append(c.Files, SourceFile{Path: filepath.ToSlash(relative), SHA256: digest(data)})
					return nil
				})
				if err != nil {
					return nil, err
				}
			}
			if skip, ok := policy.Skips[c.Name]; ok {
				c.Exclusion = &skip
			}
			c.Rejection = policy.Rejections[c.Name]
			inv.Cases = append(inv.Cases, c)
		}
	}
	slices.SortFunc(inv.Cases, func(a, b LanguageCase) int { return strings.Compare(a.Name, b.Name) })
	for _, c := range inv.Cases {
		for _, f := range c.Files {
			identity += f.Path + "\x00" + f.SHA256 + "\n"
		}
	}
	inv.SourceSHA256 = digest([]byte(identity))
	return inv, nil
}

func languageRecipe(source string) string {
	for line := range strings.SplitSeq(source, "\n") {
		if constraint.IsGoBuild(line) || constraint.IsPlusBuild(line) {
			continue
		}
		if recipe := strings.TrimSpace(strings.TrimPrefix(line, "//")); recipe != "" {
			return recipe
		}
	}
	return ""
}

func languageDirectories(source []byte) ([]string, error) {
	tree, err := parser.ParseFile(token.NewFileSet(), "testdir_test.go", source, 0)
	if err != nil {
		return nil, err
	}
	for _, decl := range tree.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != "dirs" || len(value.Values) != 1 {
				continue
			}
			list, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				break
			}
			var dirs []string
			for _, item := range list.Elts {
				literal, ok := item.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return nil, fmt.Errorf("nonliteral upstream test directory")
				}
				dir, err := strconv.Unquote(literal.Value)
				if err != nil {
					return nil, err
				}
				dirs = append(dirs, dir)
			}
			return dirs, nil
		}
	}
	return nil, fmt.Errorf("upstream language runner has no literal directory list")
}

// The upstream testdir runner ignores filename suffixes and evaluates all
// header constraints with its own tags, including test_run.
func languageEligible(source string, ctx build.Context, gcflags string) bool {
	match := func(tag string) bool {
		return tag == ctx.GOOS || tag == ctx.GOARCH || tag == "gc" || tag == "test_run" ||
			tag == "cgo" && ctx.CgoEnabled ||
			tag == "gcflags_noopt" && (strings.Contains(gcflags, "-N") || strings.Contains(gcflags, "-l")) ||
			slices.Contains(ctx.ReleaseTags, tag) || strings.HasPrefix(tag, "goexperiment.") && slices.Contains(ctx.ToolTags, tag)
	}
	for line := range strings.SplitSeq(source, "\n") {
		if strings.HasPrefix(line, "package ") {
			break
		}
		if expr, err := constraint.Parse(line); err == nil && !expr.Eval(match) {
			return false
		}
	}
	return true
}
